package builder

import (
	"context"
	"fmt"
	"sync"

	"github.com/0x00fafa/gke-image-cache-builder/internal/disk"
	"github.com/0x00fafa/gke-image-cache-builder/internal/image"
	"github.com/0x00fafa/gke-image-cache-builder/internal/vm"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Workflow manages the step-by-step execution of image cache building
type Workflow struct {
	config      *config.Config
	logger      *log.Logger
	vmManager   *vm.Manager
	diskManager *disk.Manager
	imageCache  *image.Cache
}

// NewWorkflow creates a new workflow instance
func NewWorkflow(cfg *config.Config, logger *log.Logger, vmMgr *vm.Manager, diskMgr *disk.Manager, imgCache *image.Cache) *Workflow {
	return &Workflow{
		config:      cfg,
		logger:      logger,
		vmManager:   vmMgr,
		diskManager: diskMgr,
		imageCache:  imgCache,
	}
}

// Execute runs the complete workflow
func (w *Workflow) Execute(ctx context.Context) error {
	// Step 1: Validate prerequisites
	if err := w.validatePrerequisites(ctx); err != nil {
		return fmt.Errorf("prerequisite validation failed: %w", err)
	}

	// Step 2: Setup execution environment
	resources, err := w.setupEnvironment(ctx)
	if err != nil {
		return fmt.Errorf("environment setup failed: %w", err)
	}
	defer w.cleanupResources(ctx, resources)

	// Step 3: Setup VM if in remote mode
	if w.config.IsRemoteMode() && resources.VMInstance != nil {
		if err := w.vmManager.SetupVM(ctx, resources.VMInstance); err != nil {
			return fmt.Errorf("VM setup failed: %w", err)
		}
	}

	// Step 4: Process container images
	if err := w.processContainerImages(ctx, resources); err != nil {
		return fmt.Errorf("image processing failed: %w", err)
	}

	// Step 5: Create cache disk image
	if err := w.createCacheImage(ctx, resources); err != nil {
		return fmt.Errorf("cache image creation failed: %w", err)
	}

	// Step 6: Verify cache image
	if err := w.verifyCacheImage(ctx); err != nil {
		return fmt.Errorf("cache image verification failed: %w", err)
	}

	return nil
}

func (w *Workflow) validatePrerequisites(ctx context.Context) error {
	w.logger.Info("Validating prerequisites...")

	// Validate GCP permissions
	if err := w.vmManager.ValidatePermissions(ctx, w.config.ProjectName, w.config.Zone); err != nil {
		return fmt.Errorf("GCP permissions validation failed: %w", err)
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

func (w *Workflow) setupEnvironment(ctx context.Context) (*WorkflowResources, error) {
	w.logger.Info("Setting up execution environment...")

	resources := &WorkflowResources{}

	if w.config.IsRemoteMode() {
		// Create temporary VM
		vmConfig := &vm.Config{
			Name:           fmt.Sprintf("cache-builder-%s", w.config.JobName),
			Zone:           w.config.Zone,
			MachineType:    "e2-standard-2", // Default machine type
			Network:        w.config.Network,
			Subnet:         w.config.Subnet,
			ServiceAccount: w.config.ServiceAccount,
			Preemptible:    false, // Default to non-preemptible
		}

		vmInstance, err := w.vmManager.CreateVM(ctx, vmConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create VM: %w", err)
		}
		resources.VMInstance = vmInstance
		w.logger.Infof("Created temporary VM: %s", vmInstance.Name)
	}

	// Create cache disk
	diskName, err := w.diskManager.CreateAndAttach(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create cache disk: %w", err)
	}
	resources.CacheDiskName = diskName
	w.logger.Infof("Created cache disk: %s", diskName)

	w.logger.Info("Environment setup completed")
	return resources, nil
}

func (w *Workflow) processContainerImages(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Infof("Processing %d container images...", len(w.config.ContainerImages))

	var wg sync.WaitGroup
	errChan := make(chan error, len(w.config.ContainerImages))

	// Process images in parallel for better performance
	for i, img := range w.config.ContainerImages {
		wg.Add(1)
		go func(index int, image string) {
			defer wg.Done()
			w.logger.Progressf(index+1, len(w.config.ContainerImages), "Processing %s", image)

			cacheDisk := &disk.Disk{Name: resources.CacheDiskName}
			if err := w.imageCache.PullAndCache(ctx, image, cacheDisk); err != nil {
				errChan <- fmt.Errorf("failed to process image %s: %w", image, err)
			}
		}(i, img)
	}

	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	w.logger.Info("All container images processed successfully")
	return nil
}

func (w *Workflow) createCacheImage(ctx context.Context, resources *WorkflowResources) error {
	w.logger.Info("Creating cache disk image...")

	if err := w.diskManager.CreateImage(ctx, resources.CacheDiskName); err != nil {
		return fmt.Errorf("failed to create cache image: %w", err)
	}

	w.logger.Infof("Cache image '%s' created successfully", w.config.ImageName)
	return nil
}

func (w *Workflow) verifyCacheImage(ctx context.Context) error {
	w.logger.Info("Verifying cache image...")

	if err := w.diskManager.VerifyImage(ctx, w.config.ImageName); err != nil {
		return fmt.Errorf("cache image verification failed: %w", err)
	}

	w.logger.Info("Cache image verified successfully")
	return nil
}

func (w *Workflow) cleanupResources(ctx context.Context, resources *WorkflowResources) {
	w.logger.Info("Cleaning up temporary resources...")

	if resources.VMInstance != nil {
		if err := w.vmManager.DeleteVM(ctx, resources.VMInstance.Name, w.config.Zone); err != nil {
			w.logger.Warnf("Failed to cleanup VM %s: %v", resources.VMInstance.Name, err)
		} else {
			w.logger.Infof("Cleaned up VM: %s", resources.VMInstance.Name)
		}
	}

	if resources.CacheDiskName != "" {
		if err := w.diskManager.Cleanup(ctx, resources.CacheDiskName); err != nil {
			w.logger.Warnf("Failed to cleanup disk %s: %v", resources.CacheDiskName, err)
		} else {
			w.logger.Infof("Cleaned up disk: %s", resources.CacheDiskName)
		}
	}

	w.logger.Info("Resource cleanup completed")
}

// WorkflowResources holds references to temporary resources
type WorkflowResources struct {
	VMInstance    *vm.Instance
	CacheDiskName string
}
