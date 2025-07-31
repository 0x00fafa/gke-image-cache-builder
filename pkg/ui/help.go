package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ToolInfo holds comprehensive information about the tool
type ToolInfo struct {
	ExecutableName string
	DisplayName    string
	Description    string
	Purpose        string
	TechnicalDesc  string
	ShortDesc      string
}

// GetToolInfo automatically detects and returns tool information
func GetToolInfo() *ToolInfo {
	execPath, err := os.Executable()
	if err != nil {
		return getDefaultToolInfo()
	}

	execName := filepath.Base(execPath)
	execName = strings.TrimSuffix(execName, ".exe")
	execName = strings.TrimSuffix(execName, ".bin")

	return analyzeToolName(execName)
}

// analyzeToolName provides context-aware tool information based on executable name
func analyzeToolName(name string) *ToolInfo {
	normalizedName := strings.ToLower(name)

	switch {
	case strings.Contains(normalizedName, "image-cache-builder"):
		return &ToolInfo{
			ExecutableName: name,
			DisplayName:    name + " (GKE Image Cache Builder)",
			Description:    "Build image cache disks for GKE node acceleration",
			Purpose:        "Accelerate pod startup by pre-caching container images",
			TechnicalDesc:  "Creates GKE-compatible disk images with containerd image cache",
			ShortDesc:      "GKE image cache builder",
		}

	case normalizedName == "gkeimg":
		return &ToolInfo{
			ExecutableName: name,
			DisplayName:    name + " (GKE Image Cache Builder)",
			Description:    "Build image cache disks for GKE nodes",
			Purpose:        "Pre-cache container images for faster pod startup",
			TechnicalDesc:  "GKE image cache disk builder (short form)",
			ShortDesc:      "GKE image cache builder",
		}

	case normalizedName == "imgcache":
		return &ToolInfo{
			ExecutableName: name,
			DisplayName:    name + " (Image Cache Builder)",
			Description:    "Build container image cache disks",
			Purpose:        "Pre-cache images for faster container deployment",
			TechnicalDesc:  "Container image cache disk builder",
			ShortDesc:      "Image cache builder",
		}

	default:
		return getDefaultToolInfo()
	}
}

func getDefaultToolInfo() *ToolInfo {
	return &ToolInfo{
		ExecutableName: "gke-image-cache-builder",
		DisplayName:    "gke-image-cache-builder (GKE Image Cache Builder)",
		Description:    "Build container image cache disks for GKE node acceleration",
		Purpose:        "Accelerate pod startup by eliminating image pull latency",
		TechnicalDesc:  "Creates containerd-compatible image cache disks for GKE clusters",
		ShortDesc:      "GKE image cache builder",
	}
}

const basicHelpTemplate = `{{.DisplayName}} v{{.Version}}
{{.Description}}

PURPOSE:
    {{.Purpose}}
    
    ┌─ Container Images ─┐    ┌─ Image Cache Disk ─┐    ┌─ GKE Node ─┐
    │ nginx:latest       │ ──▶│ Pre-cached Images  │ ──▶│ Instant    │
    │ redis:alpine       │    │ (containerd ready) │    │ Pod Start  │
    │ postgres:13        │    │                    │    │            │
    └────────────────────┘    └────────────────────┘    └────────────┘

USAGE:
    {{.ExecutableName}} {-L|-R} --project-name <PROJECT> --cache-name <NAME> [OPTIONS]

EXECUTION MODE (Required):
    -L, --local-mode     Execute on current GCP VM (cost-effective)
    -R, --remote-mode    Create temporary GCP VM (works anywhere)

REQUIRED:
    --project-name <PROJECT>      GCP project name
    --cache-name <NAME>           Name for the image cache disk
    --container-image <IMAGE>     Container image to cache (repeatable)

COMMON OPTIONS:
    -z, --zone <ZONE>            GCP zone (required for -R mode)
    -s, --cache-size <GB>        Cache disk size in GB (default: 10)
    -t, --timeout <DURATION>     Build timeout (default: 20m)
    -h, --help                   Show this help
        --help-full              Show all options
        --help-examples          Show usage examples

QUICK START:
    # Cache web application images locally
    {{.ExecutableName}} -L --project-name=my-project \
        --cache-name=web-app-cache \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine

    # Cache ML model images remotely
    {{.ExecutableName}} -R --project-name=ml-platform --zone=us-west1-b \
        --cache-name=ml-models-cache --cache-size=50 \
        --container-image=tensorflow/tensorflow:2.8.0-gpu

BENEFITS:
    🚀 Eliminate image pull wait time (0s pod startup)
    💰 Reduce container registry bandwidth costs
    ⚡ Improve application scaling responsiveness  
    🔄 Reuse cache disks across multiple GKE nodes

Run '{{.ExecutableName}} --help-examples' for detailed scenarios.`

const examplesHelpTemplate = `{{.DisplayName}} - Usage Examples & Scenarios

═══════════════════════════════════════════════════════════════════════════════

🏠 LOCAL MODE EXAMPLES (Execute on GCP VM)

Basic web application cache:
    {{.ExecutableName}} -L --project-name=my-project \
        --cache-name=web-stack-cache \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine \
        --container-image=postgres:13

Microservices application cache:
    {{.ExecutableName}} -L --project-name=production \
        --cache-name=microservices-cache \
        --cache-size=30 --timeout=45m \
        --cache-labels=env=production \
        --cache-labels=team=platform \
        --container-image=gcr.io/my-project/api-gateway:v2.1.0 \
        --container-image=gcr.io/my-project/user-service:v1.8.3

═══════════════════════════════════════════════════════════════════════════════

☁️  REMOTE MODE EXAMPLES (Create temporary VM)

Basic usage from local development machine:
    {{.ExecutableName}} -R --project-name=my-project \
        --zone=us-west1-b \
        --cache-name=dev-cache \
        --container-image=nginx:latest \
        --container-image=node:16-alpine

CI/CD pipeline integration:
    {{.ExecutableName}} -R --project-name=$GCP_PROJECT \
        --zone=us-central1-a \
        --cache-name=ci-cache-$BUILD_ID \
        --timeout=30m --preemptible \
        --cache-labels=build-id=$BUILD_ID \
        --container-image=gcr.io/$GCP_PROJECT/app:$GIT_SHA

═══════════════════════════════════════════════════════════════════════════════

💡 BEST PRACTICES & TIPS

Cost Optimization:
    • Use -L (local mode) when possible to avoid VM charges
    • Use --preemptible with -R mode for 60-80% cost savings
    • Choose appropriate --cache-size to avoid waste

Performance Optimization:
    • Use --timeout=30m or higher for images >5GB
    • Consider --machine-type=e2-standard-4 for faster builds
    • Group related images in single cache for efficiency

Need more help? Visit: https://github.com/ai-on-gke/tools/tree/main/gke-image-cache-builder`

// ShowHelp displays the appropriate help message
func ShowHelp(helpType string, version string) {
	toolInfo := GetToolInfo()

	var tmplStr string
	switch helpType {
	case "examples":
		tmplStr = examplesHelpTemplate
	default:
		tmplStr = basicHelpTemplate
	}

	tmpl := template.Must(template.New("help").Parse(tmplStr))

	data := struct {
		*ToolInfo
		Version string
	}{
		ToolInfo: toolInfo,
		Version:  version,
	}

	if err := tmpl.Execute(os.Stdout, data); err != nil {
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
	fmt.Printf("\nQuick start: %s {-L|-R} --project-name=<PROJECT> --cache-name=<NAME> --container-image=<IMAGE>\n", toolInfo.ExecutableName)
	fmt.Printf("Help: %s --help | --help-examples\n", toolInfo.ExecutableName)
}
