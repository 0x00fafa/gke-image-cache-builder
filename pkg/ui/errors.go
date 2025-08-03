package ui

import (
	"fmt"
	"strings"
)

// ErrorHandler provides context-aware error messages and solutions
type ErrorHandler struct {
	toolInfo *ToolInfo
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		toolInfo: GetToolInfo(),
	}
}

// HandleConfigError provides helpful error messages with solutions
func (e *ErrorHandler) HandleConfigError(err error) {
	errorMsg := err.Error()
	switch {
	case strings.Contains(errorMsg, "configuration file not found"):
		e.showConfigFileNotFoundError(err)
	case strings.Contains(errorMsg, "failed to parse YAML"):
		e.showYAMLParseError(err)
	case strings.Contains(errorMsg, "configuration validation failed"):
		e.showConfigValidationError(err)
	case strings.Contains(errorMsg, "execution mode"):
		e.showExecutionModeError()
	case strings.Contains(errorMsg, "zone") && strings.Contains(errorMsg, "required"):
		e.showZoneRequiredError()
	case strings.Contains(errorMsg, "container environments") || strings.Contains(errorMsg, "local mode"):
		e.showLocalModeEnvironmentError()
	case strings.Contains(errorMsg, "project-name"):
		e.showProjectNameError()
	case strings.Contains(errorMsg, "disk-image-name"):
		e.showDiskImageNameError()
	case strings.Contains(errorMsg, "container-image"):
		e.showContainerImageError()
	case strings.Contains(errorMsg, "invalid machine type"):
		e.showMachineTypeError(err)
	case strings.Contains(errorMsg, "invalid disk type"):
		e.showDiskTypeError(err)
	case strings.Contains(errorMsg, "container runtime"):
		e.showContainerRuntimeError(err)
	default:
		e.showGenericError(err)
	}
}

func (e *ErrorHandler) showContainerRuntimeError(err error) {
	fmt.Printf(`Error: Container runtime check failed
%v

SOLUTIONS:
    1. Install containerd:
       sudo apt update && sudo apt install containerd
       sudo systemctl start containerd
       
    2. Install Docker:
       curl -fsSL https://get.docker.com -o get-docker.sh
       sudo sh get-docker.sh
       
    3. Use remote mode instead:
       %s -R --zone=us-west1-b --project-name=<PROJECT> --disk-image-name=<NAME> --container-image=<IMAGE>

VERIFICATION:
    # Check containerd
    sudo ctr version
    
    # Check Docker
    docker version

For remote mode help: %s --help-examples`, err, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showConfigFileNotFoundError(err error) {
	fmt.Printf(`Error: Configuration file not found
%v

SOLUTIONS:
    1. Check the file path and ensure the file exists
    2. Generate a configuration template:
       %s --generate-config basic --output my-config.yaml
    3. Use command line parameters instead:
       %s -L --project-name=<PROJECT> --disk-image-name=<NAME> --container-image=<IMAGE>

EXAMPLES:
    # Generate and use a basic template
    %s --generate-config basic --output web-app.yaml
    %s --config web-app.yaml

For configuration help: %s --help-config`, err, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName,
		e.toolInfo.ExecutableName, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showYAMLParseError(err error) {
	fmt.Printf(`Error: YAML configuration file parsing failed
%v

SOLUTIONS:
    1. Check YAML syntax (indentation, colons, quotes)
    2. Validate the configuration file:
       %s --validate-config <CONFIG_FILE>
    3. Generate a new template:
       %s --generate-config basic --output new-config.yaml

COMMON YAML ISSUES:
    • Incorrect indentation (use spaces, not tabs)
    • Missing colons after keys
    • Unquoted special characters
    • Inconsistent list formatting

EXAMPLE VALID YAML:
    execution:
      mode: local
    project:
      name: my-project
    images:
      - nginx:latest
      - redis:alpine

For configuration help: %s --help-config`, err, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showConfigValidationError(err error) {
	fmt.Printf(`Error: Configuration validation failed
%v

SOLUTIONS:
    1. Check required fields in your configuration file
    2. Validate configuration syntax:
       %s --validate-config <CONFIG_FILE>
    3. Review configuration examples:
       %s --help-config
    4. Generate a working template:
       %s --generate-config basic

REQUIRED CONFIGURATION:
    execution.mode: local or remote
    project.name: your-gcp-project
    disk.name: your-disk-image-name
    images: [list of container images]

For configuration help: %s --help-config`, err, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName,
		e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showExecutionModeError() {
	fmt.Printf(`Error: Execution mode required

SOLUTION:
    Choose exactly one execution mode:
    
    LOCAL MODE (-L):  Execute on current GCP VM
    • Cost-effective (no additional VM charges)
    • Faster execution (no VM startup time)
    • Requires current machine to be a GCP VM
    • Requires containerd or Docker installed
    
    REMOTE MODE (-R): Create temporary GCP VM  
    • Works from any machine
    • Additional VM charges apply (~$0.38/build)
    • Requires --zone parameter

EXAMPLES:
    # Local mode (on GCP VM)
    %s -L --project-name=my-project --disk-image-name=web-cache --container-image=nginx:latest
    
    # Remote mode (from anywhere)
    %s -R --zone=us-west1-b --project-name=my-project --disk-image-name=web-cache --container-image=nginx:latest

Run '%s --help' for more information.`, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showZoneRequiredError() {
	fmt.Printf(`Error: Zone required for remote mode (-R)

SOLUTION:
    Specify a GCP zone with --zone parameter
    
    Available zones: us-west1-b, us-central1-a, europe-west1-b, asia-east1-a
    
    EXAMPLE:
    %s -R --zone=us-west1-b --project-name=my-project --disk-image-name=my-cache --container-image=nginx:latest

TIP: Use 'gcloud compute zones list' to see all available zones`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showLocalModeEnvironmentError() {
	fmt.Printf(`Error: Local mode (-L) environment check failed

POSSIBLE CAUSES:
    • Running in a container environment (Docker, Kubernetes)
    • Not running on a GCP VM instance
    • Network connectivity issues to GCP metadata server

SOLUTIONS:
    1. Use remote mode instead:
       %s -R --zone=us-west1-b --project-name=<PROJECT> --disk-image-name=<NAME> --container-image=<IMAGE>
       
    2. Run this command on a GCP VM instance
    
    3. Use Google Cloud Shell:
       https://shell.cloud.google.com

ENVIRONMENT DETECTION:
    This tool detected it's not running in a suitable environment for local mode.
    Local mode requires execution on a GCP VM instance with access to the metadata server.`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showProjectNameError() {
	fmt.Printf(`Error: GCP project name required

SOLUTION:
    Specify your GCP project with --project-name parameter
    
    EXAMPLES:
    %s -L --project-name=my-gcp-project --disk-image-name=web-cache --container-image=nginx:latest
    %s -R --zone=us-west1-b --project-name=production-project --disk-image-name=app-cache --container-image=node:16

TIP: Use 'gcloud config get-value project' to see your current project`, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showDiskImageNameError() {
	fmt.Printf(`Error: Disk image name required

SOLUTION:
    Specify a name for your disk image with --disk-image-name parameter
    
    Disk image name should be:
    • Descriptive of the cached images
    • Unique within your project
    • Follow GCP naming conventions (lowercase, hyphens)
    
    EXAMPLES:
    --disk-image-name=web-app-cache          # For web application images
    --disk-image-name=ml-models-cache        # For ML model images  
    --disk-image-name=microservices-cache    # For microservices stack
    --disk-image-name=team-a-cache-v1.2.0    # With version/team info

FULL EXAMPLE:
    %s -L --project-name=my-project --disk-image-name=web-app-cache --container-image=nginx:latest`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showContainerImageError() {
	fmt.Printf(`Error: At least one container image required

SOLUTION:
    Specify container images to cache with --container-image parameter
    You can specify multiple images by repeating the parameter
    
    SUPPORTED REGISTRIES:
    • Docker Hub: nginx:latest, node:16-alpine
    • Google Container Registry: gcr.io/my-project/app:v1.0
    • Artifact Registry: us-docker.pkg.dev/my-project/repo/app:latest
    • Private registries with authentication
    
    AUTHENTICATION OPTIONS:
    • None: Public images (default)
    • ServiceAccountToken: GCP registries with service account
    • DockerConfig: Use Docker configuration file
    • BasicAuth: Username/password via environment variables
    
    EXAMPLES:
    # Single image
    --container-image=nginx:latest
    
    # Multiple images
    --container-image=nginx:latest --container-image=redis:alpine --container-image=postgres:13
    
    # With authentication
    --image-pull-auth=ServiceAccountToken --container-image=gcr.io/my-project/app:latest
    
    FULL EXAMPLE:
    %s -L --project-name=my-project --disk-image-name=web-app-cache --container-image=nginx:latest`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showMachineTypeError(err error) {
	fmt.Printf(`Error: Invalid machine type
%v

SOLUTIONS:
    Use a supported machine type in your configuration or command line:
    
    SUPPORTED MACHINE TYPES:
    • e2-standard-2, e2-standard-4, e2-standard-8, e2-standard-16
    • e2-highmem-2, e2-highmem-4, e2-highmem-8, e2-highmem-16  
    • e2-highcpu-2, e2-highcpu-4, e2-highcpu-8, e2-highcpu-16
    • n1-standard-1, n1-standard-2, n1-standard-4, n1-standard-8
    • n2-standard-2, n2-standard-4, n2-standard-8, n2-standard-16

EXAMPLES:
    # Command line
    --machine-type=e2-standard-4
    
    # Configuration file
    advanced:
      machine_type: e2-standard-4

For configuration help: %s --help-config`, err, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showDiskTypeError(err error) {
	fmt.Printf(`Error: Invalid disk type
%v

SOLUTIONS:
    Use a supported disk type in your configuration or command line:
    
    SUPPORTED DISK TYPES:
    • pd-standard  (Standard persistent disk - cost-effective)
    • pd-ssd       (SSD persistent disk - high performance)
    • pd-balanced  (Balanced persistent disk - good performance/cost ratio)

EXAMPLES:
    # Command line
    --disk-type=pd-ssd
    
    # Configuration file
    disk:
      disk_type: pd-ssd

For configuration help: %s --help-config`, err, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showGenericError(err error) {
	fmt.Printf(`Error: %v

QUICK HELP:
    %s {-L|-R} --project-name=<PROJECT> --disk-image-name=<NAME> \
        --container-image=<IMAGE>
    
    Required parameters:
    • Execution mode: -L (local) or -R (remote)  
    • --project-name: Your GCP project
    • --disk-image-name: Name for the disk image
    • --container-image: Images to cache (repeatable)
    
    Additional for remote mode:
    • --zone: GCP zone (e.g., us-west1-b)
    
    For detailed help: %s --help
For examples: %s --help-examples`, err, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

// ShowNoArgsHelp displays help when no arguments are provided
func ShowNoArgsHelp() {
	toolInfo := GetToolInfo()
	fmt.Printf(`%s v2.0
%s

Missing required arguments. Quick start:

LOCAL MODE (on GCP VM):
    %s -L --project-name=<PROJECT> --disk-image-name=<NAME> \
        --container-image=<IMAGE>

REMOTE MODE (from anywhere):
    %s -R --zone=<ZONE> --project-name=<PROJECT> \
        --disk-image-name=<NAME> --container-image=<IMAGE>

EXAMPLES:
    %s -L --project-name=my-project --disk-image-name=web-cache --container-image=nginx:latest
    %s -R --zone=us-west1-b --project-name=my-project --disk-image-name=app-cache --container-image=node:16

AUTHENTICATION OPTIONS:
    --image-pull-auth=None                    # Public images (default)
    --image-pull-auth=ServiceAccountToken     # GCP registries
    --image-pull-auth=DockerConfig           # Docker config file
    --image-pull-auth=BasicAuth              # Environment variables

For detailed help: %s --help
For examples: %s --help-examples`, toolInfo.DisplayName, toolInfo.Purpose,
		toolInfo.ExecutableName, toolInfo.ExecutableName,
		toolInfo.ExecutableName, toolInfo.ExecutableName,
		toolInfo.ExecutableName, toolInfo.ExecutableName)
}
