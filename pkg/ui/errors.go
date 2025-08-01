package ui

import (
	"fmt"
	"strings"

	"github.com/0x00fafa/gke-image-cache-builder/pkg/config"
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
	case strings.Contains(errorMsg, "image-name"):
		e.showImageNameError()
	case strings.Contains(errorMsg, "container-image"):
		e.showContainerImageError()
	case strings.Contains(errorMsg, "gcs-path"):
		e.showGCSPathError()
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
    %s -L --project-name=my-project --image-name=web-cache --gcs-path=gs://bucket/logs --container-image=nginx:latest
    
    # Remote mode (from anywhere)
    %s -R --zone=us-west1-b --project-name=my-project --image-name=web-cache --gcs-path=gs://bucket/logs --container-image=nginx:latest

Run '%s --help' for more information.
`, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showZoneRequiredError() {
	fmt.Printf(`Error: Zone required for remote mode (-R)

SOLUTION:
    Specify a GCP zone with --zone parameter
    
    Available zones: us-west1-b, us-central1-a, europe-west1-b, asia-east1-a
    
EXAMPLE:
    %s -R --zone=us-west1-b --project-name=my-project --image-name=my-cache --gcs-path=gs://bucket/logs --container-image=nginx:latest

TIP: Use 'gcloud compute zones list' to see all available zones
`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showLocalModeEnvironmentError() {
	fmt.Printf(`Error: Local mode (-L) requires execution on a GCP VM instance

CURRENT ENVIRONMENT: Not a GCP VM

SOLUTIONS:
    1. Use remote mode instead:
       %s -R --zone=us-west1-b --project-name=<PROJECT> --image-name=<NAME> --gcs-path=<GCS_PATH> --container-image=<IMAGE>
       
    2. Run this command on a GCP VM instance
    
    3. Use Google Cloud Shell:
       https://shell.cloud.google.com
       
    4. SSH to a GCP VM and run there:
       gcloud compute ssh my-vm --command="%s -L ..."

DETECTION: This tool detected it's not running on a GCP VM instance.
`, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showProjectNameError() {
	fmt.Printf(`Error: GCP project name required

SOLUTION:
    Specify your GCP project with --project-name parameter
    
EXAMPLES:
    %s -L --project-name=my-gcp-project --image-name=web-cache --gcs-path=gs://bucket/logs --container-image=nginx:latest
    %s -R --zone=us-west1-b --project-name=production-project --image-name=app-cache --gcs-path=gs://bucket/logs --container-image=node:16

TIP: Use 'gcloud config get-value project' to see your current project
`, e.toolInfo.ExecutableName, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showImageNameError() {
	fmt.Printf(`Error: Image name required

SOLUTION:
    Specify a name for your disk image with --image-name parameter
    
    Image name should be:
    • Descriptive of the cached images
    • Unique within your project
    • Follow GCP naming conventions (lowercase, hyphens)
    
EXAMPLES:
    --image-name=web-app-cache          # For web application images
    --image-name=ml-models-cache        # For ML model images  
    --image-name=microservices-cache    # For microservices stack
    --image-name=team-a-cache-v1.2.0    # With version/team info

FULL EXAMPLE:
    %s -L --project-name=my-project --image-name=web-app-cache --gcs-path=gs://bucket/logs --container-image=nginx:latest
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
    • Private registries: registry.company.com/app:latest
    
EXAMPLES:
    # Single image
    --container-image=nginx:latest
    
    # Multiple images
    --container-image=nginx:latest --container-image=redis:alpine --container-image=postgres:13
    
FULL EXAMPLE:
    %s -L --project-name=my-project --image-name=web-stack --gcs-path=gs://bucket/logs \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine \
        --container-image=postgres:13
`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showGCSPathError() {
	fmt.Printf(`Error: GCS path required

SOLUTION:
    Specify a GCS path for build logs with --gcs-path parameter
    
    GCS path should be:
    • A valid GCS bucket path
    • Accessible by your GCP credentials
    • Include gs:// prefix
    
EXAMPLES:
    --gcs-path=gs://my-bucket/logs          # Basic path
    --gcs-path=gs://my-bucket/builds/logs   # With subdirectory
    --gcs-path=gs://company-logs/gke-cache  # Organized path

FULL EXAMPLE:
    %s -L --project-name=my-project --image-name=web-cache --gcs-path=gs://my-bucket/logs --container-image=nginx:latest

TIP: Create bucket with: gsutil mb gs://my-bucket
`, e.toolInfo.ExecutableName)
}

func (e *ErrorHandler) showGenericError(err error) {
	fmt.Printf(`Error: %v

QUICK HELP:
    %s {-L|-R} --project-name=<PROJECT> --image-name=<NAME> --gcs-path=<GCS_PATH> --container-image=<IMAGE>
    
    Required parameters:
    • Execution mode: -L (local) or -R (remote)  
    • --project-name: Your GCP project
    • --image-name: Name for the disk image
    • --gcs-path: GCS path for logs
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
    %s -L --project-name=<PROJECT> --image-name=<NAME> --gcs-path=<GCS_PATH> \
        --container-image=<IMAGE>

REMOTE MODE (from anywhere):
    %s -R --zone=<ZONE> --project-name=<PROJECT> \
        --image-name=<NAME> --gcs-path=<GCS_PATH> --container-image=<IMAGE>

EXAMPLES:
    %s -L --project-name=my-project --image-name=web-cache --gcs-path=gs://bucket/logs --container-image=nginx:latest
    %s -R --zone=us-west1-b --project-name=my-project --image-name=app-cache --gcs-path=gs://bucket/logs --container-image=node:16

For detailed help: %s --help
For examples: %s --help-examples
For all options: %s --help-full
`, toolInfo.DisplayName, toolInfo.Purpose,
		toolInfo.ExecutableName, toolInfo.ExecutableName,
		toolInfo.ExecutableName, toolInfo.ExecutableName,
		toolInfo.ExecutableName, toolInfo.ExecutableName, toolInfo.ExecutableName)
}

// ShowEnvironmentInfo displays current environment information
func ShowEnvironmentInfo(envInfo *config.EnvironmentInfo) {
	fmt.Println("🌍 Environment Information")
	fmt.Println()
	fmt.Printf("Environment: %s\n", envInfo.GetEnvironmentDescription())
	fmt.Printf("Recommended mode: %s\n", envInfo.GetRecommendedMode().String())

	if len(envInfo.Restrictions) > 0 {
		fmt.Println()
		fmt.Println("⚠️  Restrictions:")
		for _, restriction := range envInfo.Restrictions {
			fmt.Printf("   • %s\n", restriction)
		}
	}

	fmt.Println()
	fmt.Println("📋 Compatibility Matrix:")
	fmt.Println("   Environment    | Local Mode | Remote Mode | Recommended")
	fmt.Println("   --------------|------------|-------------|------------")
	fmt.Println("   GCP VM        | ✅ Yes      | ✅ Yes       | Local (cost)")
	fmt.Println("   Container     | ❌ No       | ✅ Yes       | Remote (only)")
	fmt.Println("   Local Machine | ❌ No       | ✅ Yes       | Remote (safe)")
}
