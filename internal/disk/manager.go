package disk

import (
	"context"
	"fmt"

	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Manager handles disk operations
type Manager struct {
	gcpClient *gcp.Client
	logger    *log.Logger
	config    *config.Config
}

// Config holds disk configuration
type Config struct {
	Name   string
	Zone   string
	SizeGB int
	Type   string
}

// ImageConfig holds image configuration
type ImageConfig struct {
	Name        string
	SourceDisk  string
	Zone        string
	Family      string
	Labels      map[string]string
	Description string
}

// Disk represents a persistent disk
type Disk struct {
	Name string
	Zone string
}

// NewManager creates a new disk manager
func NewManager(gcpClient *gcp.Client, logger *log.Logger, cfg *config.Config) *Manager {
	return &Manager{
		gcpClient: gcpClient,
		logger:    logger,
		config:    cfg,
	}
}

// CreateDisk creates a new persistent disk
func (m *Manager) CreateDisk(ctx context.Context, config *Config) (*Disk, error) {
	m.logger.Infof("Creating disk: %s", config.Name)

	// Implementation would create actual GCP disk
	disk := &Disk{
		Name: config.Name,
		Zone: config.Zone,
	}

	return disk, nil
}

// DeleteDisk deletes a persistent disk
func (m *Manager) DeleteDisk(ctx context.Context, name, zone string) error {
	m.logger.Infof("Deleting disk: %s", name)

	// Implementation would delete actual GCP disk
	return nil
}

// CreateAndAttach creates and attaches a disk
func (m *Manager) CreateAndAttach(ctx context.Context, vmName string) (string, error) {
	diskName := fmt.Sprintf("%s-disk", m.config.ImageName)
	m.logger.Info(fmt.Sprintf("Creating disk: %s", diskName))

	// Implementation would create and attach actual GCP disk
	return diskName, nil
}

// CreateImage creates a disk image
func (m *Manager) CreateImage(ctx context.Context, diskName string) error {
	m.logger.Info(fmt.Sprintf("Creating image: %s from disk: %s", m.config.ImageName, diskName))

	// Implementation would create actual GCP image
	return nil
}

// Cleanup cleans up temporary resources
func (m *Manager) Cleanup(ctx context.Context, diskName string) error {
	m.logger.Info(fmt.Sprintf("Cleaning up disk: %s", diskName))

	// Implementation would cleanup actual GCP disk
	return nil
}

// VerifyImage verifies a disk image
func (m *Manager) VerifyImage(ctx context.Context, imageName string) error {
	m.logger.Infof("Verifying image: %s", imageName)

	// Implementation would verify actual GCP image
	return nil
}
