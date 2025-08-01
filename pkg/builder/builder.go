package builder

import (
	"context"
	"fmt"

	"github.com/0x00fafa/gke-image-cache-builder/internal/auth"
	"github.com/0x00fafa/gke-image-cache-builder/internal/disk"
	"github.com/0x00fafa/gke-image-cache-builder/internal/image"
	"github.com/0x00fafa/gke-image-cache-builder/internal/vm"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Builder orchestrates the disk image building process
type Builder struct {
	config    *config.Config
	logger    *log.Logger
	authMgr   *auth.Manager
	vmMgr     *vm.Manager
	diskMgr   *disk.Manager
	imageMgr  *image.Cache
	gcpClient *gcp.Client
}

// NewBuilder creates a new Builder instance
func NewBuilder(cfg *config.Config) (*Builder, error) {
	// Create logger
	logger := log.NewLogger(cfg.GCSPath)

	// Create GCP client
	gcpClient, err := gcp.NewClient(cfg.ProjectName, cfg.GCPOAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP client: %w", err)
	}

	// Create managers
	authMgr := auth.NewManager(cfg.GCPOAuth, cfg.ImagePullAuth)
	vmMgr := vm.NewManager(gcpClient, logger, cfg)
	diskMgr := disk.NewManager(gcpClient, logger, cfg)
	imageMgr := image.NewCache(logger)

	return &Builder{
		config:    cfg,
		logger:    logger,
		authMgr:   authMgr,
		vmMgr:     vmMgr,
		diskMgr:   diskMgr,
		imageMgr:  imageMgr,
		gcpClient: gcpClient,
	}, nil
}

// BuildDiskImage executes the complete disk image building process
func (b *Builder) BuildDiskImage(ctx context.Context) error {
	b.logger.Info("Starting disk image build process")

	// Validate environment
	envInfo, err := config.ValidateEnvironment(b.config.Mode)
	if err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	b.logger.Info(fmt.Sprintf("Environment: %s", envInfo.GetEnvironmentDescription()))
	b.logger.Info(fmt.Sprintf("Execution Mode: %s", b.config.Mode.String()))

	// Setup authentication
	if err := b.authMgr.ValidateAll(ctx); err != nil {
		return fmt.Errorf("authentication setup failed: %w", err)
	}

	// Execute based on mode
	if b.config.IsLocalMode() {
		return b.buildLocal(ctx)
	} else {
		return b.buildRemote(ctx)
	}
}

// buildLocal executes the build process on the current machine (GCP VM)
func (b *Builder) buildLocal(ctx context.Context) error {
	b.logger.Info("Executing local build on current GCP VM")

	// Setup VM (install containerd locally)
	if err := b.vmMgr.SetupVM(ctx, nil); err != nil {
		return fmt.Errorf("local VM setup failed: %w", err)
	}

	// Create and attach disk
	diskName, err := b.diskMgr.CreateAndAttach(ctx, "")
	if err != nil {
		return fmt.Errorf("disk creation failed: %w", err)
	}

	// Ensure cleanup
	defer func() {
		b.diskMgr.Cleanup(context.Background(), diskName)
	}()

	// Pull and cache images
	for _, image := range b.config.ContainerImages {
		if err := b.imageMgr.PullAndCache(ctx, image, &disk.Disk{Name: diskName}); err != nil {
			return fmt.Errorf("image caching failed for %s: %w", image, err)
		}
	}

	// Create final image
	if err := b.diskMgr.CreateImage(ctx, diskName); err != nil {
		return fmt.Errorf("image creation failed: %w", err)
	}

	b.logger.Info("Local build completed successfully")
	return nil
}

// buildRemote executes the build process on a temporary GCP VM
func (b *Builder) buildRemote(ctx context.Context) error {
	b.logger.Info("Executing remote build on temporary GCP VM")

	// Create temporary VM with startup script
	vmInstance, err := b.vmMgr.CreateVM(ctx, &vm.Config{
		Name:           fmt.Sprintf("disk-builder-%s", b.config.JobName),
		Zone:           b.config.Zone,
		MachineType:    "e2-standard-2",
		Network:        b.config.Network,
		Subnet:         b.config.Subnet,
		ServiceAccount: b.config.ServiceAccount,
	})
	if err != nil {
		return fmt.Errorf("remote VM creation failed: %w", err)
	}

	// Ensure VM cleanup
	defer func() {
		b.vmMgr.DeleteVM(context.Background(), vmInstance.Name, vmInstance.Zone)
	}()

	// Wait for VM to be ready and setup complete
	if err := b.vmMgr.SetupVM(ctx, vmInstance); err != nil {
		return fmt.Errorf("remote VM setup failed: %w", err)
	}

	// Create disk on the remote VM
	diskName, err := b.diskMgr.CreateAndAttach(ctx, vmInstance.Name)
	if err != nil {
		return fmt.Errorf("remote disk creation failed: %w", err)
	}

	// Ensure disk cleanup
	defer func() {
		b.diskMgr.Cleanup(context.Background(), diskName)
	}()

	// Execute image caching on remote VM
	for _, image := range b.config.ContainerImages {
		if err := b.imageMgr.PullAndCache(ctx, image, &disk.Disk{Name: diskName}); err != nil {
			return fmt.Errorf("remote image caching failed for %s: %w", image, err)
		}
	}

	// Create final image from disk
	if err := b.diskMgr.CreateImage(ctx, diskName); err != nil {
		return fmt.Errorf("image creation failed: %w", err)
	}

	b.logger.Info("Remote build completed successfully")
	return nil
}
