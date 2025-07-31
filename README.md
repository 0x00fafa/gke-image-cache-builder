# GKE Image Cache Builder

A modern, redesigned tool for building container image cache disks to accelerate GKE node startup performance.

## 🎯 Project Background

This project is **inspired by and builds upon** the original [gke-disk-image-builder](https://github.com/ai-on-gke/tools/tree/main/gke-disk-image-builder) from the ai-on-gke/tools repository. While the original project provided a solid foundation, we identified several areas for improvement and have completely redesigned the tool with modern Go practices and enhanced user experience.

## 🚀 Purpose

Accelerate pod startup by pre-caching container images at the disk level, eliminating image pull latency for GKE workloads. This approach can reduce pod startup time from minutes to seconds for large container images.

\`\`\`
┌─ Container Images ─┐    ┌─ Image Cache Disk ─┐    ┌─ GKE Node ─┐
│ nginx:latest       │ ──▶│ Pre-cached Images  │ ──▶│ Instant    │
│ redis:alpine       │    │ (containerd ready) │    │ Pod Start  │
│ postgres:13        │    │                    │    │            │
└────────────────────┘    └────────────────────┘    └────────────┘
\`\`\`

## 📊 Comparison with Original Project

### Original Project Limitations

After analyzing the original `gke-disk-image-builder`, we identified several key limitations:

| **Issue** | **Description** | **Impact** |
|-----------|-----------------|------------|
| 🏗️ **Cloud Build Dependency** | Required Google Cloud Build for execution | Limited flexibility, additional costs, complex setup |
| 📝 **Shell Script Architecture** | Monolithic bash scripts with limited error handling | Hard to maintain, debug, and extend |
| 🔧 **Limited Configuration** | Minimal customization options | Inflexible for different use cases |
| 📋 **Poor User Experience** | Confusing parameter names, limited help system | Steep learning curve, error-prone usage |
| 🚫 **No Local Execution** | Could only run in Cloud Build environment | Required cloud resources for every build |
| 📊 **Minimal Logging** | Basic logging with no progress indicators | Poor visibility into build process |
| 🔒 **Limited Auth Options** | Basic authentication support | Restricted registry access |

### Our Improvements

| **Category** | **Original** | **Our Implementation** | **Benefit** |
|--------------|--------------|------------------------|-------------|
| **Architecture** | Shell scripts | Modern Go with clean architecture | Maintainable, testable, extensible |
| **Execution Modes** | Cloud Build only | Local + Remote modes | Cost-effective, flexible deployment |
| **User Interface** | Basic CLI | Rich help system, context-aware errors | Better developer experience |
| **Configuration** | Limited options | Comprehensive configuration | Supports diverse use cases |
| **Logging** | Basic output | Structured logging with progress | Better observability |
| **Error Handling** | Minimal | Comprehensive error handling | Reliable operation |
| **Authentication** | Basic | Multiple auth mechanisms | Broader registry support |
| **Resource Management** | Manual | Automatic cleanup | Prevents resource leaks |

## ✨ New Features

### 🎯 **Dual Execution Modes**
- **Local Mode (-L)**: Execute on current GCP VM (cost-effective)
- **Remote Mode (-R)**: Create temporary GCP VM (works anywhere)

### 🛠️ **Enhanced Configuration**
- Comprehensive parameter validation
- Flexible timeout settings
- Custom machine types and disk configurations
- Advanced labeling and tagging

### 📚 **Rich Help System**
- Context-aware error messages with solutions
- Comprehensive examples and scenarios
- Multiple help levels (basic, full, examples)

### 🔐 **Advanced Authentication**
- Multiple registry authentication methods
- Service account token support
- Private registry access

### 📊 **Better Observability**
- Structured console logging
- Progress indicators
- Verbose debugging options
- Build status tracking

### 🧹 **Automatic Resource Management**
- Automatic cleanup of temporary resources
- Resource leak prevention
- Cost optimization features

## 🗑️ Removed Features

| **Feature** | **Reason for Removal** | **Alternative** |
|-------------|------------------------|-----------------|
| **Cloud Build Integration** | Added complexity and costs | Direct VM execution |
| **GCS Logging** | Over-engineering for most use cases | Console logging with optional verbosity |
| **Complex Shell Scripts** | Hard to maintain and debug | Clean Go implementation |
| **YAML Configuration Files** | Added unnecessary complexity | Command-line parameters |

## 🚀 Quick Start

### Prerequisites
- GCP project with Compute Engine API enabled
- Appropriate IAM permissions
- For local mode: Must run on a GCP VM instance

### Local Mode (Cost-Effective)
\`\`\`bash
# Execute on current GCP VM
gke-image-cache-builder -L \
    --project-name=my-project \
    --disk-image-name=web-cache \
    --container-image=nginx:latest \
    --container-image=redis:alpine
\`\`\`

### Remote Mode (Universal)
\`\`\`bash
# Execute from anywhere (creates temporary VM)
gke-image-cache-builder -R \
    --zone=us-west1-b \
    --project-name=my-project \
    --disk-image-name=web-cache \
    --container-image=nginx:latest \
    --container-image=redis:alpine
\`\`\`

## 📦 Installation

### Download Pre-built Binary
\`\`\`bash
# Linux AMD64
curl -L https://github.com/your-org/gke-image-cache-builder/releases/latest/download/gke-image-cache-builder-linux-amd64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder

# macOS AMD64
curl -L https://github.com/your-org/gke-image-cache-builder/releases/latest/download/gke-image-cache-builder-darwin-amd64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder

# macOS ARM64 (Apple Silicon)
curl -L https://github.com/your-org/gke-image-cache-builder/releases/latest/download/gke-image-cache-builder-darwin-arm64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder
\`\`\`

### Build from Source
\`\`\`bash
git clone https://github.com/your-org/gke-image-cache-builder.git
cd gke-image-cache-builder
make build-static
\`\`\`

### Using Go Install
\`\`\`bash
go install github.com/your-org/gke-image-cache-builder/cmd@latest
\`\`\`

## 📖 Usage

### Basic Syntax
\`\`\`
gke-image-cache-builder {-L|-R} --project-name <PROJECT> --disk-image-name <NAME> [OPTIONS]
\`\`\`

### Required Parameters
| Parameter | Description | Example |
|-----------|-------------|---------|
| `-L` or `-R` | Execution mode (Local/Remote) | `-L` |
| `--project-name` | GCP project name | `--project-name=my-project` |
| `--disk-image-name` | Name for the disk image | `--disk-image-name=web-cache` |
| `--container-image` | Container images to cache (repeatable) | `--container-image=nginx:latest` |

### Common Options
| Parameter | Description | Default | Example |
|-----------|-------------|---------|---------|
| `--zone` | GCP zone (required for -R mode) | Auto-detect (local) | `--zone=us-west1-b` |
| `--cache-size` | Cache disk size in GB | 10 | `--cache-size=50` |
| `--timeout` | Build timeout | 20m | `--timeout=45m` |
| `--verbose` | Enable verbose logging | false | `--verbose` |

### Advanced Examples

#### Microservices Stack
\`\`\`bash
gke-image-cache-builder -L \
    --project-name=production \
    --disk-image-name=microservices-cache \
    --cache-size=30 \
    --timeout=45m \
    --cache-labels=env=production \
    --cache-labels=team=platform \
    --container-image=gcr.io/my-project/api-gateway:v2.1.0 \
    --container-image=gcr.io/my-project/user-service:v1.8.3 \
    --container-image=gcr.io/my-project/order-service:v1.5.2
\`\`\`

#### CI/CD Integration
\`\`\`bash
gke-image-cache-builder -R \
    --project-name=$GCP_PROJECT \
    --zone=us-central1-a \
    --disk-image-name=ci-cache-$BUILD_ID \
    --timeout=30m \
    --preemptible \
    --cache-labels=build-id=$BUILD_ID \
    --cache-labels=branch=$GIT_BRANCH \
    --container-image=gcr.io/$GCP_PROJECT/app:$GIT_SHA
\`\`\`

#### ML/AI Workloads
\`\`\`bash
gke-image-cache-builder -R \
    --project-name=ml-platform \
    --zone=us-west1-b \
    --disk-image-name=ml-models-cache \
    --cache-size=200 \
    --timeout=2h \
    --machine-type=e2-standard-8 \
    --disk-type=pd-ssd \
    --container-image=tensorflow/tensorflow:2.8.0-gpu \
    --container-image=pytorch/pytorch:1.11.0-cuda11.3-cudnn8-runtime \
    --container-image=gcr.io/ml-platform/custom-model:v3.2.0
\`\`\`

## 💡 Benefits

| Benefit | Description | Impact |
|---------|-------------|--------|
| 🚀 **Zero Image Pull Time** | Pre-cached images eliminate download wait | Pod startup in seconds vs minutes |
| 💰 **Cost Reduction** | Reduce registry bandwidth and egress costs | Significant savings for large deployments |
| ⚡ **Faster Scaling** | Instant pod startup enables rapid scaling | Better auto-scaling responsiveness |
| 🔄 **Reusable Cache** | Cache disks can be attached to multiple nodes | Efficient resource utilization |
| 🛡️ **Reliability** | Reduce dependency on external registries | More resilient deployments |

## 🔧 Advanced Configuration

### Authentication Options
\`\`\`bash
# Service account file
--gcp-oauth=/path/to/service-account.json

# Service account email
--service-account=my-sa@project.iam.gserviceaccount.com

# Image pull authentication
--image-pull-auth=ServiceAccountToken
\`\`\`

### Network Configuration
\`\`\`bash
# Custom VPC and subnet
--network=my-vpc --subnet=my-subnet

# Machine type for remote builds
--machine-type=e2-standard-4

# Use preemptible instances (cost savings)
--preemptible
\`\`\`

### Disk Configuration
\`\`\`bash
# Disk type selection
--disk-type=pd-ssd  # or pd-standard, pd-balanced

# Image family and labels
--cache-family=my-cache-family
--cache-labels=env=prod --cache-labels=team=platform
\`\`\`

## 🆘 Help System

\`\`\`bash
# Basic help
gke-image-cache-builder --help

# Complete reference
gke-image-cache-builder --help-full

# Usage examples and scenarios
gke-image-cache-builder --help-examples

# Version information
gke-image-cache-builder --version
\`\`\`

## 🐛 Troubleshooting

### Common Issues

**Local mode fails with "Not a GCP VM"**
\`\`\`bash
# Solution: Use remote mode or run on a GCP VM
gke-image-cache-builder -R --zone=us-west1-b ...
\`\`\`

**Permission denied errors**
\`\`\`bash
# Ensure proper IAM roles:
# - Compute Instance Admin (v1)
# - Compute Image User
# - Service Account User
\`\`\`

**Large images timeout**
\`\`\`bash
# Increase timeout for large images
--timeout=60m
\`\`\`

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
\`\`\`bash
git clone https://github.com/your-org/gke-image-cache-builder.git
cd gke-image-cache-builder
go mod download
make build
\`\`\`

### Running Tests
\`\`\`bash
make test
make test-binary
\`\`\`

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Original [gke-disk-image-builder](https://github.com/ai-on-gke/tools/tree/main/gke-disk-image-builder) project for inspiration
- Google Cloud Platform team for GKE and container optimization guidance
- Go community for excellent tooling and libraries

## 📞 Support

- 📖 [Documentation](https://github.com/your-org/gke-image-cache-builder/wiki)
- 🐛 [Issue Tracker](https://github.com/your-org/gke-image-cache-builder/issues)
- 💬 [Discussions](https://github.com/your-org/gke-image-cache-builder/discussions)

---

**Built with ❤️ for the Kubernetes community**
