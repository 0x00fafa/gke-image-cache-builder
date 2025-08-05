package disk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/0x00fafa/gke-image-cache-builder/pkg/gcp"
	"github.com/0x00fafa/gke-image-cache-builder/pkg/log"
)

// Manager handles disk operations with real GCP API calls
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
	m.logger.Infof("Creating disk: %s in zone: %s", config.Name, config.Zone)

	disk := &compute.Disk{
		Name:   config.Name,
		SizeGb: int64(config.SizeGB),
		Type:   fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", m.gcpClient.ProjectName(), config.Zone, config.Type),
		Zone:   config.Zone,
	}

	operation, err := m.gcpClient.Compute().Disks.Insert(m.gcpClient.ProjectName(), config.Zone, disk).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create disk: %w", err)
	}

	// Wait for operation to complete
	if err := m.gcpClient.WaitForOperation(ctx, operation, config.Zone); err != nil {
		return nil, fmt.Errorf("disk creation operation failed: %w", err)
	}

	m.logger.Successf("Disk created successfully: %s", config.Name)

	return &Disk{
		Name: config.Name,
		Zone: config.Zone,
	}, nil
}

// DeleteDisk deletes a persistent disk
func (m *Manager) DeleteDisk(ctx context.Context, name, zone string) error {
	m.logger.Infof("Deleting disk: %s", name)

	operation, err := m.gcpClient.Compute().Disks.Delete(m.gcpClient.ProjectName(), zone, name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete disk: %w", err)
	}

	// Wait for operation to complete
	if err := m.gcpClient.WaitForOperation(ctx, operation, zone); err != nil {
		return fmt.Errorf("disk deletion operation failed: %w", err)
	}

	m.logger.Successf("Disk deleted successfully: %s", name)
	return nil
}

// AttachDisk attaches a disk to a VM instance
func (m *Manager) AttachDisk(ctx context.Context, diskName, instanceName, zone string) error {
	m.logger.Infof("Attaching disk %s to instance %s", diskName, instanceName)

	attachedDisk := &compute.AttachedDisk{
		Source:     fmt.Sprintf("projects/%s/zones/%s/disks/%s", m.gcpClient.ProjectName(), zone, diskName),
		DeviceName: "secondary-disk-image-disk",
		Mode:       "READ_WRITE",
		Boot:       false,
		AutoDelete: false,
	}

	operation, err := m.gcpClient.Compute().Instances.AttachDisk(
		m.gcpClient.ProjectName(), zone, instanceName, attachedDisk).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to attach disk: %w", err)
	}

	// Wait for operation to complete
	if err := m.gcpClient.WaitForOperation(ctx, operation, zone); err != nil {
		return fmt.Errorf("disk attach operation failed: %w", err)
	}

	m.logger.Successf("Disk attached successfully: %s", diskName)
	return nil
}

// GetAttachedDiskDevicePath gets the device path of an attached disk
func (m *Manager) GetAttachedDiskDevicePath(ctx context.Context, diskName, instanceName, zone string) (string, error) {
	m.logger.Debugf("Getting device path for disk: %s", diskName)

	// Get instance information to find attached disks
	instance, err := m.gcpClient.GetInstance(ctx, zone, instanceName)
	if err != nil {
		return "", fmt.Errorf("failed to get instance info: %w", err)
	}

	// Find the disk in attached disks
	for _, disk := range instance.Disks {
		if strings.Contains(disk.Source, diskName) {
			deviceName := disk.DeviceName
			if deviceName == "" {
				deviceName = "secondary-disk-image-disk" // Default device name
			}

			// Return the device path
			devicePath := fmt.Sprintf("/dev/disk/by-id/google-%s", deviceName)
			m.logger.Debugf("Found device path: %s", devicePath)
			return devicePath, nil
		}
	}

	return "", fmt.Errorf("disk %s not found in instance %s", diskName, instanceName)
}

// CheckLocalModePermissions checks if current user has necessary permissions for local mode
func (m *Manager) CheckLocalModePermissions(ctx context.Context) error {
	m.logger.Debug("Checking local mode permissions...")

	// Check if we can create mount points (less strict check)
	testDir := "/tmp/gke-cache-test"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		// If we can't create in /tmp, try current directory
		testDir = "./gke-cache-test"
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return fmt.Errorf("cannot create directories: %w", err)
		}
	}
	defer os.RemoveAll(testDir)

	// Check if mount command is available
	if _, err := exec.LookPath("mount"); err != nil {
		return fmt.Errorf("mount command not available: %w", err)
	}

	if _, err := exec.LookPath("umount"); err != nil {
		return fmt.Errorf("umount command not available: %w", err)
	}

	if _, err := exec.LookPath("mkfs.ext4"); err != nil {
		return fmt.Errorf("mkfs.ext4 command not available: %w", err)
	}

	m.logger.Debug("Local mode permissions validated")
	return nil
}

// DetachDisk detaches a disk from a VM instance
func (m *Manager) DetachDisk(ctx context.Context, diskName, instanceName, zone string) error {
	m.logger.Infof("Detaching disk %s from instance %s", diskName, instanceName)

	operation, err := m.gcpClient.Compute().Instances.DetachDisk(
		m.gcpClient.ProjectName(), zone, instanceName, "secondary-disk-image-disk").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to detach disk: %w", err)
	}

	// Wait for operation to complete
	if err := m.gcpClient.WaitForOperation(ctx, operation, zone); err != nil {
		return fmt.Errorf("disk detach operation failed: %w", err)
	}

	m.logger.Successf("Disk detached successfully: %s", diskName)
	return nil
}

// CreateImage creates a disk image from a disk
func (m *Manager) CreateImage(ctx context.Context, config *ImageConfig) error {
	m.logger.Infof("Creating image: %s from disk: %s", config.Name, config.SourceDisk)

	image := &compute.Image{
		Name:        config.Name,
		SourceDisk:  fmt.Sprintf("projects/%s/zones/%s/disks/%s", m.gcpClient.ProjectName(), config.Zone, config.SourceDisk),
		Description: config.Description,
		Family:      config.Family,
		Labels:      config.Labels,
	}

	operation, err := m.gcpClient.Compute().Images.Insert(m.gcpClient.ProjectName(), image).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	// Wait for operation to complete (global operation)
	if err := m.gcpClient.WaitForOperation(ctx, operation, ""); err != nil {
		return fmt.Errorf("image creation operation failed: %w", err)
	}

	m.logger.Successf("Image created successfully: %s", config.Name)
	return nil
}

// VerifyImage verifies a disk image exists and is ready
func (m *Manager) VerifyImage(ctx context.Context, imageName string) error {
	m.logger.Infof("Verifying image: %s", imageName)

	image, err := m.gcpClient.GetImage(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to verify image: %w", err)
	}

	if image.Status != "READY" {
		return fmt.Errorf("image %s is not ready, status: %s", imageName, image.Status)
	}

	m.logger.Successf("Image verified successfully: %s", imageName)
	return nil
}

// CheckExistingImages checks for existing images and prompts user for action
func (m *Manager) CheckExistingImages(ctx context.Context, family string) (*ExistingImagesAction, error) {
	m.logger.Infof("Checking for existing images in family: %s", family)

	images, err := m.gcpClient.ListImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var existingImages []*compute.Image
	for _, img := range images {
		if img.Family == family {
			existingImages = append(existingImages, img)
		}
	}

	if len(existingImages) == 0 {
		m.logger.Info("No existing images found in family")
		return &ExistingImagesAction{Action: ActionProceed}, nil
	}

	m.logger.Warnf("Found %d existing images in family '%s':", len(existingImages), family)
	for i, img := range existingImages {
		m.logger.Infof("  %d. %s (created: %s)", i+1, img.Name, img.CreationTimestamp)
	}

	// In a real implementation, this would prompt the user for input
	// For now, return a default action
	return &ExistingImagesAction{
		Action:         ActionProceed,
		ExistingImages: existingImages,
	}, nil
}

// ExistingImagesAction represents the user's choice for handling existing images
type ExistingImagesAction struct {
	Action         ActionType
	ExistingImages []*compute.Image
}

// ActionType represents different actions for existing images
type ActionType int

const (
	ActionProceed ActionType = iota
	ActionReplace
	ActionCancel
)

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
