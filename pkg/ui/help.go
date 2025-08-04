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
    {{.ExecutableName}} {-L|-R} --project-name <PROJECT> --disk-image-name <NAME> [OPTIONS]
    {{.ExecutableName}} --config <CONFIG_FILE> [OPTIONS]

EXECUTION MODE (Required):
    -L, --local-mode     Execute on current GCP VM (cost-effective)
    -R, --remote-mode    Create temporary GCP VM (works anywhere)

CONFIGURATION:
    -c, --config <FILE>          Use YAML configuration file
        --generate-config <TYPE> Generate config template (basic|advanced|ci-cd|ml)
        --output <PATH>          Output path for generated config
        --validate-config <FILE> Validate YAML configuration file

REQUIRED:
    --project-name <PROJECT>      GCP project name
    --disk-image-name <NAME>      Name for the disk image
    --container-image <IMAGE>     Container image to cache (repeatable)

COMMON OPTIONS:
    -z, --zone <ZONE>            GCP zone (required for -R mode)
    -s, --disk-size <GB>         Disk size in GB (default: 10)
    -t, --timeout <DURATION>     Build timeout (default: 20m)
    -h, --help                   Show this help
        --help-full              Show all options
        --help-examples          Show usage examples
        --help-config            Show configuration file help

ADVANCED OPTIONS:
    --job-name <NAME>            Build job name
    --machine-type <TYPE>        VM machine type (default: e2-standard-2)
    --preemptible                Use preemptible VM (cost savings)
    --disk-type <TYPE>           Cache disk type (default: pd-standard)
    --ssh-public-key <PATH>      SSH public key for remote VM access

NETWORK OPTIONS (Remote Mode Only):
    -n, --network <NETWORK>      VPC network for temporary VM (default: default)
    -u, --subnet <SUBNET>        Subnet for temporary VM (default: default)
                                 Note: These settings only affect the build VM,
                                 not the final disk image

IMAGE MANAGEMENT:
    --disk-family <FAMILY>       Image family name (default: gke-image-cache)
    --disk-labels <KEY=VALUE>    Disk labels (repeatable)
                                 Example: --disk-labels env=prod
    --image-pull-policy <POLICY> Image pull behavior
                                 Options: Always, IfNotPresent (default)

QUICK START:
    # Generate a configuration template
    {{.ExecutableName}} --generate-config basic --output web-app.yaml
    
    # Use configuration file
    {{.ExecutableName}} --config web-app.yaml
    
    # Mix config file with command line (CLI overrides config)
    {{.ExecutableName}} --config base.yaml --project-name=override-project

    # Traditional command line approach
    {{.ExecutableName}} -L --project-name=my-project \
        --disk-image-name=web-app-cache \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine

BENEFITS:
    🚀 Eliminate image pull wait time (0s pod startup)
    💰 Reduce container registry bandwidth costs
    ⚡ Improve application scaling responsiveness  
    🔄 Reuse cache disks across multiple GKE nodes

Run '{{.ExecutableName}} --help-config' for configuration file details.`

const examplesHelpTemplate = `{{.DisplayName}} - Usage Examples & Scenarios

═══════════════════════════════════════════════════════════════════════════════

🏠 LOCAL MODE EXAMPLES (Execute on GCP VM)

Basic web application cache:
    {{.ExecutableName}} -L --project-name=my-project \
        --disk-image-name=web-stack-cache \
        --container-image=nginx:1.21 \
        --container-image=redis:6.2-alpine \
        --container-image=postgres:13

Microservices application cache:
    {{.ExecutableName}} -L --project-name=production \
        --disk-image-name=microservices-cache \
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
        --disk-image-name=dev-cache \
        --container-image=nginx:latest \
        --container-image=node:16-alpine

CI/CD pipeline integration:
    {{.ExecutableName}} -R --project-name=$GCP_PROJECT \
        --zone=us-central1-a \
        --disk-image-name=ci-cache-$BUILD_ID \
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

Need more help? Visit: https://github.com/0x00fafa/gke-image-cache-builder`

const configHelpTemplate = `{{.DisplayName}} - Configuration File Guide

═══════════════════════════════════════════════════════════════════════════════

📁 CONFIGURATION FILE SUPPORT

The tool supports YAML configuration files to simplify complex builds and enable
configuration reuse across environments.

PRIORITY ORDER (highest to lowest):
    1. Command line parameters
    2. Environment variables  
    3. Configuration file values
    4. Default values

═══════════════════════════════════════════════════════════════════════════════

🛠️ GENERATING CONFIGURATION TEMPLATES

Generate different types of configuration templates:

    # Basic template (minimal configuration)
    {{.ExecutableName}} --generate-config basic --output basic.yaml
    
    # Advanced template (all options)
    {{.ExecutableName}} --generate-config advanced --output advanced.yaml
    
    # CI/CD optimized template
    {{.ExecutableName}} --generate-config ci-cd --output ci-cd.yaml
    
    # ML/AI workloads template
    {{.ExecutableName}} --generate-config ml --output ml.yaml

═══════════════════════════════════════════════════════════════════════════════

📝 BASIC CONFIGURATION EXAMPLE

# web-app.yaml
execution:
  mode: local  # or remote
  zone: us-west1-b  # required for remote mode

project:
  name: my-project

disk:
  name: web-app-cache
  size_gb: 20
  family: web-cache
  labels:
    env: production
    team: platform

images:
  - nginx:1.21
  - redis:6.2-alpine
  - postgres:13

Usage: {{.ExecutableName}} --config web-app.yaml

═══════════════════════════════════════════════════════════════════════════════

🔧 ADVANCED CONFIGURATION EXAMPLE

# production.yaml
execution:
  mode: remote
  zone: us-west1-b

project:
  name: production-project

disk:
  name: microservices-cache
  size_gb: 50
  family: production-cache
  disk_type: pd-ssd
  labels:
    env: production
    version: v2.1.0

images:
  - gcr.io/my-project/api:v2.1.0
  - gcr.io/my-project/worker:v2.1.0
  - nginx:1.21
  - redis:6.2-alpine

# Network settings for temporary build VM (remote mode only)
# These settings do NOT affect the final disk image
network:
  network: production-vpc    # VPC for build VM
  subnet: production-subnet  # Subnet for build VM

advanced:
  timeout: 45m
  machine_type: e2-standard-4
  preemptible: true

auth:
  service_account: cache-builder@production.iam.gserviceaccount.com
  image_pull_auth: ServiceAccountToken

logging:
  verbose: true

Usage: {{.ExecutableName}} --config production.yaml

═══════════════════════════════════════════════════════════════════════════════

🔄 MIXED USAGE (Config + Command Line)

Command line parameters override configuration file values:

    # Use config but override project and add extra image
    {{.ExecutableName}} --config base.yaml \
        --project-name=different-project \
        --container-image=additional:image

    # Use config but switch to local mode
    {{.ExecutableName}} --config remote.yaml -L

═══════════════════════════════════════════════════════════════════════════════

✅ VALIDATION AND TESTING

Validate configuration files before use:

    # Validate configuration syntax and values
    {{.ExecutableName}} --validate-config my-config.yaml
    
    # Test configuration with dry-run (if implemented)
    {{.ExecutableName}} --config my-config.yaml --dry-run

═══════════════════════════════════════════════════════════════════════════════

💡 BEST PRACTICES

1. **Environment-specific configs**: dev.yaml, staging.yaml, prod.yaml
2. **Version control**: Store configs in your repository
3. **Validation**: Always validate configs before use
4. **Documentation**: Add comments to explain complex configurations
5. **Security**: Don't store credentials in config files, use environment variables

═══════════════════════════════════════════════════════════════════════════════

🔗 COMPLETE CONFIGURATION REFERENCE

All available configuration options:

execution:
  mode: local|remote           # Execution mode
  zone: <zone>                 # GCP zone

project:
  name: <project>              # GCP project name

disk:
  name: <name>                 # Disk image name
  size_gb: <size>              # Disk size (10-1000)
  family: <family>             # Image family
  disk_type: pd-standard|pd-ssd|pd-balanced
  labels:                      # Key-value labels
    key: value

images:                        # Container images list
  - image:tag
  - registry/image:tag

# Network settings for build VM only (remote mode)
# These do NOT affect the final disk image
network:
  network: <network>           # VPC network for build VM
  subnet: <subnet>             # Subnet for build VM

advanced:
  timeout: <duration>          # Build timeout (e.g., 30m, 1h)
  job_name: <name>             # Job name
  machine_type: <type>         # VM machine type
  preemptible: true|false      # Use preemptible instances
  ssh_public_key: <path>       # SSH public key for remote VM access

auth:
  gcp_oauth: <path>            # Service account file path
  service_account: <email>     # Service account email
  image_pull_auth: None|ServiceAccountToken

logging:
  verbose: true|false          # Verbose logging
  quiet: true|false            # Quiet mode

For more help: {{.ExecutableName}} --help-examples`

// ShowHelp displays the appropriate help message
func ShowHelp(helpType string, version string) {
	toolInfo := GetToolInfo()

	var tmplStr string
	switch helpType {
	case "examples":
		tmplStr = examplesHelpTemplate
	case "config":
		tmplStr = configHelpTemplate
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
	fmt.Printf("\nQuick start: %s {-L|-R} --project-name=<PROJECT> --disk-image-name=<NAME> --container-image=<IMAGE>\n", toolInfo.ExecutableName)
	fmt.Printf("Help: %s --help | --help-examples\n", toolInfo.ExecutableName)
}
