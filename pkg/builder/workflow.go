package builder

import (
	"context"
	"fmt"

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
		return fmt.Errorf("prerequisite validation failed: %w", err)
	}

	// Step 2: Check existing images (local mode only)
	if w.config.IsLocalMode() {
		if err := w.handleExistingImages(ctx); err != nil {
			return fmt.Errorf("existing images handling failed: %w", err)
		}
	}

	// Step 3: Setup execution environment
	resources, err := w.setupEnvironment(ctx)
	if err != nil {
		return fmt.Errorf("environment setup failed: %w", err)
	}

	// Step 4: Execute image processing based on mode
	if w.config.IsLocalMode() {
		if err := w.executeLocalMode(ctx, resources); err != nil {
			// Cleanup resources on failure
			w.cleanupResources(ctx, resources)
			return fmt.Errorf("local mode execution failed: %w", err)
		}
	} else {
		if err := w.executeRemoteMode(ctx, resources); err != nil {
			// Cleanup resources on failure
			w.cleanupResources(ctx, resources)
			return fmt.Errorf("remote mode execution failed: %w", err)
		}
	}

	// Temporarily comment out cache disk image creation and verification for debugging
	/*
		// Step 5: Create cache disk image
		if err := w.createCacheImage(ctx, resources); err != nil {
			// Cleanup resources on failure
			w.cleanupResources(ctx, resources)
			return fmt.Errorf("cache image creation failed: %w", err)
		}

		// Step 6: Verify cache image
		if err := w.verifyCacheImage(ctx); err != nil {
			// Cleanup resources on failure
			w.cleanupResources(ctx, resources)
			return fmt.Errorf("cache image verification failed: %w", err)
		}
	*/

	// Temporarily comment out final cleanup for debugging
	/*
		// Step 7: Cleanup resources on success
		w.cleanupResources(ctx, resources)
	*/

	return nil
}

func (w *Workflow) validatePrerequisites(ctx context.Context) error {
	w.logger.Info("Validating prerequisites...")

	// Validate GCP permissions
	if err := w.vmManager.ValidatePermissions(ctx, w.config.ProjectName, w.config.Zone); err != nil {
		return fmt.Errorf("GCP permissions validation failed: %w", err)
	}

	// Additional checks for local mode
	if w.config.IsLocalMode() {
		if err := w.diskManager.CheckLocalModePermissions(ctx); err != nil {
			return fmt.Errorf("local mode permissions check failed: %w", err)
		}
	}

	// Validate container image accessibility
	for _, img := range w.config.ContainerImages {
		if err := w.imageCache.ValidateImageAccess(ctx, img); err != nil {
			return fmt.Errorf("image access validation failed for %s: %w", img, err)
		}
	}

	w.logger.Info("Prerequisites validated successfully")
	return nil
}

func (w *Workflow) handleExistingImages(ctx context.Context) error {
	w.logger.Info("Checking for existing images...")

	existingInfo, err := w.imageCache.CheckExistingImages(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing images: %w", err)
	}

	if len(existingInfo.Images) > 0 {
		switch existingInfo.Action {
		case image.ActionClean:
			if err := w.imageCache.CleanExistingImages(ctx, existingInfo.Images); err != nil {
				return fmt.Errorf("failed to clean existing images: %w", err)
			}
		case image.ActionCancel:
			return fmt.Errorf("operation cancelled by user")
		case image.ActionMerge:
			w.logger.Info("Continuing with existing images (merge mode)")
		}
	}

	return nil
}

func (w *Workflow) setupEnvironment(ctx context.Context) (*WorkflowResources, error) {
	w.logger.Info("Setting up execution environment...")
	resources := &WorkflowResources{}

	// Create cache disk
	diskConfig := &disk.Config{
		Name:   fmt.Sprintf("%s-disk", w.config.DiskImageName),
		Zone:   w.config.Zone,
		SizeGB: w.config.DiskSizeGB,
		Type:   w.config.DiskType,
	}

	cacheDisk, err := w.diskManager.CreateDisk(ctx, diskConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache disk: %w", err)
	}
	resources.CacheDisk = cacheDisk
	w.logger.Infof("Created cache disk: %s", cacheDisk.Name)

	if w.config.IsRemoteMode() {
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

		vmInstance, err := w.vmManager.CreateVM(ctx, vmConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create VM: %w", err)
		}
		resources.VMInstance = vmInstance
		w.logger.Infof("Created temporary VM: %s", vmInstance.Name)

		// Attach disk to remote VM
		if err := w.diskManager.AttachDisk(ctx, cacheDisk.Name, vmInstance.Name, w.config.Zone); err != nil {
			return nil, fmt.Errorf("failed to attach disk to VM: %w", err)
		}
	} else {
		// Local mode: attach disk to current instance
		// Get current instance metadata
		instanceMetadata, err := w.gcpClient.GetCurrentInstanceMetadata(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current instance metadata: %w", err)
		}

		if err := w.diskManager.AttachDisk(ctx, cacheDisk.Name, instanceMetadata.Name, w.config.Zone); err != nil {
			return nil, fmt.Errorf("failed to attach disk to current instance: %w", err)
		}

		w.logger.Infof("Attached disk to current instance: %s", instanceMetadata.Name)
	}

	w.logger.Info("Environment setup completed")
	return resources, nil
}

func (w *Workflow) executeLocalMode(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Info("Executing local mode image processing...")

	// Get current instance metadata
	instanceMetadata, err := w.gcpClient.GetCurrentInstanceMetadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current instance metadata: %w", err)
	}

	w.logger.Infof("Running on instance: %s in zone: %s", instanceMetadata.Name, instanceMetadata.Zone)

	// Get device path for the attached disk
	devicePath, err := w.diskManager.GetAttachedDiskDevicePath(ctx, resources.CacheDisk.Name, instanceMetadata.Name, w.config.Zone)
	if err != nil {
		return fmt.Errorf("failed to get device path: %w", err)
	}

	w.logger.Infof("Using device path: %s", devicePath)

	// Execute the integrated script workflow with device path
	processConfig := &image.ProcessConfig{
		DeviceName:     "secondary-disk-image-disk",
		AuthMechanism:  w.config.ImagePullAuth,
		StoreChecksums: true, // Always store checksums for verification
		Images:         w.config.ContainerImages,
	}

	if err := w.imageCache.ProcessImagesWithScriptAndDevice(ctx, processConfig, devicePath); err != nil {
		return fmt.Errorf("local image processing failed: %w", err)
	}

	// Detach disk from current instance
	if err := w.diskManager.DetachDisk(ctx, resources.CacheDisk.Name, instanceMetadata.Name, w.config.Zone); err != nil {
		return fmt.Errorf("failed to detach disk: %w", err)
	}

	w.logger.Success("Local mode execution completed")
	return nil
}

func (w *Workflow) executeRemoteMode(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Info("Executing remote mode image processing...")

	// Get device path for the attached disk
	devicePath, err := w.diskManager.GetAttachedDiskDevicePath(ctx, resources.CacheDisk.Name, resources.VMInstance.Name, w.config.Zone)
	if err != nil {
		return fmt.Errorf("failed to get device path: %w", err)
	}

	w.logger.Infof("Using device path: %s", devicePath)

	// Execute remote build on the temporary VM
	remoteBuildConfig := &vm.RemoteBuildConfig{
		DeviceName:      "secondary-disk-image-disk",
		AuthMechanism:   w.config.ImagePullAuth,
		StoreChecksums:  true,
		ContainerImages: w.config.ContainerImages,
		Timeout:         w.config.Timeout,
	}

	if err := w.vmManager.ExecuteRemoteImageBuild(ctx, resources.VMInstance, remoteBuildConfig); err != nil {
		return fmt.Errorf("remote image build failed: %w", err)
	}

	// Temporarily comment out disk detachment and subsequent steps for debugging
	/*
		// Detach disk from remote VM
		if err := w.diskManager.DetachDisk(ctx, resources.CacheDisk.Name, resources.VMInstance.Name, w.config.Zone); err != nil {
			return fmt.Errorf("failed to detach disk from VM: %w", err)
		}
	*/

	w.logger.Success("Remote mode execution completed")
	return nil
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

	// Temporarily comment out VM cleanup for debugging
	/*
		// Cleanup VM first (and wait for completion)
		if resources.VMInstance != nil {
			if err := w.vmManager.DeleteVM(ctx, resources.VMInstance.Name, w.config.Zone); err != nil {
				w.logger.Warnf("Failed to cleanup VM %s: %v", resources.VMInstance.Name, err)
			} else {
				w.logger.Infof("Cleaned up VM: %s", resources.VMInstance.Name)
			}
		}
	*/

	// For local mode, ensure disk is detached before deletion
	if w.config.IsLocalMode() && resources.CacheDisk != nil {
		instanceMetadata, err := w.gcpClient.GetCurrentInstanceMetadata(ctx)
		if err == nil {
			// Try to detach disk if still attached
			w.diskManager.DetachDisk(ctx, resources.CacheDisk.Name, instanceMetadata.Name, w.config.Zone)
		}
	}

	// Temporarily comment out disk cleanup for debugging
	/*
		// Cleanup disk after VM is deleted
		if resources.CacheDisk != nil {
			if err := w.diskManager.DeleteDisk(ctx, resources.CacheDisk.Name, w.config.Zone); err != nil {
				w.logger.Warnf("Failed to cleanup disk %s: %v", resources.CacheDisk.Name, err)
			} else {
				w.logger.Infof("Cleaned up disk: %s", resources.CacheDisk.Name)
			}
		}
	*/

	w.logger.Info("Resource cleanup completed")
}

// WorkflowResources holds references to temporary resources
type WorkflowResources struct {
	VMInstance *vm.Instance
	CacheDisk  *disk.Disk
}
