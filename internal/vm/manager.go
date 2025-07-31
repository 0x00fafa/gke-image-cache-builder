package vm

import (
	"context"
	"fmt"

	"github.com/0x00fafa/gke-image-cache-builder/internal/scripts"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Manager handles VM lifecycle operations
type Manager struct {
	gcpClient *gcp.Client
	logger    *log.Logger
}

// NewManager creates a new VM manager
func NewManager(gcpClient *gcp.Client, logger *log.Logger) *Manager {
	return &Manager{
		gcpClient: gcpClient,
		logger:    logger,
	}
}

// CreateVM creates a new VM instance
func (m *Manager) CreateVM(ctx context.Context, config *Config) (*Instance, error) {
	m.logger.Infof("Creating VM: %s", config.Name)

	// Implementation would create actual GCP VM
	instance := &Instance{
		Name: config.Name,
		Zone: config.Zone,
	}

	return instance, nil
}

// DeleteVM deletes a VM instance
func (m *Manager) DeleteVM(ctx context.Context, name, zone string) error {
	m.logger.Infof("Deleting VM: %s", name)

	// Implementation would delete actual GCP VM
	return nil
}

// SetupVM executes the embedded setup script on the VM
func (m *Manager) SetupVM(ctx context.Context, instance *Instance) error {
	m.logger.Infof("Setting up VM: %s", instance.Name)

	// Execute the embedded setup script
	if err := scripts.ExecuteSetupScript(); err != nil {
		return fmt.Errorf("failed to setup VM: %w", err)
	}

	m.logger.Infof("VM setup completed: %s", instance.Name)
	return nil
}

// ValidatePermissions validates GCP permissions
func (m *Manager) ValidatePermissions(ctx context.Context, projectName, zone string) error {
	m.logger.Debug("Validating GCP permissions...")

	// Implementation would validate actual GCP permissions
	return nil
}

// Config holds VM configuration
type Config struct {
	Name           string
	Zone           string
	MachineType    string
	Network        string
	Subnet         string
	ServiceAccount string
	Preemptible    bool
}

// Instance represents a VM instance
type Instance struct {
	Name string
	Zone string
}
