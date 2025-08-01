package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// EnvironmentType represents the type of environment
type EnvironmentType int

const (
	EnvironmentLocal EnvironmentType = iota
	EnvironmentContainer
	EnvironmentGCPVM
)

// EnvironmentInfo holds information about the current environment
type EnvironmentInfo struct {
	Type         EnvironmentType
	IsContainer  bool
	IsGCPVM      bool
	CurrentZone  string
	Restrictions []string
}

// GetEnvironmentDescription returns a human-readable description of the environment
func (e *EnvironmentInfo) GetEnvironmentDescription() string {
	switch e.Type {
	case EnvironmentGCPVM:
		if e.CurrentZone != "" {
			return "GCP VM (zone: " + e.CurrentZone + ")"
		}
		return "GCP VM"
	case EnvironmentContainer:
		return "Container Environment"
	case EnvironmentLocal:
		return "Local Machine"
	default:
		return "Unknown Environment"
	}
}

// GetRecommendedMode returns the recommended execution mode for the current environment
func (e *EnvironmentInfo) GetRecommendedMode() ExecutionMode {
	switch e.Type {
	case EnvironmentGCPVM:
		return ModeLocal // Cost-effective on GCP VMs
	case EnvironmentContainer:
		return ModeRemote // Only option in containers
	case EnvironmentLocal:
		return ModeRemote // Safer option for local machines
	default:
		return ModeRemote
	}
}

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
	if c.ImageName == "" {
		return fmt.Errorf("image-name is required")
	}
	if c.GCSPath == "" {
		return fmt.Errorf("gcs-path is required")
	}
	if len(c.ContainerImages) == 0 {
		return fmt.Errorf("at least one container-image is required")
	}
	return nil
}

func (c *Config) validateModeSpecificFields() error {
	// Check for container environment restrictions
	if isRunningInContainer() && c.IsLocalMode() {
		return fmt.Errorf("local mode (-L) is not supported in container environments. Container environments lack the necessary container runtime and GCP VM context required for local mode. Use remote mode (-R) instead, which creates a temporary GCP VM with the required environment")
	}

	if c.IsRemoteMode() {
		if c.Zone == "" {
			return fmt.Errorf("zone is required for remote mode")
		}
	}

	if c.IsLocalMode() {
		if !isRunningOnGCP() {
			return fmt.Errorf("local mode (-L) requires execution on a GCP VM instance. Current environment is not a GCP VM. Use remote mode (-R) to create a temporary GCP VM, or run this tool on a GCP VM instance")
		}
		// Auto-detect zone if not specified
		if c.Zone == "" {
			zone, err := getCurrentVMZone()
			if err != nil {
				return fmt.Errorf("failed to auto-detect zone in local mode: %w. Please specify zone manually with --zone", err)
			}
			c.Zone = zone
		}
	}

	return nil
}

func (c *Config) validateOptionalFields() error {
	if c.DiskSizeGB < 10 || c.DiskSizeGB > 1000 {
		return fmt.Errorf("disk-size-gb must be between 10 and 1000")
	}

	if c.Timeout < time.Minute {
		return fmt.Errorf("timeout must be at least 1 minute")
	}

	// Validate container image formats
	for i, image := range c.ContainerImages {
		if err := validateContainerImage(image); err != nil {
			return fmt.Errorf("invalid container image #%d '%s': %w", i+1, image, err)
		}
	}

	return nil
}

// ValidateEnvironment validates the current environment and execution mode
func ValidateEnvironment(mode ExecutionMode) (*EnvironmentInfo, error) {
	envInfo := &EnvironmentInfo{}

	// Detect environment type
	envInfo.IsContainer = isRunningInContainer()
	envInfo.IsGCPVM = isRunningOnGCP()

	// Set environment type
	if envInfo.IsGCPVM {
		envInfo.Type = EnvironmentGCPVM
		if zone, err := getCurrentVMZone(); err == nil {
			envInfo.CurrentZone = zone
		}
	} else if envInfo.IsContainer {
		envInfo.Type = EnvironmentContainer
	} else {
		envInfo.Type = EnvironmentLocal
	}

	// Validate mode compatibility
	if err := validateModeCompatibility(envInfo, mode); err != nil {
		return envInfo, err
	}

	return envInfo, nil
}

// validateModeCompatibility checks if the execution mode is compatible with the environment
func validateModeCompatibility(envInfo *EnvironmentInfo, mode ExecutionMode) error {
	switch envInfo.Type {
	case EnvironmentContainer:
		if mode == ModeLocal {
			envInfo.Restrictions = append(envInfo.Restrictions, "Local mode (-L) is not supported in container environments")
			return fmt.Errorf("local mode (-L) is not supported in container environments. Use remote mode (-R) instead")
		}

	case EnvironmentLocal:
		if mode == ModeLocal {
			envInfo.Restrictions = append(envInfo.Restrictions, "Local mode (-L) requires containerd installation on the host machine")
			return fmt.Errorf("local mode (-L) is not recommended on local machines. Use remote mode (-R) for safety")
		}

	case EnvironmentGCPVM:
		// GCP VMs support both modes
		if mode == ModeLocal {
			envInfo.Restrictions = append(envInfo.Restrictions, "Local mode will install containerd on this GCP VM")
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

	// Basic format validation
	if !strings.Contains(image, ":") && !strings.Contains(image, "@") {
		return fmt.Errorf("image should include a tag or digest (e.g., nginx:latest)")
	}

	return nil
}

// isRunningOnGCP checks if the current environment is a GCP VM
func isRunningOnGCP() bool {
	// Try to access GCP metadata server
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200 && resp.Header.Get("Metadata-Flavor") == "Google"
}

// getCurrentVMZone gets the zone of the current GCP VM
func getCurrentVMZone() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata request: %w", err)
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to access GCP metadata server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("metadata server returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata response: %w", err)
	}

	// Response format: projects/PROJECT_NUMBER/zones/ZONE_NAME
	zonePath := string(body)
	parts := strings.Split(zonePath, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected zone format: %s", zonePath)
	}

	return parts[len(parts)-1], nil
}

// isRunningInContainer checks if the current environment is a container
func isRunningInContainer() bool {
	// Method 1: Check for /.dockerenv file (Docker)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Method 2: Check /proc/1/cgroup for container indicators
	if data, err := ioutil.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") ||
			strings.Contains(content, "containerd") ||
			strings.Contains(content, "kubepods") {
			return true
		}
	}

	// Method 3: Check for Kubernetes environment variables
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	// Method 4: Check for common container environment variables
	containerEnvVars := []string{
		"DOCKER_CONTAINER",
		"CONTAINER",
		"PODMAN_CONTAINER",
	}

	for _, envVar := range containerEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}
