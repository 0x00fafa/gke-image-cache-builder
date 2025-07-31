#!/bin/bash
# GKE Image Cache Builder - VM Setup and Verification Script
# This script is embedded in the binary and executed at runtime

set -e

# Configuration
CONTAINERD_VERSION="1.6.6"
RUNC_VERSION="1.1.4"
CNI_VERSION="1.1.1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

# Error handling
cleanup_on_error() {
    log_error "Script failed. Performing cleanup..."
    exit 1
}

trap cleanup_on_error ERR

# Main execution
main() {
    log_info "Starting GKE Image Cache Builder VM setup and verification"
    
    # Step 1: System preparation
    prepare_system
    
    # Step 2: Install containerd if not present
    install_containerd
    
    # Step 3: Configure containerd for image caching
    configure_containerd
    
    # Step 4: Verify installation
    verify_installation
    
    # Step 5: Setup image cache environment
    setup_cache_environment
    
    log_success "VM setup and verification completed successfully"
}

# System preparation
prepare_system() {
    log_info "Preparing system environment..."
    
    # Update package lists
    apt-get update -qq
    
    # Install required packages
    apt-get install -y \
        apt-transport-https \
        ca-certificates \
        curl \
        gnupg \
        lsb-release \
        wget \
        jq
    
    # Create necessary directories
    mkdir -p /etc/containerd
    mkdir -p /opt/cni/bin
    mkdir -p /var/lib/containerd
    
    log_success "System preparation completed"
}

# Install containerd if not already installed
install_containerd() {
    log_info "Checking containerd installation..."
    
    if command -v containerd >/dev/null 2>&1; then
        local current_version=$(containerd --version | awk '{print $3}' | sed 's/v//')
        log_info "containerd is already installed (version: $current_version)"
        
        # Check if version is acceptable
        if version_ge "$current_version" "$CONTAINERD_VERSION"; then
            log_success "containerd version is acceptable"
            return 0
        else
            log_warn "containerd version is outdated, upgrading..."
        fi
    fi
    
    log_info "Installing containerd $CONTAINERD_VERSION..."
    
    # Download and install containerd
    wget -q "https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz"
    tar Cxzvf /usr/local "containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz"
    rm "containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz"
    
    # Install runc
    wget -q "https://github.com/opencontainers/runc/releases/download/v${RUNC_VERSION}/runc.amd64"
    install -m 755 runc.amd64 /usr/local/sbin/runc
    rm runc.amd64
    
    # Install CNI plugins
    wget -q "https://github.com/containernetworking/plugins/releases/download/v${CNI_VERSION}/cni-plugins-linux-amd64-v${CNI_VERSION}.tgz"
    tar Cxzvf /opt/cni/bin "cni-plugins-linux-amd64-v${CNI_VERSION}.tgz"
    rm "cni-plugins-linux-amd64-v${CNI_VERSION}.tgz"
    
    # Create systemd service
    cat > /etc/systemd/system/containerd.service << 'EOF'
[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target local-fs.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/local/bin/containerd
Type=notify
Delegate=yes
KillMode=process
Restart=always
RestartSec=5
LimitNPROC=infinity
LimitCORE=infinity
LimitNOFILE=infinity
TasksMax=infinity
OOMScoreAdjust=-999

[Install]
WantedBy=multi-user.target
EOF
    
    # Enable and start containerd
    systemctl daemon-reload
    systemctl enable containerd
    systemctl start containerd
    
    log_success "containerd installation completed"
}

# Configure containerd for optimal image caching
configure_containerd() {
    log_info "Configuring containerd for image caching..."
    
    # Generate default config
    containerd config default > /etc/containerd/config.toml
    
    # Optimize for image caching
    cat >> /etc/containerd/config.toml << 'EOF'

# GKE Image Cache Builder optimizations
[plugins."io.containerd.grpc.v1.cri".containerd]
  snapshotter = "overlayfs"
  default_runtime_name = "runc"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
  SystemdCgroup = true

# Image cache optimizations
[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"

[plugins."io.containerd.grpc.v1.cri".registry.mirrors]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["https://registry-1.docker.io"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."gcr.io"]
    endpoint = ["https://gcr.io"]
EOF
    
    # Restart containerd to apply configuration
    systemctl restart containerd
    
    log_success "containerd configuration completed"
}

# Verify installation and functionality
verify_installation() {
    log_info "Verifying containerd installation..."
    
    # Check containerd service status
    if ! systemctl is-active --quiet containerd; then
        log_error "containerd service is not running"
        return 1
    fi
    
    # Check containerd version
    local version=$(containerd --version)
    log_info "containerd version: $version"
    
    # Check runc version
    local runc_version=$(runc --version | head -n1)
    log_info "runc version: $runc_version"
    
    # Test containerd functionality
    if ! ctr version >/dev/null 2>&1; then
        log_error "containerd client test failed"
        return 1
    fi
    
    # Test image operations
    log_info "Testing image pull functionality..."
    if ! ctr images pull docker.io/library/hello-world:latest >/dev/null 2>&1; then
        log_error "Test image pull failed"
        return 1
    fi
    
    # Cleanup test image
    ctr images remove docker.io/library/hello-world:latest >/dev/null 2>&1 || true
    
    log_success "containerd verification completed"
}

# Setup image cache environment
setup_cache_environment() {
    log_info "Setting up image cache environment..."
    
    # Create cache directories
    mkdir -p /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs
    mkdir -p /var/lib/containerd/io.containerd.content.v1.content
    
    # Set appropriate permissions
    chown -R root:root /var/lib/containerd
    chmod -R 755 /var/lib/containerd
    
    # Create cache management script
    cat > /usr/local/bin/cache-manager.sh << 'EOF'
#!/bin/bash
# Image cache management helper script

case "$1" in
    "list")
        echo "Cached images:"
        ctr images list -q
        ;;
    "size")
        echo "Cache disk usage:"
        du -sh /var/lib/containerd
        ;;
    "clean")
        echo "Cleaning unused images..."
        ctr images prune
        ;;
    *)
        echo "Usage: $0 {list|size|clean}"
        exit 1
        ;;
esac
EOF
    
    chmod +x /usr/local/bin/cache-manager.sh
    
    log_success "Image cache environment setup completed"
}

# Version comparison helper
version_ge() {
    [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$2" ]
}

# Execute main function
main "$@"
