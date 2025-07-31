package builder

import (
	"context"
	"fmt"

	"github.com/ai-on-gke/tools/gke-image-cache-builder/internal/auth"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/internal/disk"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/internal/image"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/internal/vm"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/pkg/config"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/pkg/gcp"
	"github.com/ai-on-gke/tools/gke-image-cache-builder/pkg/log"
)

// Builder handles the image cache creation process
type Builder struct {
	config      *config.Config
	gcpClient   *gcp.Client
	logger      *log.Logger
	authManager *auth.Manager
	vmManager   *vm.Manager
	diskManager *disk.Manager
	imageCache  *image.Cache
}

// NewBuilder creates a new Builder instance
func NewBuilder(cfg *config.Config) (*Builder, error) {
	// Initialize logger (console only, no GCS)
	logger := log.NewConsoleLogger(cfg.Verbose, cfg.Quiet)

	// Initialize GCP client
	gcpClient, err := gcp.NewClient(cfg.ProjectName, cfg.GCPOAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP client: %w", err)
	}

	// Initialize managers
	authManager := auth.NewManager(cfg.GCPOAuth, cfg.ImagePullAuth)
	vmManager := vm.NewManager(gcpClient, logger)
	diskManager := disk.NewManager(gcpClient, logger)
	imageCache := image.NewCache(logger)

	return &Builder{
		config:      cfg,
		gcpClient:   gcpClient,
		logger:      logger,
		authManager: authManager,
		vmManager:   vmManager,
		diskManager: diskManager,
		imageCache:  imageCache,
	}, nil
}

// BuildImageCache orchestrates the entire image cache creation process
func (b *Builder) BuildImageCache(ctx context.Context) error {
	b.logger.Info("Starting image cache build process")
	b.logger.Infof("Cache name: %s", b.config.CacheName)
	b.logger.Infof("Container images: %v", b.config.ContainerImages)

	workflow := NewWorkflow(b.config, b.logger, b.vmManager, b.diskManager, b.imageCache)

	if err := workflow.Execute(ctx); err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	b.logger.Success("Image cache build completed successfully")
	return nil
}
