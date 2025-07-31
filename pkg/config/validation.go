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
		return fmt.Errorf("execution mode required: use -L (local) or -R (remote)")
	}
	return nil
}

func (c *Config) validateRequiredFields() error {
	if c.ProjectName == "" {
		return fmt.Errorf("project-name is required")
	}
	if c.DiskImageName == "" { // 修改：从 CacheName 改为 DiskImageName
		return fmt.Errorf("disk-image-name is required")
	}
	if len(c.ContainerImages) == 0 {
		return fmt.Errorf("at least one container-image is required")
	}
	return nil
}

func (c *Config) validateModeSpecificFields() error {
	if c.IsRemoteMode() {
		if c.Zone == "" {
			return fmt.Errorf("zone is required for remote mode (-R)")
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
		return fmt.Errorf("cache-size must be between 10 and 1000 GB")
	}

	if c.Timeout < time.Minute {
		return fmt.Errorf("timeout must be at least 1 minute")
	}

	// Validate container image formats
	for _, image := range c.ContainerImages {
		if err := validateContainerImage(image); err != nil {
			return fmt.Errorf("invalid container image '%s': %w", image, err)
		}
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

	return nil
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
