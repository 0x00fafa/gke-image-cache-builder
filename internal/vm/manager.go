package vm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"

	"github.com/0x00fafa/gke-image-cache-builder/internal/scripts"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Manager handles VM lifecycle operations with real GCP API calls
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
	m.logger.Infof("Creating VM: %s in zone: %s", config.Name, config.Zone)

	// Prepare startup script
	startupScript := m.generateStartupScript(config)

	// Prepare metadata items
	metadataItems := []*compute.MetadataItems{
		{
			Key:   "startup-script",
			Value: &startupScript,
		},
	}

	// Add SSH key if provided
	if config.SSHPublicKey != "" {
		metadataItems = append(metadataItems, &compute.MetadataItems{
			Key:   "ssh-keys",
			Value: &config.SSHPublicKey,
		})
	}

	instance := &compute.Instance{
		Name:        config.Name,
		MachineType: fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", m.gcpClient.ProjectName(), config.Zone, config.MachineType),
		Zone:        config.Zone,
		Disks: []*compute.AttachedDisk{
			{
				Boot:       true,
				AutoDelete: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-2204-jammy-v20250723",
					DiskSizeGb:  20,
					DiskType:    fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", m.gcpClient.ProjectName(), config.Zone),
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Network: fmt.Sprintf("projects/%s/global/networks/%s", m.gcpClient.ProjectName(), config.Network),
				Subnetwork: fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s",
					m.gcpClient.ProjectName(), m.getRegionFromZone(config.Zone), config.Subnet),
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: config.ServiceAccount,
				Scopes: []string{
					"https://www.googleapis.com/auth/cloud-platform",
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: metadataItems,
		},
		Scheduling: &compute.Scheduling{
			Preemptible: config.Preemptible,
		},
		Tags: &compute.Tags{
			Items: []string{"gke-image-cache-builder"},
		},
	}

	operation, err := m.gcpClient.Compute().Instances.Insert(m.gcpClient.ProjectName(), config.Zone, instance).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	// Wait for operation to complete
	if err := m.gcpClient.WaitForOperation(ctx, operation, config.Zone); err != nil {
		return nil, fmt.Errorf("VM creation operation failed: %w", err)
	}

	// Wait for VM to be running
	if err := m.waitForVMRunning(ctx, config.Name, config.Zone); err != nil {
		return nil, fmt.Errorf("VM failed to start: %w", err)
	}

	// Get the VM instance to retrieve network information
	vmInstance, err := m.gcpClient.GetInstance(ctx, config.Zone, config.Name)
	if err != nil {
		m.logger.Warnf("Failed to get VM instance details: %v", err)
	} else {
		// Print the public IP address
		if len(vmInstance.NetworkInterfaces) > 0 && len(vmInstance.NetworkInterfaces[0].AccessConfigs) > 0 {
			publicIP := vmInstance.NetworkInterfaces[0].AccessConfigs[0].NatIP
			if publicIP != "" {
				m.logger.Infof("VM public IP address: %s", publicIP)
			}
		}
	}

	m.logger.Successf("VM created successfully: %s", config.Name)

	return &Instance{
		Name: config.Name,
		Zone: config.Zone,
	}, nil
}

// DeleteVM deletes a VM instance
func (m *Manager) DeleteVM(ctx context.Context, name, zone string) error {
	m.logger.Infof("Deleting VM: %s", name)

	operation, err := m.gcpClient.Compute().Instances.Delete(m.gcpClient.ProjectName(), zone, name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	// Wait for operation to complete
	if err := m.gcpClient.WaitForOperation(ctx, operation, zone); err != nil {
		return fmt.Errorf("VM deletion operation failed: %w", err)
	}

	m.logger.Successf("VM deleted successfully: %s", name)
	return nil
}

// SetupVM executes the setup script on the VM (for local mode)
func (m *Manager) SetupVM(ctx context.Context, instance *Instance) error {
	m.logger.Infof("Setting up VM: %s", instance.Name)

	// For local mode, execute the script directly
	if err := scripts.ExecuteSetupScript(); err != nil {
		return fmt.Errorf("failed to setup VM: %w", err)
	}

	m.logger.Infof("VM setup completed: %s", instance.Name)
	return nil
}

// ExecuteRemoteImageBuild executes image build on remote VM
func (m *Manager) ExecuteRemoteImageBuild(ctx context.Context, instance *Instance, config *RemoteBuildConfig) error {
	m.logger.Infof("Executing remote image build on VM: %s", instance.Name)

	// Monitor the VM's serial console output for completion
	return m.monitorRemoteExecution(ctx, instance.Name, instance.Zone, config.Timeout)
}

// ValidatePermissions validates GCP permissions
func (m *Manager) ValidatePermissions(ctx context.Context, projectName, zone string) error {
	m.logger.Debug("Validating GCP permissions...")

	// Test basic compute permissions by trying to list instances
	_, err := m.gcpClient.Compute().Instances.List(projectName, zone).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("insufficient GCP permissions: %w", err)
	}

	// Test disk permissions
	_, err = m.gcpClient.Compute().Disks.List(projectName, zone).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("insufficient disk permissions: %w", err)
	}

	// Test image permissions
	_, err = m.gcpClient.Compute().Images.List(projectName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("insufficient image permissions: %w", err)
	}

	m.logger.Debug("GCP permissions validated successfully")
	return nil
}

// generateStartupScript generates the startup script for remote VM
func (m *Manager) generateStartupScript(config *Config) string {
	// Generate image list as a space-separated string
	images := "nginx:latest" // Default fallback
	if len(config.ContainerImages) > 0 {
		images = strings.Join(config.ContainerImages, " ")
	}

	script := `#!/bin/bash
set -e

# Log all output
exec > >(tee /var/log/gke-image-cache-builder.log)
exec 2>&1

echo "Starting GKE Image Cache Builder setup..."

# Download and execute the setup script
cat > /tmp/setup-and-verify.sh << 'SCRIPT_EOF'
` + scripts.GetSetupScript() + `
SCRIPT_EOF

chmod +x /tmp/setup-and-verify.sh

# Execute full workflow with parameters
/tmp/setup-and-verify.sh full-workflow secondary-disk-image-disk ` + config.ImagePullAuth + ` true ` + images + `

echo "Setup completed successfully"
`
	return script
}

// waitForVMRunning waits for VM to be in RUNNING state
func (m *Manager) waitForVMRunning(ctx context.Context, instanceName, zone string) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for VM to start")
		case <-ticker.C:
			instance, err := m.gcpClient.GetInstance(ctx, zone, instanceName)
			if err != nil {
				continue
			}
			if instance.Status == "RUNNING" {
				return nil
			}
			m.logger.Debugf("VM status: %s, waiting...", instance.Status)
		}
	}
}

// monitorRemoteExecution monitors remote execution via serial console
func (m *Manager) monitorRemoteExecution(ctx context.Context, instanceName, zone string, timeout time.Duration) error {
	m.logger.Info("Monitoring remote execution...")

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("remote execution timeout")
		case <-ticker.C:
			// Check serial console output for completion signal
			output, err := m.getSerialConsoleOutput(ctx, instanceName, zone)
			if err != nil {
				m.logger.Debugf("Failed to get serial console output: %v", err)
				continue
			}

			if strings.Contains(output, "Unpacking is completed.") {
				m.logger.Success("Remote execution completed successfully")
				return nil
			}

			if strings.Contains(output, "ERROR") || strings.Contains(output, "FAILED") {
				return fmt.Errorf("remote execution failed, check VM logs")
			}
		}
	}
}

// getSerialConsoleOutput gets the serial console output from VM
func (m *Manager) getSerialConsoleOutput(ctx context.Context, instanceName, zone string) (string, error) {
	output, err := m.gcpClient.Compute().Instances.GetSerialPortOutput(
		m.gcpClient.ProjectName(), zone, instanceName).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return output.Contents, nil
}

// getRegionFromZone extracts region from zone name
func (m *Manager) getRegionFromZone(zone string) string {
	parts := strings.Split(zone, "-")
	if len(parts) >= 2 {
		return strings.Join(parts[:2], "-")
	}
	return zone
}

// Config holds VM configuration
type Config struct {
	Name            string
	Zone            string
	MachineType     string
	Network         string
	Subnet          string
	ServiceAccount  string
	Preemptible     bool
	ContainerImages []string
	ImagePullAuth   string
	SSHPublicKey    string
}

// RemoteBuildConfig holds remote build configuration
type RemoteBuildConfig struct {
	DeviceName      string
	AuthMechanism   string
	StoreChecksums  bool
	ContainerImages []string
	Timeout         time.Duration
}

// Instance represents a VM instance
type Instance struct {
	Name string
	Zone string
}
