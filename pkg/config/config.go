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
	CacheFamilyName string
	CacheLabels     map[string]string
	JobName         string
	GCPOAuth        string
	CacheSizeGB     int
	ImagePullAuth   string
	Timeout         time.Duration
	Network         string
	Subnet          string
	ServiceAccount  string

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
		Mode:            ModeUnspecified,
		CacheFamilyName: "gke-image-cache",
		JobName:         "image-cache-build",
		CacheSizeGB:     10,
		ImagePullAuth:   "None",
		Timeout:         20 * time.Minute,
		Network:         "default",
		Subnet:          "default",
		ServiceAccount:  "default",
		MachineType:     "e2-standard-2",
		DiskType:        "pd-standard",
		CacheLabels:     make(map[string]string),
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
