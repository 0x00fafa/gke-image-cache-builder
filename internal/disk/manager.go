package disk

import (
	"context"

	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Manager handles disk operations
type Manager struct {
	gcpClient *gcp.Client
	logger    *log.Logger
}

// NewManager creates a new disk manager
func NewManager(gcpClient *gcp.Client, logger *log.Logger) *Manager {
	return &Manager{
		gcpClient: gcpClient,
		logger:    logger,
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

// CreateImage creates a disk image
func (m *Manager) CreateImage(ctx context.Context, config *ImageConfig) error {
	m.logger.Infof("Creating image: %s", config.Name)

	// Implementation would create actual GCP image
	return nil
}

// VerifyImage verifies a disk image
func (m *Manager) VerifyImage(ctx context.Context, imageName string) error {
	m.logger.Infof("Verifying image: %s", imageName)

	// Implementation would verify actual GCP image
	return nil
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
