# GKE Image Cache Builder

A modern, redesigned tool for building container image cache disks to accelerate GKE node startup performance.

## 🎯 Project Background

This project is **inspired by and builds upon** the original [gke-disk-image-builder](https://github.com/ai-on-gke/tools/tree/main/gke-disk-image-builder) from the ai-on-gke/tools repository. While the original project provided a solid foundation, we identified several areas for improvement and have completely redesigned the tool with modern Go practices and enhanced user experience.

## 🚀 Purpose

Accelerate pod startup by pre-caching container images at the disk level, eliminating image pull latency for GKE workloads. This approach can reduce pod startup time from minutes to seconds for large container images.

```
┌─ Container Images ─┐    ┌─ Image Cache Disk ─┐    ┌─ GKE Node ─┐
│ nginx:latest       │ ──▶│ Pre-cached Images  │ ──▶│ Instant    │
│ redis:alpine       │    │ (containerd ready) │    │ Pod Start  │
│ postgres:13        │    │                    │    │            │
└────────────────────┘    └────────────────────┘    └────────────┘
```

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

### 📁 **YAML Configuration File Support**
- Generate configuration templates for different use cases
- Mix configuration files with command-line overrides
- Environment-specific configurations (dev, staging, production)
- Built-in validation and help system

### 🛠️ **Enhanced Configuration**
- Comprehensive parameter validation
- Flexible timeout settings
- Custom machine types and disk configurations
- Advanced labeling and tagging

### 📚 **Rich Help System**
- Context-aware error messages with solutions
- Comprehensive examples and scenarios
- Multiple help levels (basic, full, examples, config)

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

## 📁 Configuration File Support

### Why Use Configuration Files?

Configuration files provide several advantages over command-line only approaches:

- **Simplified Commands**: Complex builds become simple `--config my-app.yaml`
- **Reusability**: Share configurations across teams and environments
- **Version Control**: Track configuration changes alongside code
- **Environment Management**: Separate configs for dev, staging, production
- **Documentation**: YAML comments explain configuration choices

### Configuration Priority

Parameters are applied in this order (highest to lowest priority):
1. **Command line parameters** (highest priority)
2. **Environment variables**
3. **Configuration file values**
4. **Default values** (lowest priority)

### Quick Configuration Start

```bash
# Generate a configuration template
gke-image-cache-builder --generate-config basic --output web-app.yaml

# Edit the generated file, then use it
gke-image-cache-builder --config web-app.yaml

# Mix configuration file with command line overrides
gke-image-cache-builder --config base.yaml --project-name=override-project
```

### Available Templates

| Template | Description | Use Case |
|----------|-------------|----------|
| `basic` | Minimal configuration | Simple applications, getting started |
| `advanced` | All available options | Production deployments, complex setups |
| `ci-cd` | CI/CD optimized | Automated pipelines, cost optimization |
| `ml` | ML/AI workloads | Large models, GPU workloads, long timeouts |

### Configuration File Examples

#### Basic Configuration
```yaml
# web-app.yaml
execution:
  mode: local
  
project:
  name: my-project

disk:  # 改为 disk
  name: web-app-cache
  size_gb: 20
  labels:
    env: production
    team: platform

images:
  - nginx:1.21
  - redis:6.2-alpine
  - postgres:13
```

#### Advanced Production Configuration
```yaml
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
    cost-center: engineering

images:
  - gcr.io/my-project/api-gateway:v2.1.0
  - gcr.io/my-project/user-service:v1.8.3
  - gcr.io/my-project/order-service:v1.5.2
  - nginx:1.21
  - redis:6.2-alpine

# Network settings for build VM only (remote mode)
# These do NOT affect the final disk image
network:
  network: production-vpc
  subnet: production-subnet

advanced:
  timeout: 45m
  machine_type: e2-standard-4
  preemptible: true

auth:
  service_account: cache-builder@production.iam.gserviceaccount.com
  image_pull_auth: ServiceAccountToken

logging:
  verbose: true
```

### Configuration Management Commands

```bash
# Generate configuration templates
gke-image-cache-builder --generate-config basic --output basic.yaml
gke-image-cache-builder --generate-config advanced --output advanced.yaml
gke-image-cache-builder --generate-config ci-cd --output ci-cd.yaml
gke-image-cache-builder --generate-config ml --output ml.yaml

# Validate configuration files
gke-image-cache-builder --validate-config my-config.yaml

# Get configuration help
gke-image-cache-builder --help-config
```

## 🚀 Quick Start

### Prerequisites
- GCP project with Compute Engine API enabled
- Appropriate IAM permissions
- For local mode: Must run on a GCP VM instance

### Method 1: Configuration File (Recommended)
```bash
# Generate a configuration template
gke-image-cache-builder --generate-config basic --output web-app.yaml

# Edit the configuration file
# vim web-app.yaml

# Build using configuration
gke-image-cache-builder --config web-app.yaml
```

### Method 2: Command Line (Traditional)
```bash
# Local Mode (Cost-Effective)
gke-image-cache-builder -L \
    --project-name=my-project \
    --disk-image-name=web-cache \
    --container-image=nginx:latest \
    --container-image=redis:alpine

# Remote Mode (Universal)
gke-image-cache-builder -R \
    --zone=us-west1-b \
    --project-name=my-project \
    --disk-image-name=web-cache \
    --container-image=nginx:latest \
    --container-image=redis:alpine
```

### Method 3: Hybrid Approach
```bash
# Use config file but override specific parameters
gke-image-cache-builder --config production.yaml \
    --project-name=staging-project \
    --disk-image-name=staging-cache
```

## 📦 Installation

### Download Pre-built Binary
```bash
# Linux AMD64
curl -L https://github.com/0x00fafa/gke-image-cache-builder/releases/latest/download/gke-image-cache-builder-linux-amd64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder

# macOS AMD64
curl -L https://github.com/0x00fafa/gke-image-cache-builder/releases/latest/download/gke-image-cache-builder-darwin-amd64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder

# macOS ARM64 (Apple Silicon)
curl -L https://github.com/0x00fafa/gke-image-cache-builder/releases/latest/download/gke-image-cache-builder-darwin-arm64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder
```

### Build from Source
```bash
git clone https://github.com/0x00fafa/gke-image-cache-builder.git
cd gke-image-cache-builder
make build-static
```

### Using Go Install
```bash
go install github.com/0x00fafa/gke-image-cache-builder/cmd@latest
```

## 📖 Usage

### Basic Syntax
```
# Command line approach
gke-image-cache-builder {-L|-R} --project-name <PROJECT> --disk-image-name <NAME> [OPTIONS]

# Configuration file approach  
gke-image-cache-builder --config <CONFIG_FILE> [OPTIONS]

# Hybrid approach (config + CLI overrides)
gke-image-cache-builder --config <CONFIG_FILE> [CLI_OVERRIDES]
```

### Configuration File Parameters
| Section | Parameter | Description | Example |
|---------|-----------|-------------|---------|
| `execution` | `mode` | Execution mode | `local` or `remote` |
| `execution` | `zone` | GCP zone | `us-west1-b` |
| `project` | `name` | GCP project name | `my-project` |
| `disk` | `name` | Disk image name | `web-app-cache` |
| `disk` | `size_gb` | Disk size in GB | `20` |
| `disk` | `family` | Image family | `web-cache` |
| `disk` | `disk_type` | Disk type | `pd-ssd` |
| `disk` | `labels` | Key-value labels | `env: production` |
| `images` | - | Container images list | `- nginx:latest` |
| `network` | `network` | VPC network for build VM only | `my-vpc` |
| `network` | `subnet` | Subnet for build VM only | `my-subnet` |
| `advanced` | `timeout` | Build timeout | `45m` |
| `advanced` | `machine_type` | VM machine type | `e2-standard-4` |
| `advanced` | `preemptible` | Use preemptible VM | `true` |
| `auth` | `service_account` | Service account email | `sa@project.iam.gserviceaccount.com` |
| `auth` | `image_pull_auth` | Image pull auth | `ServiceAccountToken` |
| `logging` | `verbose` | Verbose logging | `true` |
| `logging` | `quiet` | Quiet mode | `false` |

### Advanced Examples

#### Multi-Environment Setup
```bash
# Development
gke-image-cache-builder --config configs/dev.yaml

# Staging  
gke-image-cache-builder --config configs/staging.yaml

# Production
gke-image-cache-builder --config configs/production.yaml
```

#### CI/CD Integration with Configuration
```bash
# Generate CI/CD optimized config
gke-image-cache-builder --generate-config ci-cd --output .github/gke-cache.yaml

# Use in GitHub Actions
gke-image-cache-builder --config .github/gke-cache.yaml \
    --disk-image-name=ci-cache-${{ github.run_id }} \
    --cache-labels=build-id=${{ github.run_id }} \
    --cache-labels=branch=${{ github.ref_name }}
```

#### Configuration with Environment Variables
```yaml
# config.yaml with environment variable placeholders
project:
  name: ${GCP_PROJECT}
  
disk:  # 改为 disk
  name: ${CACHE_NAME:-default-cache}
  labels:
    build-id: ${BUILD_ID}
    branch: ${GIT_BRANCH}
```

#### ML/AI Workload Configuration
```bash
# Generate ML-optimized configuration
gke-image-cache-builder --generate-config ml --output ml-config.yaml

# Use for ML workloads
gke-image-cache-builder --config ml-config.yaml
```

## 💡 Benefits

| Benefit | Description | Impact |
|---------|-------------|--------|
| 🚀 **Zero Image Pull Time** | Pre-cached images eliminate download wait | Pod startup in seconds vs minutes |
| 💰 **Cost Reduction** | Reduce registry bandwidth and egress costs | Significant savings for large deployments |
| ⚡ **Faster Scaling** | Instant pod startup enables rapid scaling | Better auto-scaling responsiveness |
| 🔄 **Reusable Cache** | Cache disks can be attached to multiple nodes | Efficient resource utilization |
| 🛡️ **Reliability** | Reduce dependency on external registries | More resilient deployments |

## 🔧 Advanced Configuration

### Network Configuration
```bash
# Custom VPC and subnet for build VM (remote mode only)
# These settings only affect the temporary VM used for building,
# NOT the final disk image
--network=my-vpc --subnet=my-subnet

# Machine type for remote builds
--machine-type=e2-standard-4

# Use preemptible instances (cost savings)
--preemptible
```

### Disk Configuration
```bash
# Disk type selection
--disk-type=pd-ssd  # or pd-standard, pd-balanced

# Image family and labels
--disk-family=my-cache-family  # 改为 disk-family
--disk-labels=env=prod --disk-labels=team=platform  # 改为 disk-labels
```

## 🆘 Help System
```bash
# Basic help
gke-image-cache-builder --help

# Complete reference
gke-image-cache-builder --help-full

# Usage examples and scenarios
gke-image-cache-builder --help-examples

# Version information
gke-image-cache-builder --version
```

## 🐛 Troubleshooting

### Common Issues

**Local mode fails with "Not a GCP VM"**
```bash
# Solution: Use remote mode or run on a GCP VM
gke-image-cache-builder -R --zone=us-west1-b ...
```

**Permission denied errors**
```bash
# Ensure proper IAM roles:
# - Compute Instance Admin (v1)
# - Compute Image User
# - Service Account User
```

**Large images timeout**
```bash
# Increase timeout for large images
--timeout=60m
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
```bash
git clone https://github.com/0x00fafa/gke-image-cache-builder.git
cd gke-image-cache-builder
go mod download
make build
```

### Running Tests
```bash
make test
make test-binary
```

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Original [gke-disk-image-builder](https://github.com/ai-on-gke/tools/tree/main/gke-disk-image-builder) project for inspiration
- Google Cloud Platform team for GKE and container optimization guidance
- Go community for excellent tooling and libraries

## 📞 Support

- 📖 [Documentation](https://github.com/0x00fafa/gke-image-cache-builder/wiki)
- 🐛 [Issue Tracker](https://github.com/0x00fafa/gke-image-cache-builder/issues)
- 💬 [Discussions](https://github.com/0x00fafa/gke-image-cache-builder/discussions)

---

**Built with ❤️ for the Kubernetes community**
