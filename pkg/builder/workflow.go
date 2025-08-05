package builder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0x00fafa/gke-image-cache-builder/internal/disk"
	"github.com/0x00fafa/gke-image-cache-builder/internal/image"
	"github.com/0x00fafa/gke-image-cache-builder/internal/vm"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Workflow manages the step-by-step execution of image cache building
type Workflow struct {
	config      *config.Config
	logger      *log.Logger
	vmManager   *vm.Manager
	diskManager *disk.Manager
	imageCache  *image.Cache
	gcpClient   *gcp.Client
}

// NewWorkflow creates a new workflow instance
func NewWorkflow(cfg *config.Config, logger *log.Logger, vmMgr *vm.Manager, diskMgr *disk.Manager, imgCache *image.Cache, gcpClient *gcp.Client) *Workflow {
	return &Workflow{
		config:      cfg,
		logger:      logger,
		vmManager:   vmMgr,
		diskManager: diskMgr,
		imageCache:  imgCache,
		gcpClient:   gcpClient,
	}
}

// Execute runs the complete workflow
func (w *Workflow) Execute(ctx context.Context) error {
	// Step 1: Validate prerequisites
	if err := w.validatePrerequisites(ctx); err != nil {
		w.scheduleCleanup(ctx, nil, 5*time.Minute)
		return fmt.Errorf("prerequisite validation failed: %w", err)
	}

	// Step 2: Check existing images (local mode only)
	if w.config.IsLocalMode() {
		if err := w.handleExistingImages(ctx); err != nil {
			w.scheduleCleanup(ctx, nil, 5*time.Minute)
			return fmt.Errorf("existing images handling failed: %w", err)
		}
	}

	// Step 3: Setup execution environment
	resources, err := w.setupEnvironment(ctx)
	if err != nil {
		w.scheduleCleanup(ctx, resources, 5*time.Minute)
		return fmt.Errorf("environment setup failed: %w", err)
	}

	// Step 4: Execute image processing based on mode
	if w.config.IsLocalMode() {
		if err := w.executeLocalMode(ctx, resources); err != nil {
			w.scheduleCleanup(ctx, resources, 5*time.Minute)
			return fmt.Errorf("local mode execution failed: %w", err)
		}
	} else {
		if err := w.executeRemoteMode(ctx, resources); err != nil {
			w.scheduleCleanup(ctx, resources, 5*time.Minute)
			return fmt.Errorf("remote mode execution failed: %w", err)
		}
	}

	// Step 5: Create cache disk image
	if err := w.createCacheImage(ctx, resources); err != nil {
		w.scheduleCleanup(ctx, resources, 5*time.Minute)
		return fmt.Errorf("cache image creation failed: %w", err)
	}

	// Step 6: Verify cache image
	if err := w.verifyCacheImage(ctx); err != nil {
		w.scheduleCleanup(ctx, resources, 5*time.Minute)
		return fmt.Errorf("cache image verification failed: %w", err)
	}

	// Step 7: Cleanup resources on success after 5 minutes
	w.scheduleCleanup(ctx, resources, 5*time.Minute)

	return nil
}

func (w *Workflow) validatePrerequisites(ctx context.Context) error {
	w.logger.Info("🔍 Validating prerequisites...")

	// Validate GCP permissions
	w.logger.Info("🔐 Checking GCP permissions...")
	if err := w.vmManager.ValidatePermissions(ctx, w.config.ProjectName, w.config.Zone); err != nil {
		return fmt.Errorf("GCP permissions validation failed: %w", err)
	}
	w.logger.Success("🔐 GCP permissions validated successfully")

	// Additional checks for local mode
	if w.config.IsLocalMode() {
		w.logger.Info("🏠 Checking local mode permissions...")
		if err := w.diskManager.CheckLocalModePermissions(ctx); err != nil {
			return fmt.Errorf("local mode permissions check failed: %w", err)
		}
		w.logger.Success("🏠 Local mode permissions validated")
	}

	// Validate container image accessibility
	w.logger.Info("🐳 Validating container image accessibility...")
	for i, img := range w.config.ContainerImages {
		w.logger.Progress(i+1, len(w.config.ContainerImages), fmt.Sprintf("Validating image: %s", img))
		if err := w.imageCache.ValidateImageAccess(ctx, img); err != nil {
			return fmt.Errorf("image access validation failed for %s: %w", img, err)
		}
	}
	w.logger.Success("🐳 All container images validated successfully")

	w.logger.Success("✅ Prerequisites validated successfully")
	return nil
}

func (w *Workflow) handleExistingImages(ctx context.Context) error {
	w.logger.Info("🔍 Checking for existing images...")

	existingInfo, err := w.imageCache.CheckExistingImages(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing images: %w", err)
	}

	if len(existingInfo.Images) > 0 {
		switch existingInfo.Action {
		case image.ActionClean:
			w.logger.Info("🧹 Cleaning existing images...")
			if err := w.imageCache.CleanExistingImages(ctx, existingInfo.Images); err != nil {
				return fmt.Errorf("failed to clean existing images: %w", err)
			}
			w.logger.Success("🧹 Existing images cleaned successfully")
		case image.ActionCancel:
			w.logger.Warn("🚫 Operation cancelled by user")
			return fmt.Errorf("operation cancelled by user")
		case image.ActionMerge:
			w.logger.Info("🔄 Continuing with existing images (merge mode)")
		}
	} else {
		w.logger.Info("✅ No existing images found")
	}

	return nil
}

func (w *Workflow) setupEnvironment(ctx context.Context) (*WorkflowResources, error) {
	w.logger.Info("🏗️ Setting up execution environment...")
	resources := &WorkflowResources{}

	// Create cache disk
	w.logger.Info("💾 Creating cache disk...")
	diskConfig := &disk.Config{
		Name:   fmt.Sprintf("%s-disk", w.config.DiskImageName),
		Zone:   w.config.Zone,
		SizeGB: w.config.DiskSizeGB,
		Type:   w.config.DiskType,
	}

	cacheDisk, err := w.diskManager.CreateDisk(ctx, diskConfig)
	if err != nil {
		w.logger.Error("❌ Failed to create cache disk")
		return nil, fmt.Errorf("failed to create cache disk: %w", err)
	}
	resources.CacheDisk = cacheDisk
	w.logger.Successf("💾 Created cache disk: %s", cacheDisk.Name)

	if w.config.IsRemoteMode() {
		w.logger.Info("☁️ Setting up remote mode environment...")
		// Create temporary VM
		vmConfig := &vm.Config{
			Name:            fmt.Sprintf("cache-builder-%s", w.config.JobName),
			Zone:            w.config.Zone,
			MachineType:     w.config.MachineType,
			Network:         w.config.Network,
			Subnet:          w.config.Subnet,
			ServiceAccount:  w.config.ServiceAccount,
			Preemptible:     w.config.Preemptible,
			ContainerImages: w.config.ContainerImages,
			ImagePullAuth:   w.config.ImagePullAuth,
			SSHPublicKey:    w.config.SSHPublicKey,
		}

		w.logger.Info("🖥️ Creating temporary VM...")
		vmInstance, err := w.vmManager.CreateVM(ctx, vmConfig)
		if err != nil {
			w.logger.Error("❌ Failed to create temporary VM")
			return nil, fmt.Errorf("failed to create VM: %w", err)
		}
		resources.VMInstance = vmInstance
		w.logger.Successf("🖥️ Created temporary VM: %s", vmInstance.Name)

		// Attach disk to remote VM
		w.logger.Info("🔗 Attaching disk to remote VM...")
		if err := w.diskManager.AttachDisk(ctx, cacheDisk.Name, vmInstance.Name, w.config.Zone); err != nil {
			w.logger.Error("❌ Failed to attach disk to VM")
			return nil, fmt.Errorf("failed to attach disk to VM: %w", err)
		}
		w.logger.Success("🔗 Disk attached to remote VM successfully")
		w.logger.Info("☁️ Remote mode environment setup completed")
	} else {
		w.logger.Info("🏠 Setting up local mode environment...")
		// Local mode: attach disk to current instance
		// Get current instance metadata
		w.logger.Info("📍 Getting current instance metadata...")
		instanceMetadata, err := w.gcpClient.GetCurrentInstanceMetadata(ctx)
		if err != nil {
			w.logger.Error("❌ Failed to get current instance metadata")
			return nil, fmt.Errorf("failed to get current instance metadata: %w", err)
		}

		// Attach disk to current instance
		w.logger.Info("🔗 Attaching disk to current instance...")
		if err := w.diskManager.AttachDisk(ctx, cacheDisk.Name, instanceMetadata.Name, w.config.Zone); err != nil {
			w.logger.Error("❌ Failed to attach disk to current instance")
			return nil, fmt.Errorf("failed to attach disk to current instance: %w", err)
		}

		w.logger.Successf("🔗 Disk attached to current instance: %s", instanceMetadata.Name)
		w.logger.Info("🏠 Local mode environment setup completed")
	}

	w.logger.Success("✅ Environment setup completed successfully")
	return resources, nil
}

func (w *Workflow) executeLocalMode(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Info("🏠 Executing local mode image processing...")

	// Get current instance metadata
	w.logger.Info("📍 Getting current instance metadata...")
	instanceMetadata, err := w.gcpClient.GetCurrentInstanceMetadata(ctx)
	if err != nil {
		w.logger.Error("❌ Failed to get current instance metadata")
		return fmt.Errorf("failed to get current instance metadata: %w", err)
	}

	w.logger.Infof("📍 Running on instance: %s in zone: %s", instanceMetadata.Name, instanceMetadata.Zone)

	// Get device path for the attached disk
	w.logger.Info("🔍 Getting device path for attached disk...")
	devicePath, err := w.diskManager.GetAttachedDiskDevicePath(ctx, resources.CacheDisk.Name, instanceMetadata.Name, w.config.Zone)
	if err != nil {
		w.logger.Error("❌ Failed to get device path")
		return fmt.Errorf("failed to get device path: %w", err)
	}

	w.logger.Infof("🔍 Using device path: %s", devicePath)

	// Execute the integrated script workflow with device path
	w.logger.Info("🐳 Processing container images...")
	processConfig := &image.ProcessConfig{
		DeviceName:     "secondary-disk-image-disk",
		AuthMechanism:  w.config.ImagePullAuth,
		StoreChecksums: true, // Always store checksums for verification
		Images:         w.config.ContainerImages,
	}

	if err := w.imageCache.ProcessImagesWithScriptAndDevice(ctx, processConfig, devicePath); err != nil {
		w.logger.Error("❌ Local image processing failed")
		return fmt.Errorf("local image processing failed: %w", err)
	}

	// Detach disk from current instance
	w.logger.Info("🔓 Detaching disk from current instance...")
	if err := w.diskManager.DetachDisk(ctx, resources.CacheDisk.Name, instanceMetadata.Name, w.config.Zone); err != nil {
		w.logger.Warnf("⚠️ Failed to detach disk: %v", err)
		return fmt.Errorf("failed to detach disk: %w", err)
	}

	w.logger.Success("🏠 Local mode execution completed successfully")
	return nil
}

func (w *Workflow) executeRemoteMode(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Info("☁️ Executing remote mode image processing...")

	// Wait for environment to be ready on the remote VM
	w.logger.Info("⏳ Waiting for remote environment to be ready...")
	if err := w.waitForRemoteEnvironment(ctx, resources.VMInstance); err != nil {
		w.logger.Error("❌ Failed waiting for remote environment")
		return fmt.Errorf("failed waiting for remote environment: %w", err)
	}
	w.logger.Success("✅ Remote environment is ready")

	// Execute remote image processing with proper timing
	w.logger.Info("🐳 Processing container images on remote VM...")
	if err := w.executeRemoteImageProcessing(ctx, resources); err != nil {
		w.logger.Error("❌ Remote image processing failed")
		return fmt.Errorf("remote image processing failed: %w", err)
	}
	w.logger.Success("✅ Remote image processing completed successfully")

	// Detach disk from remote VM
	w.logger.Info("🔓 Detaching disk from remote VM...")
	if err := w.diskManager.DetachDisk(ctx, resources.CacheDisk.Name, resources.VMInstance.Name, w.config.Zone); err != nil {
		w.logger.Warnf("⚠️ Failed to detach disk from VM: %v", err)
		return fmt.Errorf("failed to detach disk from VM: %w", err)
	}

	w.logger.Success("☁️ Remote mode execution completed successfully")
	return nil
}

// waitForRemoteEnvironment waits for the remote environment to be ready
func (w *Workflow) waitForRemoteEnvironment(ctx context.Context, instance *vm.Instance) error {
	w.logger.Info("⏳ Waiting for remote environment to be ready...")

	timeoutCtx, cancel := context.WithTimeout(ctx, w.config.Timeout)
	defer cancel()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// First, let's wait a bit for the VM to fully boot and start executing the startup script
	w.logger.Info("⏳ Initial wait for VM to boot and start executing startup script...")
	time.Sleep(120 * time.Second) // Increased to 2 minutes

	for {
		select {
		case <-timeoutCtx.Done():
			w.logger.Error("❌ Timeout waiting for remote environment")
			// Log the final serial console output for debugging
			output, err := w.getRemoteCommandOutput(ctx, instance, "")
			if err == nil {
				w.logger.Debugf("Final serial console output: %s", getLastNCharacters(output, 2000))
			}
			return fmt.Errorf("timeout waiting for remote environment")
		case <-ticker.C:
			// Check serial console output for completion signal
			output, err := w.getRemoteCommandOutput(ctx, instance, "")
			if err != nil {
				w.logger.Debugf("⚠️ Failed to get serial console output: %v", err)
				continue
			}

			// Look for specific completion messages in the output
			if strings.Contains(output, "Environment setup completed.") && strings.Contains(output, "environment_ready.flag") {
				w.logger.Success("✅ Remote environment is ready")
				return nil
			}

			// Also check for the new completion flag
			if strings.Contains(output, "Full workflow completed successfully") {
				w.logger.Success("✅ Remote environment is ready")
				return nil
			}

			// Also check for errors
			if strings.Contains(output, "ERROR") || strings.Contains(output, "Failed") {
				w.logger.Error("❌ Remote environment setup failed")
				w.logger.Debugf("Serial console output: %s", getLastNCharacters(output, 2000))
				return fmt.Errorf("remote environment setup failed")
			}

			w.logger.Info("⏳ Remote environment is not ready yet, waiting...")
			w.logger.Debugf("Last 1000 characters of serial console output: %s", getLastNCharacters(output, 1000))
		}
	}
}

// executeRemoteImageProcessing executes the image processing on the remote VM
func (w *Workflow) executeRemoteImageProcessing(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Info("Executing remote image processing...")

	// Generate the command to execute on the remote VM
	images := "nginx:latest" // Default fallback
	if len(w.config.ContainerImages) > 0 {
		images = strings.Join(w.config.ContainerImages, " ")
	}

	command := fmt.Sprintf(
		"/tmp/setup-and-verify.sh prepare-disk secondary-disk-image-disk && "+
			"/tmp/setup-and-verify.sh pull-images %s true %s && "+
			"echo 'Unpacking is completed.'",
		w.config.ImagePullAuth,
		images,
	)

	// Execute the command on the remote VM
	output, err := w.getRemoteCommandOutput(ctx, resources.VMInstance, command)
	if err != nil {
		return fmt.Errorf("failed to execute remote image processing: %w", err)
	}

	w.logger.Debugf("Remote command output: %s", output)

	// Check if the command completed successfully
	if !strings.Contains(output, "Unpacking is completed.") {
		return fmt.Errorf("remote image processing did not complete successfully")
	}

	return nil
}

// getRemoteCommandOutput executes a command on the remote VM and returns the output
func (w *Workflow) getRemoteCommandOutput(ctx context.Context, instance *vm.Instance, command string) (string, error) {
	// For now, we'll use serial console output as a workaround
	// In a production implementation, we would use SSH or GCP's OS Login API
	output, err := w.vmManager.GetSerialConsoleOutput(ctx, instance.Name, instance.Zone)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (w *Workflow) createCacheImage(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Info("Creating cache disk image...")

	imageConfig := &disk.ImageConfig{
		Name:        w.config.DiskImageName,
		SourceDisk:  resources.CacheDisk.Name,
		Zone:        w.config.Zone,
		Family:      w.config.DiskFamilyName,
		Labels:      w.config.DiskLabels,
		Description: fmt.Sprintf("Image cache containing %d container images", len(w.config.ContainerImages)),
	}

	if err := w.diskManager.CreateImage(ctx, imageConfig); err != nil {
		return fmt.Errorf("failed to create cache image: %w", err)
	}

	w.logger.Infof("Cache image '%s' created successfully", w.config.DiskImageName)
	return nil
}

func (w *Workflow) verifyCacheImage(ctx context.Context) error {
	w.logger.Info("Verifying cache image...")

	if err := w.diskManager.VerifyImage(ctx, w.config.DiskImageName); err != nil {
		return fmt.Errorf("cache image verification failed: %w", err)
	}

	w.logger.Info("Cache image verified successfully")
	return nil
}

func (w *Workflow) cleanupResources(ctx context.Context, resources *WorkflowResources) {
	w.logger.Info("Cleaning up temporary resources...")

	// Cleanup VM first (and wait for completion)
	if resources.VMInstance != nil {
		if err := w.vmManager.DeleteVM(ctx, resources.VMInstance.Name, w.config.Zone); err != nil {
			w.logger.Warnf("Failed to cleanup VM %s: %v", resources.VMInstance.Name, err)
		} else {
			w.logger.Infof("Cleaned up VM: %s", resources.VMInstance.Name)
		}
	}

	// For local mode, ensure disk is detached before deletion
	if w.config.IsLocalMode() && resources.CacheDisk != nil {
		instanceMetadata, err := w.gcpClient.GetCurrentInstanceMetadata(ctx)
		if err == nil {
			// Try to detach disk if still attached
			w.diskManager.DetachDisk(ctx, resources.CacheDisk.Name, instanceMetadata.Name, w.config.Zone)
		}
	}

	// Cleanup disk after VM is deleted
	if resources.CacheDisk != nil {
		if err := w.diskManager.DeleteDisk(ctx, resources.CacheDisk.Name, w.config.Zone); err != nil {
			w.logger.Warnf("Failed to cleanup disk %s: %v", resources.CacheDisk.Name, err)
		} else {
			w.logger.Infof("Cleaned up disk: %s", resources.CacheDisk.Name)
		}
	}

	w.logger.Info("Resource cleanup completed")
}

// getLastNCharacters returns the last n characters of a string
func getLastNCharacters(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// scheduleCleanup schedules cleanup of resources after a delay
func (w *Workflow) scheduleCleanup(ctx context.Context, resources *WorkflowResources, delay time.Duration) {
	go func() {
		// Create a new context for cleanup that is not tied to the original context
		cleanupCtx := context.Background()

		w.logger.Infof("Scheduling cleanup in %v...", delay)
		time.Sleep(delay)

		if resources != nil {
			w.cleanupResources(cleanupCtx, resources)
		}
	}()
}

// WorkflowResources holds references to temporary resources
type WorkflowResources struct {
	VMInstance *vm.Instance
	CacheDisk  *disk.Disk
}
