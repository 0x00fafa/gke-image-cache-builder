package builder

import (
	"context"
	"fmt"

	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"

	"github.com/0x00fafa/gke-image-cache-builder/internal/disk"
	"github.com/0x00fafa/gke-image-cache-builder/internal/image"
	"github.com/0x00fafa/gke-image-cache-builder/internal/vm"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
)

type Builder struct {
	config      *config.Config
	logger      *log.Logger
	gcpClient   *gcp.Client
	vmManager   *vm.Manager
	diskManager *disk.Manager
	imageCache  *image.Cache
}

func NewBuilder(cfg *config.Config, logger *log.Logger, gcpClient *gcp.Client) *Builder {
	return &Builder{
		config:      cfg,
		logger:      logger,
		gcpClient:   gcpClient,
		vmManager:   vm.NewManager(gcpClient, logger),
		diskManager: disk.NewManager(gcpClient, logger),
		imageCache:  image.NewCache(logger),
	}
}

// SetSSHPublicKey sets the SSH public key for VM access
func (b *Builder) SetSSHPublicKey(key string) {
	b.config.SSHPublicKey = key
}

func (b *Builder) BuildImageCache(ctx context.Context) error {
	b.logger.Info("Starting image cache build process")
	b.logger.Infof("Disk image name: %s", b.config.DiskImageName)
	b.logger.Infof("Container images: %v", b.config.ContainerImages)

	// Create a channel to signal when the build is done
	buildDone := make(chan struct{})

	// Start the build in a goroutine
	var buildErr error
	go func() {
		defer close(buildDone)
		workflow := NewWorkflow(b.config, b.logger, b.vmManager, b.diskManager, b.imageCache, b.gcpClient)
		buildErr = workflow.Execute(ctx)
	}()

	// Wait for the build to complete
	<-buildDone

	if buildErr != nil {
		// Even if the build failed, we still return the error
		// The workflow should have scheduled cleanup
		return fmt.Errorf("workflow execution failed: %w", buildErr)
	}

	b.logger.Success("Image cache build completed successfully")
	return nil
}
