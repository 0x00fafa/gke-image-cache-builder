package ui

import (
	"fmt"
	"os"
	"text/template"
)

// ShowHelp displays the appropriate help message
func ShowHelp(helpType string, version string) {
	toolInfo := GetToolInfo()
	toolInfo.Version = version

	var tmplStr string
	switch helpType {
	case "full":
		tmplStr = fullHelpTemplate
	case "examples":
		tmplStr = examplesHelpTemplate
	case "quick":
		tmplStr = quickHelpTemplate
	default:
		tmplStr = basicHelpTemplate
	}

	tmpl := template.Must(template.New("help").Parse(tmplStr))
	if err := tmpl.Execute(os.Stdout, toolInfo); err != nil {
		fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
	}
}

// ShowVersionInfo displays version and tool information
func ShowVersionInfo(version, buildTime, gitCommit string) {
	toolInfo := GetToolInfo()

	fmt.Printf("%s v%s\n", toolInfo.DisplayName, version)
	fmt.Printf("Build: %s\n", buildTime)
	if gitCommit != "" {
		fmt.Printf("Commit: %s\n", gitCommit)
	}
	fmt.Printf("\n%s\n", toolInfo.Purpose)
	fmt.Printf("\nQuick start: %s {-L|-R} --project-name=<PROJECT> --image-name=<NAME> --gcs-path=<GCS_PATH> --container-image=<IMAGE>\n", toolInfo.ExecutableName)
	fmt.Printf("Help: %s --help | --help-examples | --help-full\n", toolInfo.ExecutableName)
}

// ToolInfo holds comprehensive information about the tool
type ToolInfo struct {
	ExecutableName string
	DisplayName    string
	Description    string
	Purpose        string
	TechnicalDesc  string
	ShortDesc      string
	Version        string
}

// GetToolInfo returns tool information
func GetToolInfo() *ToolInfo {
	return &ToolInfo{
		ExecutableName: "gke-image-cache-builder",
		DisplayName:    "GKE Image Cache Builder",
		Description:    "Build disk images with pre-cached container images for GKE node acceleration",
		Purpose:        "Accelerate pod startup by eliminating image pull latency",
		TechnicalDesc:  "Creates containerd-compatible disk images for GKE clusters",
		ShortDesc:      "GKE image cache builder",
	}
}

const basicHelpTemplate = `{{.DisplayName}} v{{.Version}}
{{.Description}}

PURPOSE:
    {{.Purpose}}
    
    ┌─ Container Images ─┐    ┌─ Disk Image ─┐    ┌─ GKE Node ─┐
    │ nginx:latest       │ ──▶│ Pre-cached   │ ──▶│ Instant    │
    │ redis:alpine       │    │ Images       │    │ Pod Start  │
    │ postgres:13        │    │              │    │            │
    └────────────────────┘    └──────────────┘    └────────────┘

USAGE:
    {{.ExecutableName}} {-L|-R} --project-name <PROJECT> --image-name <NAME> --gcs-path <GCS_PATH> [OPTIONS]

EXECUTION MODE (Required):
    -L, --local-mode     Execute on current GCP VM (cost-effective)
    -R, --remote-mode    Create temporary GCP VM (works anywhere)

REQUIRED:
    --project-name <PROJECT>      GCP project name
    --image-name <NAME>           Name for the disk image
    --gcs-path <GCS_PATH>         GCS path for build logs
    --container-image <IMAGE>     Container image to cache (repeatable)

COMMON OPTIONS:
    -z, --zone <ZONE>            GCP zone (required for -R mode)
    --disk-size-gb <SIZE>        Disk size in GB (default: 20)
    -t, --timeout <DURATION>     Build timeout (default: 20m)
    -h, --help                   Show this help
        --help-full              Show all options
        --help-examples          Show usage examples

QUICK START:
    # Cache web application images locally
    {{.ExecutableName}} -L --project-name=my-project \
        --image-name=web-app-cache --gcs-path=gs://bucket/logs \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine

    # Cache ML model images remotely
    {{.ExecutableName}} -R --project-name=ml-platform --zone=us-west1-b \
        --image-name=ml-models-cache --disk-size-gb=50 --gcs-path=gs://bucket/logs \
        --container-image=tensorflow/tensorflow:2.8.0-gpu

BENEFITS:
    🚀 Eliminate image pull wait time (0s pod startup)
    💰 Reduce container registry bandwidth costs
    ⚡ Improve application scaling responsiveness  
    🔄 Reuse disk images across multiple GKE nodes

Run '{{.ExecutableName}} --help-examples' for detailed scenarios.`

const fullHelpTemplate = `{{.DisplayName}} v{{.Version}} - Complete Reference
{{.Description}}

TECHNICAL OVERVIEW:
    {{.TechnicalDesc}}

PURPOSE:
    {{.Purpose}}

USAGE:
    {{.ExecutableName}} {-L|-R} --project-name <PROJECT> --image-name <NAME> --gcs-path <GCS_PATH> [OPTIONS]

EXECUTION MODE (Required - choose exactly one):
    -L, --local-mode             Execute on current GCP VM instance
                                • Cost-effective (no additional VM charges)
                                • Requires current machine to be a GCP VM
                                • Faster execution (no VM startup time)
                                • Automatic zone detection
                                
    -R, --remote-mode            Create temporary GCP VM for execution
                                • Works from any machine (local, CI/CD, etc.)
                                • Additional VM charges apply (~$0.38/build)
                                • Automatic resource cleanup
                                • Requires explicit zone specification

REQUIRED OPTIONS:
    --project-name <PROJECT>     GCP project name
    --image-name <NAME>          Name for the generated disk image
    --gcs-path <GCS_PATH>        GCS path for build logs
    --container-image <IMAGE>    Container image to pre-cache
                                Can be specified multiple times
                                Supports: Docker Hub, GCR, Artifact Registry

ZONE & LOCATION:
    -z, --zone <ZONE>           GCP zone (required for -R mode)
                               Auto-detected in -L mode
                               Examples: us-west1-b, europe-west1-c
    -n, --network <NETWORK>     VPC network (default: default)
    -u, --subnet <SUBNET>       Subnet (default: default)

DISK CONFIGURATION:
    --disk-size-gb <SIZE>       Disk size in GB (default: 20)
                               Min: 10GB, Max: 1000GB
                               Consider image sizes + 20% overhead
    -t, --timeout <DURATION>    Build timeout (default: 20m)
                               Increase for large images or slow networks
                               Examples: 30m, 1h, 90s

IMAGE MANAGEMENT:
    --image-family-name <FAMILY> Image family name (default: gke-disk-image)
    --image-labels <KEY=VALUE>   Image labels (repeatable)
                                Example: --image-labels env=prod
    --image-pull-auth <TYPE>     Image pull behavior
                                Options: None (default), ServiceAccountToken

AUTHENTICATION:
    --gcp-oauth <PATH>          Path to GCP service account credential file
    --service-account <EMAIL>   Service account email (default: default)

ADVANCED OPTIONS:
    --job-name <NAME>          Build job name (default: disk-image-build)

HELP & INFORMATION:
    -h, --help                Show basic help
        --help-full           Show this complete help
        --help-examples       Show usage examples and scenarios
        --version             Show version and build information

For examples and best practices: {{.ExecutableName}} --help-examples`

const examplesHelpTemplate = `{{.DisplayName}} - Usage Examples & Scenarios

═══════════════════════════════════════════════════════════════════════════════

🏠 LOCAL MODE EXAMPLES (Execute on GCP VM)

Basic web application cache:
    {{.ExecutableName}} -L --project-name=my-project \
        --image-name=web-stack-cache --gcs-path=gs://my-bucket/logs \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine \
        --container-image=postgres:13

Microservices application cache:
    {{.ExecutableName}} -L --project-name=production \
        --image-name=microservices-cache --gcs-path=gs://prod-logs/cache-builds \
        --disk-size-gb=30 --timeout=45m \
        --image-labels=env=production \
        --image-labels=team=platform \
        --container-image=gcr.io/my-project/api-gateway:v2.1.0 \
        --container-image=gcr.io/my-project/user-service:v1.8.3 \
        --container-image=gcr.io/my-project/order-service:v1.5.2 \
        --container-image=gcr.io/my-project/payment-service:v2.0.1

Large application with custom configuration:
    {{.ExecutableName}} -L --project-name=enterprise \
        --image-name=enterprise-app-cache --gcs-path=gs://enterprise-logs/image-cache \
        --disk-size-gb=100 --timeout=2h \
        --image-family-name=enterprise-cache \
        --container-image=gcr.io/enterprise/app:latest \
        --container-image=gcr.io/enterprise/worker:latest \
        --container-image=gcr.io/enterprise/scheduler:latest

═══════════════════════════════════════════════════════════════════════════════

☁️  REMOTE MODE EXAMPLES (Create temporary VM)

Basic usage from local development machine:
    {{.ExecutableName}} -R --project-name=my-project \
        --zone=us-west1-b \
        --image-name=dev-cache --gcs-path=gs://dev-bucket/logs \
        --container-image=nginx:latest \
        --container-image=node:16-alpine

CI/CD pipeline integration:
    {{.ExecutableName}} -R --project-name=$GCP_PROJECT \
        --zone=us-central1-a \
        --image-name=ci-cache-$BUILD_ID --gcs-path=gs://$GCS_BUCKET/ci-logs \
        --timeout=30m \
        --image-labels=build-id=$BUILD_ID \
        --image-labels=branch=$GIT_BRANCH \
        --container-image=gcr.io/$GCP_PROJECT/app:$GIT_SHA

High-performance build with custom VM:
    {{.ExecutableName}} -R --project-name=ml-platform \
        --zone=us-west1-b \
        --image-name=ml-models-cache --gcs-path=gs://ml-logs/cache-builds \
        --disk-size-gb=200 --timeout=3h \
        --container-image=tensorflow/tensorflow:2.8.0-gpu \
        --container-image=pytorch/pytorch:1.11.0-cuda11.3-cudnn8-runtime \
        --container-image=gcr.io/ml-platform/custom-model:v3.2.0

Secure private registry access:
    {{.ExecutableName}} -R --project-name=secure-project \
        --zone=europe-west1-b \
        --image-name=private-app-cache --gcs-path=gs://secure-logs/cache \
        --network=private-vpc --subnet=secure-subnet \
        --service-account=cache-builder@secure-project.iam.gserviceaccount.com \
        --image-pull-auth=ServiceAccountToken \
        --container-image=gcr.io/secure-project/private-app:latest

═══════════════════════════════════════════════════════════════════════════════

Need more help? 
• Documentation: https://github.com/0x00fafa/gke-image-cache-builder
• Issues: https://github.com/0x00fafa/gke-image-cache-builder/issues
• Discussions: https://github.com/0x00fafa/gke-image-cache-builder/discussions`

const quickHelpTemplate = `{{.ExecutableName}} - Quick Reference

USAGE: {{.ExecutableName}} {-L|-R} --project-name <PROJECT> --image-name <NAME> --gcs-path <GCS_PATH> --container-image <IMAGE>

MODES:
  -L  Local mode (on GCP VM)    -R  Remote mode (create VM)

EXAMPLES:
  {{.ExecutableName}} -L --project-name=my-project --image-name=web-cache --gcs-path=gs://bucket/logs --container-image=nginx:latest
  {{.ExecutableName}} -R --zone=us-west1-b --project-name=my-project --image-name=app-cache --gcs-path=gs://bucket/logs --container-image=node:16

More help: {{.ExecutableName}} --help | --help-full | --help-examples`
