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
	case strings.Contains(errorMsg, "execution mode"):
		e.showExecutionModeError()
	case strings.Contains(errorMsg, "zone") && strings.Contains(errorMsg, "required"):
		e.showZoneRequiredError()
	case strings.Contains(errorMsg, "GCP VM") || strings.Contains(errorMsg, "local mode"):
		e.showLocalModeEnvironmentError()
	case strings.Contains(errorMsg, "project-name"):
		e.showProjectNameError()
	case strings.Contains(errorMsg, "cache-name"):
		e.showCacheNameError()
	case strings.Contains(errorMsg, "container-image"):
		e.showContainerImageError()
	case strings.Contains(errorMsg, "disk-image-name"): // 修改错误匹配
		e.showDiskImageNameError() // 修改函数名
	default:
		e.showGenericError(err)
	}
}

func (e *ErrorHandler) showExecutionModeError() {
	fmt.Printf(`Error: Execution mode required

SOLUTION:
    Choose exactly one execution mode:
    
    LOCAL MODE (-L):  Execute on current GCP VM
    • Cost-effective (no additional VM charges)
    • Faster execution (no VM startup time)
    • Requires current machine to be a GCP VM
    
    REMOTE MODE (-R): Create temporary GCP VM  
    • Works from any machine
    • Additional VM charges apply (~$0.38/build)
    • Requires --zone parameter

EXAMPLES:
    # Local mode (on GCP VM)
    %s -L --project-name=my-project --disk-image-name=web-cache --container-image=nginx:latest
    
    # Remote mode (from anywhere)
    %s -R --zone=us-west1-b --project-name=my-project --disk-image-name=web-cache --container-image=nginx:latest

Run '%s --help' for more information.
`, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showZoneRequiredError() {
	fmt.Printf(`Error: Zone required for remote mode (-R)

SOLUTION:
    Specify a GCP zone with --zone parameter
    
    Available zones: us-west1-b, us-central1-a, europe-west1-b, asia-east1-a
    
EXAMPLE:
    %s -R --zone=us-west1-b --project-name=my-project --disk-image-name=my-cache --container-image=nginx:latest

TIP: Use 'gcloud compute zones list' to see all available zones
`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showLocalModeEnvironmentError() {
	fmt.Printf(`Error: Local mode (-L) requires execution on a GCP VM instance

CURRENT ENVIRONMENT: Not a GCP VM

SOLUTIONS:
    1. Use remote mode instead:
       %s -R --zone=us-west1-b --project-name=<PROJECT> --disk-image-name=<NAME> --container-image=<IMAGE>
       
    2. Run this command on a GCP VM instance
    
    3. Use Google Cloud Shell:
       https://shell.cloud.google.com

DETECTION: This tool detected it's not running on a GCP VM instance.
`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showProjectNameError() {
	fmt.Printf(`Error: GCP project name required

SOLUTION:
    Specify your GCP project with --project-name parameter
    
EXAMPLES:
    %s -L --project-name=my-gcp-project --disk-image-name=web-cache --container-image=nginx:latest
    %s -R --zone=us-west1-b --project-name=production-project --disk-image-name=app-cache --container-image=node:16

TIP: Use 'gcloud config get-value project' to see your current project
`, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showCacheNameError() {
	fmt.Printf(`Error: Cache name required

SOLUTION:
    Specify a name for your image cache disk with --cache-name parameter
    
    Cache name should be:
    • Descriptive of the cached images
    • Unique within your project
    • Follow GCP naming conventions (lowercase, hyphens)
    
EXAMPLES:
    --cache-name=web-app-cache          # For web application images
    --cache-name=ml-models-cache        # For ML model images  
    --cache-name=microservices-cache    # For microservices stack

FULL EXAMPLE:
    %s -L --project-name=my-project --cache-name=web-stack \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine \
        --container-image=postgres:13
`, e.toolInfo.ExecutableName)
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
    
EXAMPLES:
    # Single image
    --container-image=nginx:latest
    
    # Multiple images
    --container-image=nginx:latest --container-image=redis:alpine --container-image=postgres:13
    
FULL EXAMPLE:
    %s -L --project-name=my-project --disk-image-name=web-app-cache --container-image=nginx:latest
`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showDiskImageNameError() { // 修改函数名
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
    %s -L --project-name=my-project --disk-image-name=web-app-cache --container-image=nginx:latest
`, e.toolInfo.ExecutableName)
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
For examples: %s --help-examples
`, err, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
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

For detailed help: %s --help
For examples: %s --help-examples
`, toolInfo.DisplayName, toolInfo.Purpose,
		toolInfo.ExecutableName, toolInfo.ExecutableName,
		toolInfo.ExecutableName, toolInfo.ExecutableName,
		toolInfo.ExecutableName, toolInfo.ExecutableName)
}
