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
		// Format SSH key properly for GCP
		// GCP expects: "username:ssh-rsa AAAAB3NzaC1yc2E... user@host"
		sshKey := config.SSHPublicKey
		if !strings.Contains(sshKey, ":") {
			// Use "abc" as the username as requested
			sshKey = "abc:" + sshKey
		}
		metadataItems = append(metadataItems, &compute.MetadataItems{
			Key:   "ssh-keys",
			Value: &sshKey,
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
		// Print the public IP address and SSH connection info
		if len(vmInstance.NetworkInterfaces) > 0 && len(vmInstance.NetworkInterfaces[0].AccessConfigs) > 0 {
			publicIP := vmInstance.NetworkInterfaces[0].AccessConfigs[0].NatIP
			if publicIP != "" {
				m.logger.Infof("VM public IP address: %s", publicIP)
				m.logger.Infof("SSH connection command: ssh abc@%s", publicIP)
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
	// Prepare the image list
	images := "nginx:latest" // Default fallback
	if len(config.ContainerImages) > 0 {
		images = strings.Join(config.ContainerImages, " ")
	}

	// Prepare the auth mechanism
	authMechanism := "none"
	if config.ImagePullAuth != "" {
		authMechanism = config.ImagePullAuth
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

# Execute the full workflow in background to avoid blocking startup
{
    # Wait for system to be fully ready
    sleep 30
    
    echo "Starting full GKE Image Cache Builder workflow..."
    
    # Execute setup (environment preparation)
    /tmp/setup-and-verify.sh setup
    
    # Setup containerd
    /tmp/setup-and-verify.sh setup-containerd
    
    echo "Environment setup completed."
    
    # Create a flag file to indicate environment is ready
    touch /tmp/environment_ready.flag
    
    # Wait for the disk to be attached by the main process
    echo "Waiting for disk to be attached..."
    for i in {1..60}; do  # Wait up to 5 minutes
        if [ -b /dev/disk/by-id/google-secondary-disk-image-disk ]; then
            echo "Disk attached successfully"
            break
        fi
        echo "Waiting for disk... ($i/60)"
        sleep 5
    done
    
    # Check if disk is attached
    if [ ! -b /dev/disk/by-id/google-secondary-disk-image-disk ]; then
        echo "ERROR: Disk not attached within timeout period"
        # Log available disks for debugging
        echo "Available disks:"
        ls -la /dev/disk/by-id/
        exit 1
    fi
    
    echo "Disk attached, starting image processing..."
    
    # Wait a bit more for containerd to be fully ready
    sleep 30
    
    # Execute the full workflow
    /tmp/setup-and-verify.sh full-workflow secondary-disk-image-disk ` + authMechanism + ` true ` + images + `
    
    echo "Unpacking is completed."
    
    # Create completion flag
    touch /tmp/workflow_completed.flag
    
    echo "Full workflow completed successfully"
} &

echo "Setup script initiated in background"
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

// GetSerialConsoleOutput gets the serial console output from VM (public method)
func (m *Manager) GetSerialConsoleOutput(ctx context.Context, instanceName, zone string) (string, error) {
	output, err := m.gcpClient.Compute().Instances.GetSerialPortOutput(
		m.gcpClient.ProjectName(), zone, instanceName).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return output.Contents, nil
}

// getSerialConsoleOutput gets the serial console output from VM (private method for backward compatibility)
func (m *Manager) getSerialConsoleOutput(ctx context.Context, instanceName, zone string) (string, error) {
	return m.GetSerialConsoleOutput(ctx, instanceName, zone)
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
