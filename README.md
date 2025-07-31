# GKE Image Cache Builder

Build container image cache disks for GKE node acceleration.

## Purpose

Accelerate pod startup by pre-caching container images at the disk level, eliminating image pull latency for GKE workloads.

## Quick Start

### Local Mode (on GCP VM)
\`\`\`bash
gke-image-cache-builder -L --project-name=my-project \
    --disk-image-name=web-cache \
    --container-image=nginx:latest \
    --container-image=redis:alpine
\`\`\`

### Remote Mode (from anywhere)
\`\`\`bash
gke-image-cache-builder -R --zone=us-west1-b \
    --project-name=my-project \
    --disk-image-name=web-cache \
    --container-image=nginx:latest \
    --container-image=redis:alpine
\`\`\`

## Installation

### Download Binary
\`\`\`bash
# Linux
curl -L https://github.com/ai-on-gke/tools/releases/latest/download/gke-image-cache-builder-linux-amd64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder

# macOS
curl -L https://github.com/ai-on-gke/tools/releases/latest/download/gke-image-cache-builder-darwin-amd64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder
\`\`\`

### Build from Source
\`\`\`bash
git clone https://github.com/ai-on-gke/tools.git
cd tools/gke-image-cache-builder
make build
\`\`\`

## Usage

\`\`\`
gke-image-cache-builder {-L|-R} --project-name <PROJECT> --disk-image-name <NAME> [OPTIONS]

EXECUTION MODE (Required):
  -L, --local-mode     Execute on current GCP VM (cost-effective)
  -R, --remote-mode    Create temporary GCP VM (works anywhere)

REQUIRED:
  --project-name <PROJECT>      GCP project name
  --disk-image-name <NAME>      Name for the disk image
  --container-image <IMAGE>     Container image to cache (repeatable)

COMMON OPTIONS:
  -z, --zone <ZONE>            GCP zone (required for -R mode)
  -s, --cache-size <GB>        Cache disk size in GB (default: 10)
  -t, --timeout <DURATION>     Build timeout (default: 20m)
\`\`\`

## Benefits

🚀 **Eliminate image pull wait time** - 0s pod startup  
💰 **Reduce registry bandwidth costs** - Images cached locally  
⚡ **Improve scaling responsiveness** - Faster horizontal pod autoscaling  
🔄 **Reuse across nodes** - Cache disks can be attached to multiple GKE nodes  

## Examples

See `gke-image-cache-builder --help-examples` for detailed usage scenarios.

## License

Apache 2.0
