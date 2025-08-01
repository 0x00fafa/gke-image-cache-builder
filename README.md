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

## 📦 Installation & Usage

### 🐳 Docker Usage (Recommended)

Docker provides the easiest way to use the tool without installing dependencies.

#### **Interactive Mode (Recommended for Exploration)**

Start an interactive container with a shell to explore the tool:

```bash
# Basic interactive mode
docker run -it gke-image-cache-builder

# With volume mounts for configs and output
docker run -it \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/output:/app/output \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder
```

**Inside the container, you can:**
```bash
# Show help
gke-image-cache-builder --help

# Generate configuration templates
gke-image-cache-builder --generate-config basic --output /app/output/my-config.yaml

# Build cache using configuration
gke-image-cache-builder --config /app/configs/my-config.yaml

# Build cache with command line
gke-image-cache-builder -L --project-name=my-project \
    --disk-image-name=web-cache \
    --container-image=nginx:latest
```

#### **Batch/Task Mode (Recommended for Automation)**

Execute specific commands directly without entering the container:

```bash
# Show help
docker run --rm gke-image-cache-builder --help

# Show version
docker run --rm gke-image-cache-builder --version

# Generate configuration template
docker run --rm \
  -v $(pwd)/output:/app/output \
  gke-image-cache-builder \
  --generate-config basic --output /app/output/web-config.yaml

# Build cache using configuration file
docker run --rm \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  --config /app/configs/my-config.yaml

# Build cache with command line parameters
docker run --rm \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  -R --zone=us-west1-b --project-name=my-project \
  --disk-image-name=web-cache \
  --container-image=nginx:latest \
  --container-image=redis:alpine
```

#### **Docker Volume Mounts**

| Mount Point | Purpose | Example |
|-------------|---------|---------|
| `/app/configs` | Configuration files (read-only) | `-v $(pwd)/configs:/app/configs:ro` |
| `/app/output` | Generated files and outputs | `-v $(pwd)/output:/app/output` |
| `/app/credentials.json` | GCP service account key | `-v $(pwd)/sa.json:/app/credentials.json:ro` |

#### **Docker Environment Variables**

```bash
# Set GCP credentials path
-e GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json

# Set timezone
-e TZ=America/New_York

# Enable verbose logging
-e VERBOSE=true
```

#### **Complete Docker Examples**

**Web Application Cache:**
```bash
# Create configuration
docker run --rm -v $(pwd):/workspace gke-image-cache-builder \
  --generate-config basic --output /workspace/web-config.yaml

# Edit the configuration file
vim web-config.yaml

# Build the cache
docker run --rm \
  -v $(pwd)/web-config.yaml:/app/configs/config.yaml:ro \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  --config /app/configs/config.yaml
```

**CI/CD Pipeline Integration:**
```bash
# In your CI/CD pipeline
docker run --rm \
  -v $PWD/ci-config.yaml:/app/configs/config.yaml:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json \
  -v $GOOGLE_APPLICATION_CREDENTIALS:/app/credentials.json:ro \
  gke-image-cache-builder \
  --config /app/configs/config.yaml \
  --disk-image-name=ci-cache-$BUILD_ID \
  --disk-labels=build-id=$BUILD_ID
```

**ML/AI Workloads:**
```bash
# Generate ML-optimized configuration
docker run --rm -v $(pwd):/workspace gke-image-cache-builder \
  --generate-config ml --output /workspace/ml-config.yaml

# Build large ML image cache
docker run --rm \
  -v $(pwd)/ml-config.yaml:/app/configs/config.yaml:ro \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  --config /app/configs/config.yaml
```

#### **Docker Compose Usage**

For complex setups, use Docker Compose:

```yaml
# docker-compose.yml
version: '3.8'
services:
  gke-cache-builder:
    image: gke-image-cache-builder:latest
    volumes:
      - ./configs:/app/configs:ro
      - ./output:/app/output
      - ./service-account.json:/app/credentials.json:ro
    environment:
      - GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json
    stdin_open: true
    tty: true
```

```bash
# Start interactive session
docker-compose run --rm gke-cache-builder

# Run specific task
docker-compose run --rm gke-cache-builder --help
```

#### **Docker Best Practices**

1. **Use specific tags in production:**
   ```bash
   docker run gke-image-cache-builder:v2.0.0 --help
   ```

2. **Mount volumes as read-only when possible:**
   ```bash
   -v $(pwd)/configs:/app/configs:ro
   ```

3. **Use environment variables for credentials:**
   ```bash
   -e GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json
   ```

4. **Clean up containers automatically:**
   ```bash
   docker run --rm gke-image-cache-builder --help
   ```

5. **Use Docker secrets for sensitive data:**
   ```bash
   echo "service-account-content" | docker secret create gcp-sa -
   ```

#### **Docker Troubleshooting**

**Container exits immediately:**
```bash
# Check if you need interactive mode
docker run -it gke-image-cache-builder

# Check logs
docker logs <container-id>
```

**Permission issues:**
```bash
# Check file permissions
ls -la service-account.json

# Fix permissions
chmod 600 service-account.json
```

**Volume mount issues:**
```bash
# Use absolute paths
docker run -v /absolute/path/to/configs:/app/configs:ro gke-image-cache-builder

# Check if files exist
docker run --rm -v $(pwd)/configs:/app/configs:ro gke-image-cache-builder ls -la /app/configs
```

### 📥 Binary Installation

#### Download Pre-built Binary
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

#### Build from Source
```bash
git clone https://github.com/0x00fafa/gke-image-cache-builder.git
cd gke-image-cache-builder
make build-static
```

#### Using Go Install
```bash
go install github.com/0x00fafa/gke-image-cache-builder/cmd@latest
```

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

disk:
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
- For Docker: Docker installed and running

### Method 1: Docker + Configuration File (Recommended)
```bash
# Generate a configuration template
docker run --rm -v $(pwd):/workspace gke-image-cache-builder \
  --generate-config basic --output /workspace/web-app.yaml

# Edit the configuration file
# vim web-app.yaml

# Build using configuration
docker run --rm \
  -v $(pwd)/web-app.yaml:/app/configs/config.yaml:ro \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  --config /app/configs/config.yaml
```

### Method 2: Docker + Command Line
```bash
# Local Mode (Cost-Effective) - requires running on GCP VM
docker run --rm \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  -L --project-name=my-project \
  --disk-image-name=web-cache \
  --container-image=nginx:latest \
  --container-image=redis:alpine

# Remote Mode (Universal) - works from anywhere
docker run --rm \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  -R --zone=us-west1-b \
  --project-name=my-project \
  --disk-image-name=web-cache \
  --container-image=nginx:latest \
  --container-image=redis:alpine
```

### Method 3: Interactive Docker Mode
```bash
# Start interactive container
docker run -it \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/output:/app/output \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder

# Inside the container:
gke-image-cache-builder --generate-config basic --output /app/output/my-config.yaml
gke-image-cache-builder --config /app/configs/my-config.yaml
```

### Method 4: Binary Installation (Traditional)
```bash
# Download and install binary
curl -L https://github.com/0x00fafa/gke-image-cache-builder/releases/latest/download/gke-image-cache-builder-linux-amd64 -o gke-image-cache-builder
chmod +x gke-image-cache-builder

# Use configuration file
./gke-image-cache-builder --config web-app.yaml

# Use command line
./gke-image-cache-builder -L --project-name=my-project --disk-image-name=web-cache --container-image=nginx:latest
```

## 📖 Usage

### Basic Syntax
```
# Docker approach (recommended)
docker run [docker-options] gke-image-cache-builder [tool-options]

# Binary approach
gke-image-cache-builder {-L|-R} --project-name <PROJECT> --disk-image-name <NAME> [OPTIONS]
gke-image-cache-builder --config <CONFIG_FILE> [OPTIONS]
```

### Docker Usage Patterns

#### **Pattern 1: One-shot Commands**
```bash
# Show help
docker run --rm gke-image-cache-builder --help

# Generate config
docker run --rm -v $(pwd):/workspace gke-image-cache-builder \
  --generate-config basic --output /workspace/config.yaml

# Execute build
docker run --rm -v $(pwd)/config.yaml:/app/config.yaml:ro gke-image-cache-builder \
  --config /app/config.yaml
```

#### **Pattern 2: Interactive Session**
```bash
# Start interactive session
docker run -it -v $(pwd)/configs:/app/configs:ro gke-image-cache-builder

# Inside container, run multiple commands
gke-image-cache-builder --help
gke-image-cache-builder --generate-config advanced --output /app/configs/advanced.yaml
gke-image-cache-builder --config /app/configs/my-config.yaml
exit
```

#### **Pattern 3: Docker Compose Workflow**
```bash
# Start services
docker-compose up -d

# Run interactive session
docker-compose exec gke-cache-builder bash

# Run specific tasks
docker-compose run --rm gke-cache-builder --help
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

#### Multi-Environment Setup with Docker
```bash
# Development
docker run --rm -v $(pwd)/configs:/app/configs:ro gke-image-cache-builder \
  --config /app/configs/dev.yaml

# Staging  
docker run --rm -v $(pwd)/configs:/app/configs:ro gke-image-cache-builder \
  --config /app/configs/staging.yaml

# Production
docker run --rm -v $(pwd)/configs:/app/configs:ro gke-image-cache-builder \
  --config /app/configs/production.yaml
```

#### CI/CD Integration with Docker
```bash
# In your CI/CD pipeline (GitHub Actions, GitLab CI, etc.)
docker run --rm \
  -v $PWD/ci-config.yaml:/app/configs/config.yaml:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/app/credentials.json \
  -v $GOOGLE_APPLICATION_CREDENTIALS:/app/credentials.json:ro \
  gke-image-cache-builder \
  --config /app/configs/config.yaml \
  --disk-image-name=ci-cache-$BUILD_ID \
  --disk-labels=build-id=$BUILD_ID \
  --disk-labels=branch=$BRANCH_NAME
```

#### ML/AI Workload with Docker
```bash
# Generate ML-optimized configuration
docker run --rm -v $(pwd):/workspace gke-image-cache-builder \
  --generate-config ml --output /workspace/ml-config.yaml

# Build ML image cache
docker run --rm \
  -v $(pwd)/ml-config.yaml:/app/configs/config.yaml:ro \
  -v $(pwd)/service-account.json:/app/credentials.json:ro \
  gke-image-cache-builder \
  --config /app/configs/config.yaml
```

## 💡 Benefits

| Benefit | Description | Impact |
|---------|-------------|--------|
| 🚀 **Zero Image Pull Time** | Pre-cached images eliminate download wait | Pod startup in seconds vs minutes |
| 💰 **Cost Reduction** | Reduce registry bandwidth and egress costs | Significant savings for large deployments |
| ⚡ **Faster Scaling** | Instant pod startup enables rapid scaling | Better auto-scaling responsiveness |
| 🔄 **Reusable Cache** | Cache disks can be attached to multiple nodes | Efficient resource utilization |
| 🛡️ **Reliability** | Reduce dependency on external registries | More resilient deployments |
| 🐳 **Container Ready** | Docker support for easy deployment | Works anywhere Docker runs |

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
--disk-family=my-cache-family
--disk-labels=env=prod --disk-labels=team=platform
```

## 🆘 Help System
```bash
# Basic help
gke-image-cache-builder --help

# Complete reference
gke-image-cache-builder --help-full

# Usage examples and scenarios
gke-image-cache-builder --help-examples

# Configuration file help
gke-image-cache-builder --help-config

# Version information
gke-image-cache-builder --version

# Docker-specific help
docker run --rm gke-image-cache-builder help
```

## 🐛 Troubleshooting

### Common Issues

**Local mode fails with "Not a GCP VM"**
```bash
# Solution: Use remote mode or run on a GCP VM
gke-image-cache-builder -R --zone=us-west1-b ...

# Docker solution: Remote mode works from anywhere
docker run --rm gke-image-cache-builder -R --zone=us-west1-b ...
```

**Permission denied errors**
```bash
# Ensure proper IAM roles:
# - Compute Instance Admin (v1)
# - Compute Image User
# - Service Account User

# Docker: Check file permissions
chmod 600 service-account.json
```

**Large images timeout**
```bash
# Increase timeout for large images
--timeout=60m

# Docker example
docker run --rm gke-image-cache-builder --config=/app/config.yaml --timeout=60m
```

**Docker container exits immediately**
```bash
# Use interactive mode for exploration
docker run -it gke-image-cache-builder

# Check if you're using the right command
docker run --rm gke-image-cache-builder --help
```

**Volume mount issues**
```bash
# Use absolute paths
docker run -v /absolute/path/to/configs:/app/configs:ro gke-image-cache-builder

# Check if files exist in container
docker run --rm -v $(pwd)/configs:/app/configs:ro gke-image-cache-builder ls -la /app/configs
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
make docker-test
```

### Docker Development
```bash
# Build development image
make docker-build

# Test Docker functionality
make docker-test

# Run interactive development container
make docker-interactive
```

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Original [gke-disk-image-builder](https://github.com/ai-on-gke/tools/tree/main/gke-disk-image-builder) project for inspiration
- Google Cloud Platform team for GKE and container optimization guidance
- Go community for excellent tooling and libraries
- Docker community for containerization best practices

## 📞 Support

- 📖 [Documentation](https://github.com/0x00fafa/gke-image-cache-builder/wiki)
- 🐛 [Issue Tracker](https://github.com/0x00fafa/gke-image-cache-builder/issues)
- 💬 [Discussions](https://github.com/0x00fafa/gke-image-cache-builder/discussions)
- 🐳 [Docker Hub](https://hub.docker.com/r/0x00fafa/gke-image-cache-builder)

---

**Built with ❤️ for the Kubernetes community**
