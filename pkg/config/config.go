package config

import (
	"time"
)

// ExecutionMode defines how the tool executes
type ExecutionMode int

const (
	ModeUnspecified ExecutionMode = iota
	ModeLocal                     // Execute on current GCP VM
	ModeRemote                    // Create temporary GCP VM
)

// Config holds all configuration for the image cache builder
type Config struct {
	// Execution mode
	Mode ExecutionMode

	// Required fields
	ProjectName     string
	DiskImageName   string // 修改：从 CacheName 改为 DiskImageName
	Zone            string
	ContainerImages []string

	// Optional fields with defaults
	DiskFamilyName string            // 改为 DiskFamilyName
	DiskLabels     map[string]string // 改为 DiskLabels
	JobName        string
	GCPOAuth       string
	DiskSizeGB     int // 改为 DiskSizeGB
	ImagePullAuth  string
	Timeout        time.Duration
	Network        string
	Subnet         string
	ServiceAccount string

	// Advanced options
	MachineType string
	Preemptible bool
	DiskType    string

	// Logging options (console only, no GCS)
	Verbose bool
	Quiet   bool
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Mode:           ModeUnspecified,
		DiskFamilyName: "gke-image-cache", // 改为 DiskFamilyName
		JobName:        "image-cache-build",
		DiskSizeGB:     10, // 改为 DiskSizeGB
		ImagePullAuth:  "None",
		Timeout:        20 * time.Minute,
		Network:        "default",
		Subnet:         "default",
		ServiceAccount: "default",
		MachineType:    "e2-standard-2",
		DiskType:       "pd-standard",
		DiskLabels:     make(map[string]string), // 改为 DiskLabels
	}
}

// IsLocalMode returns true if executing on current GCP VM
func (c *Config) IsLocalMode() bool {
	return c.Mode == ModeLocal
}

// IsRemoteMode returns true if creating temporary GCP VM
func (c *Config) IsRemoteMode() bool {
	return c.Mode == ModeRemote
}
