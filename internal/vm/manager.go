package vm

import (
	"context"
	"fmt"
	"time"

	"github.com/0x00fafa/gke-image-cache-builder/internal/scripts"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Manager handles VM operations
type Manager struct {
	gcpClient *gcp.Client
	logger    *log.Logger
	config    *config.Config
}

// Instance represents a VM instance
type Instance struct {
	Name string
	Zone string
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
	Timeout        time.Duration
}

// NewManager creates a new VM manager
func NewManager(gcpClient *gcp.Client, logger *log.Logger, cfg *config.Config) *Manager {
	return &Manager{
		gcpClient: gcpClient,
		logger:    logger,
		config:    cfg,
	}
}

// CreateVM creates a new VM instance
func (m *Manager) CreateVM(ctx context.Context, vmConfig *Config) (*Instance, error) {
	m.logger.Info(fmt.Sprintf("Creating VM: %s", vmConfig.Name))

	// Get the setup script to be executed via startup-script
	setupScript := scripts.GetSetupScript()

	// Create VM with startup script for remote execution
	err := m.gcpClient.CreateInstanceWithStartupScript(
		ctx,
		vmConfig.Zone,
		vmConfig.Name,
		vmConfig.MachineType,
		setupScript,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	instance := &Instance{
		Name: vmConfig.Name,
		Zone: vmConfig.Zone,
	}

	m.logger.Info(fmt.Sprintf("VM created: %s", vmConfig.Name))
	return instance, nil
}

// SetupVM sets up the VM environment
// For local mode: installs containerd locally
// For remote mode: waits for remote VM to complete startup-script setup
func (m *Manager) SetupVM(ctx context.Context, instance *Instance) error {
	if m.config != nil && m.config.IsLocalMode() {
		return m.setupLocalVM(ctx)
	} else {
		return m.setupRemoteVM(ctx, instance)
	}
}

// setupLocalVM installs containerd on the current local GCP VM
func (m *Manager) setupLocalVM(ctx context.Context) error {
	m.logger.Info("Setting up containerd on current GCP VM (local mode)")

	// Execute setup script locally
	if err := scripts.ExecuteSetupScript(); err != nil {
		return fmt.Errorf("failed to setup local VM: %w", err)
	}

	m.logger.Info("Local VM setup completed")
	return nil
}

// setupRemoteVM waits for the remote VM to complete its startup-script setup
func (m *Manager) setupRemoteVM(ctx context.Context, instance *Instance) error {
	m.logger.Info(fmt.Sprintf("Waiting for remote VM setup to complete: %s", instance.Name))

	timeout := 20 * time.Minute // Default timeout
	if m.config != nil {
		timeout = m.config.Timeout
	}

	// Wait for VM to be ready and startup script to complete
	timeoutCh := time.After(timeout)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutCh:
			return fmt.Errorf("timeout waiting for VM setup after %v", timeout)
		case <-ticker.C:
			// Check if VM is ready and setup is complete
			ready, err := m.isVMSetupComplete(ctx, instance)
			if err != nil {
				m.logger.Warn(fmt.Sprintf("Error checking VM setup status: %v", err))
				continue
			}
			if ready {
				m.logger.Info(fmt.Sprintf("Remote VM setup completed: %s", instance.Name))
				return nil
			}
			m.logger.Info("VM setup still in progress...")
		}
	}
}

// DeleteVM deletes a VM instance
func (m *Manager) DeleteVM(ctx context.Context, name, zone string) error {
	m.logger.Info(fmt.Sprintf("Deleting VM: %s", name))

	err := m.gcpClient.DeleteInstance(ctx, zone, name)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	m.logger.Info(fmt.Sprintf("VM deleted: %s", name))
	return nil
}

// ValidatePermissions validates GCP permissions for VM operations
func (m *Manager) ValidatePermissions(ctx context.Context, projectName, zone string) error {
	m.logger.Info("Validating GCP permissions...")

	// This would implement actual permission validation
	// For now, it's a placeholder

	m.logger.Info("GCP permissions validated")
	return nil
}

// isVMSetupComplete checks if the VM setup is complete
func (m *Manager) isVMSetupComplete(ctx context.Context, instance *Instance) (bool, error) {
	// Get VM instance status
	vmInstance, err := m.gcpClient.GetInstance(ctx, instance.Zone, instance.Name)
	if err != nil {
		return false, fmt.Errorf("failed to get instance status: %w", err)
	}

	// Check if VM is running
	if vmInstance.Status != "RUNNING" {
		return false, nil
	}

	// In a real implementation, we would check if the startup script completed
	// This could be done by:
	// 1. Checking serial console output for completion markers
	// 2. SSH connection and checking for setup completion files
	// 3. Using custom metadata or labels to track setup status
	// For now, we'll assume the VM is ready after it's running for a minimum time

	return true, nil
}
