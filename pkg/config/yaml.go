package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the YAML configuration file structure
type YAMLConfig struct {
	Execution ExecutionConfig `yaml:"execution"`
	Project   ProjectConfig   `yaml:"project"`
	Image     ImageConfig     `yaml:"image"`
	Images    []string        `yaml:"images"`
	Network   NetworkConfig   `yaml:"network,omitempty"`
	Advanced  AdvancedConfig  `yaml:"advanced,omitempty"`
	Auth      AuthConfig      `yaml:"auth,omitempty"`
	Logging   LoggingConfig   `yaml:"logging,omitempty"`
}

type ExecutionConfig struct {
	Mode string `yaml:"mode"` // "local" or "remote"
	Zone string `yaml:"zone,omitempty"`
}

type ProjectConfig struct {
	Name string `yaml:"name"`
}

type ImageConfig struct {
	Name   string            `yaml:"name"`
	SizeGB int               `yaml:"size_gb,omitempty"`
	Family string            `yaml:"family,omitempty"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

type NetworkConfig struct {
	Network string `yaml:"network,omitempty"`
	Subnet  string `yaml:"subnet,omitempty"`
}

type AdvancedConfig struct {
	Timeout     string `yaml:"timeout,omitempty"`
	JobName     string `yaml:"job_name,omitempty"`
	MachineType string `yaml:"machine_type,omitempty"`
	Preemptible bool   `yaml:"preemptible,omitempty"`
}

type AuthConfig struct {
	GCPOAuth       string `yaml:"gcp_oauth,omitempty"`
	ServiceAccount string `yaml:"service_account,omitempty"`
	ImagePullAuth  string `yaml:"image_pull_auth,omitempty"`
}

type LoggingConfig struct {
	Verbose bool `yaml:"verbose,omitempty"`
	Quiet   bool `yaml:"quiet,omitempty"`
}

// LoadFromYAML loads configuration from a YAML file
func (c *Config) LoadFromYAML(filePath string) error {
	if filePath == "" {
		return nil // No config file specified
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", filePath)
	}

	// Read file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %s: %w", filePath, err)
	}

	// Parse YAML
	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return fmt.Errorf("failed to parse YAML configuration file %s: %w", filePath, err)
	}

	// Apply configuration (only if not already set by command line)
	if err := c.applyYAMLConfig(&yamlConfig, filePath); err != nil {
		return fmt.Errorf("failed to apply configuration from %s: %w", filePath, err)
	}

	return nil
}

// applyYAMLConfig applies YAML configuration to Config struct
// Command line parameters take precedence over config file
func (c *Config) applyYAMLConfig(yamlConfig *YAMLConfig, filePath string) error {
	// Execution mode
	if c.Mode == ModeUnspecified && yamlConfig.Execution.Mode != "" {
		switch yamlConfig.Execution.Mode {
		case "local":
			c.Mode = ModeLocal
		case "remote":
			c.Mode = ModeRemote
		default:
			return fmt.Errorf("invalid execution mode '%s' in %s, must be 'local' or 'remote'", yamlConfig.Execution.Mode, filePath)
		}
	}

	// Zone
	if c.Zone == "" && yamlConfig.Execution.Zone != "" {
		c.Zone = yamlConfig.Execution.Zone
	}

	// Project name
	if c.ProjectName == "" && yamlConfig.Project.Name != "" {
		c.ProjectName = yamlConfig.Project.Name
	}

	// Image configuration
	if c.ImageName == "" && yamlConfig.Image.Name != "" {
		c.ImageName = yamlConfig.Image.Name
	}

	if c.DiskSizeGB == 20 && yamlConfig.Image.SizeGB > 0 { // 20 is default
		c.DiskSizeGB = yamlConfig.Image.SizeGB
	}

	if c.ImageFamilyName == "gke-disk-image" && yamlConfig.Image.Family != "" { // default value
		c.ImageFamilyName = yamlConfig.Image.Family
	}

	// Labels (merge with existing)
	if len(yamlConfig.Image.Labels) > 0 {
		if c.ImageLabels == nil {
			c.ImageLabels = make(map[string]string)
		}
		for k, v := range yamlConfig.Image.Labels {
			if _, exists := c.ImageLabels[k]; !exists { // Don't override CLI labels
				c.ImageLabels[k] = v
			}
		}
	}

	// Container images (append if not already set)
	if len(c.ContainerImages) == 0 && len(yamlConfig.Images) > 0 {
		c.ContainerImages = yamlConfig.Images
	}

	// Network configuration
	if c.Network == "default" && yamlConfig.Network.Network != "" { // default value
		c.Network = yamlConfig.Network.Network
	}

	if c.Subnet == "default" && yamlConfig.Network.Subnet != "" { // default value
		c.Subnet = yamlConfig.Network.Subnet
	}

	// Advanced configuration
	if c.Timeout == 20*time.Minute && yamlConfig.Advanced.Timeout != "" { // default value
		timeout, err := time.ParseDuration(yamlConfig.Advanced.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout format '%s' in %s: %w", yamlConfig.Advanced.Timeout, filePath, err)
		}
		c.Timeout = timeout
	}

	if c.JobName == "disk-image-build" && yamlConfig.Advanced.JobName != "" { // default value
		c.JobName = yamlConfig.Advanced.JobName
	}

	// Authentication
	if c.GCPOAuth == "" && yamlConfig.Auth.GCPOAuth != "" {
		c.GCPOAuth = yamlConfig.Auth.GCPOAuth
	}

	if c.ServiceAccount == "default" && yamlConfig.Auth.ServiceAccount != "" { // default value
		c.ServiceAccount = yamlConfig.Auth.ServiceAccount
	}

	if c.ImagePullAuth == "None" && yamlConfig.Auth.ImagePullAuth != "" { // default value
		c.ImagePullAuth = yamlConfig.Auth.ImagePullAuth
	}

	return nil
}

// GenerateYAMLTemplate generates a YAML configuration template
func GenerateYAMLTemplate(outputPath string, templateType string) error {
	var template string

	switch templateType {
	case "basic":
		template = basicYAMLTemplate
	case "advanced":
		template = advancedYAMLTemplate
	case "ci-cd":
		template = cicdYAMLTemplate
	case "ml":
		template = mlYAMLTemplate
	default:
		template = basicYAMLTemplate
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write template to file
	if err := ioutil.WriteFile(outputPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write template to %s: %w", outputPath, err)
	}

	return nil
}

// ValidateYAMLFile validates a YAML configuration file
func ValidateYAMLFile(filePath string) error {
	// Create a temporary config to test loading
	tempConfig := NewConfig()
	if err := tempConfig.LoadFromYAML(filePath); err != nil {
		return err
	}

	// Validate the loaded configuration
	if err := tempConfig.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed for %s: %w", filePath, err)
	}

	return nil
}

const basicYAMLTemplate = `# GKE Image Cache Builder - Basic Configuration Template
# This template provides a minimal configuration for building image cache disks

execution:
  mode: local  # Options: local, remote
  # zone: us-west1-b  # Required for remote mode

project:
  name: my-project  # Replace with your GCP project name

image:
  name: web-app-cache  # Name for the disk image
  size_gb: 20  # Disk size in GB
  family: gke-disk-image  # Image family name
  labels:
    env: development
    team: platform

# Container images to cache
images:
  - nginx:latest
  - redis:alpine
  - postgres:13

# Optional network configuration for build VM (remote mode only)
# network:
#   network: default
#   subnet: default

# Optional advanced settings
# advanced:
#   timeout: 20m
#   job_name: disk-image-build

# Optional authentication
# auth:
#   gcp_oauth: /path/to/service-account.json
#   service_account: default
#   image_pull_auth: None
`

const advancedYAMLTemplate = `# GKE Image Cache Builder - Advanced Configuration Template
# This template includes all available configuration options

execution:
  mode: remote  # Options: local, remote
  zone: us-west1-b  # GCP zone (required for remote mode)

project:
  name: production-project  # GCP project name

image:
  name: microservices-cache  # Disk image name
  size_gb: 50  # Disk size in GB
  family: production-cache  # Image family name
  labels:
    env: production
    team: platform
    version: v1.0.0
    cost-center: engineering

# Container images to cache
images:
  - gcr.io/my-project/api-gateway:v2.1.0
  - gcr.io/my-project/user-service:v1.8.3
  - gcr.io/my-project/order-service:v1.5.2
  - gcr.io/my-project/payment-service:v2.0.1
  - nginx:1.21
  - redis:6.2-alpine
  - postgres:13

# Network configuration for build VM (remote mode only)
network:
  network: production-vpc      # VPC network for build VM
  subnet: production-subnet    # Subnet for build VM

# Advanced settings
advanced:
  timeout: 45m  # Build timeout
  job_name: production-cache-build

# Authentication configuration
auth:
  gcp_oauth: /path/to/service-account.json
  service_account: cache-builder@production-project.iam.gserviceaccount.com
  image_pull_auth: ServiceAccountToken
`

const cicdYAMLTemplate = `# GKE Image Cache Builder - CI/CD Configuration Template
# Optimized for continuous integration and deployment pipelines

execution:
  mode: remote  # Always use remote mode in CI/CD
  zone: us-central1-a  # Choose zone close to your CI/CD infrastructure

project:
  name: ${GCP_PROJECT}  # Use environment variable

image:
  name: ci-cache-${BUILD_ID}  # Dynamic naming with build ID
  size_gb: 30
  family: ci-cache
  labels:
    env: ci
    build-id: ${BUILD_ID}
    branch: ${GIT_BRANCH}
    commit: ${GIT_COMMIT}

# Application images (replace with your images)
images:
  - gcr.io/${GCP_PROJECT}/app:${GIT_SHA}
  - gcr.io/${GCP_PROJECT}/worker:${GIT_SHA}
  - gcr.io/${GCP_PROJECT}/scheduler:${GIT_SHA}
  # Base images
  - node:16-alpine
  - nginx:1.21
  - redis:6.2-alpine

# Network configuration for CI/CD build VM
network:
  network: default
  subnet: default

# CI/CD optimized settings
advanced:
  timeout: 30m  # Reasonable timeout for CI/CD
  job_name: ci-build-${BUILD_NUMBER}

# Authentication (use service account in CI/CD)
auth:
  service_account: ci-cache-builder@${GCP_PROJECT}.iam.gserviceaccount.com
  image_pull_auth: ServiceAccountToken
`

const mlYAMLTemplate = `# GKE Image Cache Builder - ML/AI Configuration Template
# Optimized for machine learning and AI workloads

execution:
  mode: remote  # Remote mode for flexibility
  zone: us-west1-b  # Choose GPU-available zone if needed

project:
  name: ml-platform-project

image:
  name: ml-models-cache
  size_gb: 200  # Large size for ML models and datasets
  family: ml-cache
  labels:
    env: production
    workload: ml
    team: data-science
    model-version: v3.2.0

# ML/AI container images
images:
  # TensorFlow
  - tensorflow/tensorflow:2.8.0-gpu
  - tensorflow/tensorflow:2.8.0
  - tensorflow/serving:2.8.0
  
  # PyTorch
  - pytorch/pytorch:1.11.0-cuda11.3-cudnn8-runtime
  - pytorch/pytorch:1.11.0-cuda11.3-cudnn8-devel
  
  # Jupyter and ML tools
  - jupyter/tensorflow-notebook:latest
  - jupyter/pytorch-notebook:latest
  
  # Custom ML models (replace with your images)
  - gcr.io/ml-platform-project/custom-model:v3.2.0
  - gcr.io/ml-platform-project/data-processor:v1.5.0
  - gcr.io/ml-platform-project/model-server:v2.1.0

# Network configuration for ML build VM
network:
  network: ml-vpc
  subnet: ml-subnet

# ML-optimized settings
advanced:
  timeout: 2h  # Long timeout for large ML images
  job_name: ml-cache-build

# Authentication
auth:
  service_account: ml-cache-builder@ml-platform-project.iam.gserviceaccount.com
  image_pull_auth: ServiceAccountToken
`
