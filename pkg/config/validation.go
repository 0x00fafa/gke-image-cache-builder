package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate checks if all required fields are set and valid
func (c *Config) Validate() error {
	if err := c.validateExecutionMode(); err != nil {
		return err
	}

	if err := c.validateRequiredFields(); err != nil {
		return err
	}

	if err := c.validateModeSpecificFields(); err != nil {
		return err
	}

	if err := c.validateOptionalFields(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateExecutionMode() error {
	if c.Mode == ModeUnspecified {
		return fmt.Errorf("execution mode required: use -L (local) or -R (remote), or specify 'mode: local/remote' in config file")
	}
	return nil
}

func (c *Config) validateRequiredFields() error {
	if c.ProjectName == "" {
		return fmt.Errorf("project-name is required (use --project-name or 'project.name' in config file)")
	}
	if c.DiskImageName == "" {
		return fmt.Errorf("disk-image-name is required (use --disk-image-name or 'cache.name' in config file)")
	}
	if len(c.ContainerImages) == 0 {
		return fmt.Errorf("at least one container-image is required (use --container-image or 'images' list in config file)")
	}
	return nil
}

func (c *Config) validateModeSpecificFields() error {
	if c.IsRemoteMode() {
		if c.Zone == "" {
			return fmt.Errorf("zone is required for remote mode (use --zone or 'execution.zone' in config file)")
		}
	}

	if c.IsLocalMode() {
		if !isRunningOnGCP() {
			return fmt.Errorf("local mode (-L) requires execution on a GCP VM instance")
		}
		// Auto-detect zone if not specified
		if c.Zone == "" {
			zone, err := getCurrentVMZone()
			if err != nil {
				return fmt.Errorf("failed to auto-detect zone in local mode: %w", err)
			}
			c.Zone = zone
		}
	}

	return nil
}

func (c *Config) validateOptionalFields() error {
	if c.CacheSizeGB < 10 || c.CacheSizeGB > 1000 {
		return fmt.Errorf("cache-size must be between 10 and 1000 GB (use --cache-size or 'cache.size_gb' in config file)")
	}

	if c.Timeout < time.Minute {
		return fmt.Errorf("timeout must be at least 1 minute (use --timeout or 'advanced.timeout' in config file)")
	}

	// Validate container image formats
	for i, image := range c.ContainerImages {
		if err := validateContainerImage(image); err != nil {
			return fmt.Errorf("invalid container image #%d '%s': %w (check --container-image or 'images' list in config file)", i+1, image, err)
		}
	}

	// Validate machine type
	if err := validateMachineType(c.MachineType); err != nil {
		return fmt.Errorf("invalid machine type '%s': %w (use --machine-type or 'advanced.machine_type' in config file)", c.MachineType, err)
	}

	// Validate disk type
	if err := validateDiskType(c.DiskType); err != nil {
		return fmt.Errorf("invalid disk type '%s': %w (use --disk-type or 'cache.disk_type' in config file)", c.DiskType, err)
	}

	// Validate image pull auth
	if err := validateImagePullAuth(c.ImagePullAuth); err != nil {
		return fmt.Errorf("invalid image pull auth '%s': %w (use --image-pull-auth or 'auth.image_pull_auth' in config file)", c.ImagePullAuth, err)
	}

	return nil
}

func validateContainerImage(image string) error {
	if image == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	if strings.Contains(image, " ") {
		return fmt.Errorf("image name cannot contain spaces")
	}

	// Basic format validation
	if !strings.Contains(image, ":") && !strings.Contains(image, "@") {
		return fmt.Errorf("image should include a tag or digest (e.g., nginx:latest)")
	}

	return nil
}

func validateMachineType(machineType string) error {
	validTypes := []string{
		"e2-standard-2", "e2-standard-4", "e2-standard-8", "e2-standard-16",
		"e2-highmem-2", "e2-highmem-4", "e2-highmem-8", "e2-highmem-16",
		"e2-highcpu-2", "e2-highcpu-4", "e2-highcpu-8", "e2-highcpu-16",
		"n1-standard-1", "n1-standard-2", "n1-standard-4", "n1-standard-8",
		"n2-standard-2", "n2-standard-4", "n2-standard-8", "n2-standard-16",
	}

	for _, valid := range validTypes {
		if machineType == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported machine type, supported types: %s", strings.Join(validTypes, ", "))
}

func validateDiskType(diskType string) error {
	validTypes := []string{"pd-standard", "pd-ssd", "pd-balanced"}

	for _, valid := range validTypes {
		if diskType == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported disk type, supported types: %s", strings.Join(validTypes, ", "))
}

func validateImagePullAuth(authType string) error {
	validTypes := []string{"None", "ServiceAccountToken"}

	for _, valid := range validTypes {
		if authType == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported image pull auth type, supported types: %s", strings.Join(validTypes, ", "))
}

// isRunningOnGCP checks if the current environment is a GCP VM
func isRunningOnGCP() bool {
	// This would implement actual GCP metadata server check
	return true
}

// getCurrentVMZone gets the zone of the current GCP VM
func getCurrentVMZone() (string, error) {
	// This would implement actual GCP metadata server query
	return "us-west1-b", nil
}
