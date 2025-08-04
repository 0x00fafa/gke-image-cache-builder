package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
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
		// Check if running in container environment
		if isRunningInContainer() {
			return fmt.Errorf("local mode (-L) is not supported in container environments. Use remote mode (-R) instead")
		}

		// Check if running on GCP VM
		if !isRunningOnGCP() {
			return fmt.Errorf("local mode (-L) requires execution on a GCP VM instance. Use remote mode (-R) for execution from other environments")
		}

		// Auto-detect zone if not specified
		if c.Zone == "" {
			zone, err := getCurrentVMZone()
			if err != nil {
				return fmt.Errorf("failed to auto-detect zone in local mode: %w", err)
			}
			c.Zone = zone
		}

		// Check container runtime availability
		if err := checkContainerRuntime(); err != nil {
			return fmt.Errorf("container runtime check failed in local mode: %w", err)
		}
	}
	return nil
}

func (c *Config) validateOptionalFields() error {
	if c.DiskSizeGB < 10 || c.DiskSizeGB > 1000 {
		return fmt.Errorf("disk-size must be between 10 and 1000 GB (use --disk-size or 'disk.size_gb' in config file)")
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
		return fmt.Errorf("invalid disk type '%s': %w (use --disk-type or 'disk.disk_type' in config file)", c.DiskType, err)
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
	validTypes := []string{"None", "ServiceAccountToken", "DockerConfig", "BasicAuth"}
	for _, valid := range validTypes {
		if authType == valid {
			return nil
		}
	}
	return fmt.Errorf("unsupported image pull auth type, supported types: %s", strings.Join(validTypes, ", "))
}

// isRunningInContainer checks if the current environment is a container
func isRunningInContainer() bool {
	// Check for container environment indicators
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for container indicators
	if data, err := ioutil.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		// More precise check - exclude GCP VM specific indicators
		if (strings.Contains(content, "docker") ||
			strings.Contains(content, "containerd")) &&
			!strings.Contains(content, "google") {
			return true
		}
	}

	// Check for Kubernetes environment
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	return false
}

// isRunningOnGCP checks if the current environment is a GCP VM
func isRunningOnGCP() bool {
	client := &http.Client{
		Timeout: 10 * time.Second, // Increased timeout for better reliability
	}

	// Try to access GCP metadata server
	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		// Try alternative approach - check for GCP-specific files
		if _, err := os.Stat("/sys/class/dmi/id/product_name"); err == nil {
			// Read the file to check if it contains GCP identifiers
			if data, readErr := ioutil.ReadFile("/sys/class/dmi/id/product_name"); readErr == nil {
				content := strings.ToLower(string(data))
				if strings.Contains(content, "google") || strings.Contains(content, "gcp") {
					return true
				}
			}
		}
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
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
		return "", fmt.Errorf("failed to query metadata server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("metadata server returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata response: %w", err)
	}

	// Zone format: projects/PROJECT_NUMBER/zones/ZONE_NAME
	zonePath := string(body)
	parts := strings.Split(zonePath, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid zone format: %s", zonePath)
	}

	return parts[len(parts)-1], nil
}

// checkContainerRuntime checks if container runtime is available
func checkContainerRuntime() error {
	// Check for containerd
	if err := checkCommand("ctr", "version"); err == nil {
		return nil
	}

	// Check for docker
	if err := checkCommand("docker", "version"); err == nil {
		return nil
	}

	return fmt.Errorf("no container runtime found. Please install containerd or docker")
}

// checkCommand checks if a command is available and working
func checkCommand(command string, args ...string) error {
	// Check if command exists
	_, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("command %s not found: %w", command, err)
	}

	// Try to execute the command with args
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("command %s %v failed: %w", command, args, err)
	}

	return nil
}
