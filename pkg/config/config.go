package config

import (
	"time"
)

// ExecutionMode represents how the disk image building should be executed
type ExecutionMode int

const (
	ModeUnspecified ExecutionMode = iota
	ModeLocal                     // -L flag: execute on current machine
	ModeRemote                    // -R flag: execute on temporary GCP VM
)

// String returns the string representation of ExecutionMode
func (m ExecutionMode) String() string {
	switch m {
	case ModeLocal:
		return "local"
	case ModeRemote:
		return "remote"
	default:
		return "unspecified"
	}
}

// Config holds all configuration for the disk image builder
type Config struct {
	// Execution mode
	Mode ExecutionMode

	// Required fields
	ProjectName     string
	ImageName       string
	Zone            string
	GCSPath         string
	ContainerImages []string

	// Optional fields with defaults
	ImageFamilyName string
	JobName         string
	GCPOAuth        string
	DiskSizeGB      int
	ImagePullAuth   string
	Timeout         time.Duration
	Network         string
	Subnet          string
	ServiceAccount  string
	ImageLabels     map[string]string
}

// NewConfig returns a new Config with default values
func NewConfig() *Config {
	return &Config{
		Mode:            ModeUnspecified,
		ImageFamilyName: "gke-disk-image",
		JobName:         "disk-image-build",
		DiskSizeGB:      20,
		ImagePullAuth:   "None",
		Timeout:         20 * time.Minute,
		Network:         "default",
		Subnet:          "default",
		ServiceAccount:  "default",
		ImageLabels:     make(map[string]string),
	}
}

// IsLocalMode returns true if the execution mode is local
func (c *Config) IsLocalMode() bool {
	return c.Mode == ModeLocal
}

// IsRemoteMode returns true if the execution mode is remote
func (c *Config) IsRemoteMode() bool {
	return c.Mode == ModeRemote
}
