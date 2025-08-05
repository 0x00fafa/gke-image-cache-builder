#!/bin/bash
# GKE Image Cache Builder - Enhanced VM Setup and Image Processing Script
# This script combines setup, image processing, and verification functionality

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

# Global variables
DEVICE_NODE=""
MOUNT_POINT="/mnt/disks/container_layers"
ACCESS_TOKEN=""
OAUTH_MECHANISM="none"
STORE_SNAPSHOT_CHECKSUMS="false"

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

log_progress() {
    local current=$1
    local total=$2
    local message=$3
    echo -e "${BLUE}[PROGRESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - ($current/$total) $message"
}

# Error handling
cleanup_on_error() {
    log_error "Script failed. Performing cleanup..."
    cleanup_resources
    exit 1
}

trap cleanup_on_error ERR

# Main execution modes
main() {
    local mode="$1"
    shift
    
    case "$mode" in
        "setup")
            setup_system_environment
            ;;
        "setup-containerd")
            setup_containerd
            ;;
        "prepare-disk")
            local device_name="$1"
            local mount_point="${2:-$MOUNT_POINT}"
            prepare_disk_operations "$device_name" "$mount_point"
            ;;
        "pull-images")
            local auth_mechanism="$1"
            local store_checksums="$2"
            shift 2
            pull_and_process_images "$auth_mechanism" "$store_checksums" "$@"
            ;;
        "verify-image")
            local mount_point="${1:-$MOUNT_POINT}"
            verify_disk_image "$mount_point"
            ;;
        "full-workflow")
            local device_name="$1"
            local auth_mechanism="$2"
            local store_checksums="$3"
            shift 3
            execute_full_workflow "$device_name" "$auth_mechanism" "$store_checksums" "$@"
            ;;
        "cleanup")
            cleanup_resources
            ;;
        *)
            show_usage
            exit 1
            ;;
    esac
}

# Show usage information
show_usage() {
    echo "Usage: $0 <mode> [options]"
    echo ""
    echo "Modes:"
    echo "  setup                           - Setup system environment"
    echo "  setup-containerd               - Setup containerd only"
    echo "  prepare-disk <device> [mount]  - Prepare disk operations"
    echo "  pull-images <auth> <checksums> <images...> - Pull and process images"
    echo "  verify-image [mount_point]     - Verify disk image integrity"
    echo "  full-workflow <device> <auth> <checksums> <images...> - Complete workflow"
    echo "  cleanup                        - Cleanup resources"
    echo ""
    echo "Examples:"
    echo "  $0 setup"
    echo "  $0 prepare-disk secondary-disk-image-disk"
    echo "  $0 pull-images serviceaccounttoken true nginx:latest redis:alpine"
    echo "  $0 full-workflow secondary-disk-image-disk none false nginx:latest"
}

# System preparation
setup_system_environment() {
    log_info "Setting up system environment..."
    
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
        jq \
        util-linux \
        e2fsprogs
    
    # Create necessary directories
    mkdir -p /etc/containerd
    mkdir -p /opt/cni/bin
    mkdir -p /var/lib/containerd
    mkdir -p "$MOUNT_POINT"
    
    log_success "System environment setup completed"
}

# Setup containerd with registry mirrors
setup_containerd() {
    log_info "Setting up containerd..."
    
    if command -v containerd >/dev/null 2>&1; then
        local current_version=$(containerd --version | awk '{print $3}' | sed 's/v//')
        log_info "containerd is already installed (version: $current_version)"
        
        if version_ge "$current_version" "$CONTAINERD_VERSION"; then
            log_success "containerd version is acceptable"
        else
            log_warn "containerd version is outdated, upgrading..."
            install_containerd
        fi
    else
        install_containerd
    fi
    
    # Configure registry mirrors
    configure_registry_mirrors
    
    # Start and verify containerd
    systemctl daemon-reload
    systemctl enable containerd
    systemctl start containerd
    
    # Verify containerd is working
    if ctr version | grep -q "Server:"; then
        log_success "containerd is ready to use"
    else
        log_error "containerd is not running properly"
        exit 1
    fi
}

# Install containerd and dependencies
install_containerd() {
    log_info "Installing containerd $CONTAINERD_VERSION..."
    
    # Install containerd
    if ! apt list --installed | grep -q containerd; then
        apt install -y containerd
    fi
    
    # Download and install specific version if needed
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
    create_containerd_service
    
    log_success "containerd installation completed"
}

# Configure registry mirrors
configure_registry_mirrors() {
    log_info "Configuring registry mirrors..."
    
    # Generate default config
    containerd config default > /etc/containerd/config.toml
    
    # Configure docker.io registry mirrors
    mkdir -p /etc/containerd/certs.d/docker.io
    if [ ! -f /etc/containerd/certs.d/docker.io/hosts.toml ]; then
        tee /etc/containerd/certs.d/docker.io/hosts.toml <<EOF
server = "https://registry-1.docker.io"

[host."https://mirror.gcr.io"]
  capabilities = ["pull", "resolve"]
EOF
        log_success "Registry mirrors configured"
    else
        log_info "Registry mirrors already configured"
    fi
    
    # Add GKE optimizations to containerd config
    cat >> /etc/containerd/config.toml << 'EOF'

# GKE Image Cache Builder optimizations
[plugins."io.containerd.grpc.v1.cri".containerd]
  snapshotter = "overlayfs"
  default_runtime_name = "runc"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
  SystemdCgroup = true

[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"
EOF
}

# Create containerd systemd service
create_containerd_service() {
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
}

# Prepare disk operations
prepare_disk_operations() {
    local device_name="$1"
    local mount_point="$2"
    
    log_info "Preparing disk operations for device: $device_name"
    
    # Set device node path
    DEVICE_NODE="/dev/disk/by-id/google-${device_name}"
    
    # Check if disk is partitioned
    if [[ -e "$DEVICE_NODE-part1" ]]; then
        DEVICE_NODE="$DEVICE_NODE-part1"
        log_info "Using partitioned device: $DEVICE_NODE"
    else
        log_info "Using whole device: $DEVICE_NODE"
    fi
    
    # Check if the device exists
    if ! [ -b "$DEVICE_NODE" ]; then
        log_error "Device $DEVICE_NODE does not exist"
        exit 1
    fi
    
    # Create filesystem
    log_info "Creating ext4 filesystem on $DEVICE_NODE"
    mkfs.ext4 -F -m 0 -E lazy_itable_init=0,lazy_journal_init=0,discard "$DEVICE_NODE"
    
    if [ $? -ne 0 ]; then
        log_error "Failed to create filesystem on $DEVICE_NODE"
        exit 1
    fi
    
    # Create and mount directory
    mkdir -p "$mount_point"
    mount -o discard,defaults "$DEVICE_NODE" "$mount_point"
    
    if [ $? -ne 0 ]; then
        log_error "Failed to mount $DEVICE_NODE to $mount_point"
        exit 1
    fi
    
    # Set permissions
    chmod a+w "$mount_point"
    
    # Initialize metadata files
    rm -f "$mount_point/snapshots.metadata"
    rm -f "$mount_point/images.metadata"
    touch "$mount_point/snapshots.metadata"
    touch "$mount_point/images.metadata"
    chmod a+w "$mount_point/snapshots.metadata"
    chmod a+w "$mount_point/images.metadata"
    
    log_success "Disk preparation completed"
}

# Setup authentication
setup_authentication() {
    local auth_mechanism="$1"
    
    OAUTH_MECHANISM=$(echo "$auth_mechanism" | tr '[:upper:]' '[:lower:]')
    
    if [ "$OAUTH_MECHANISM" = "serviceaccounttoken" ]; then
        log_info "Fetching OAuth token..."
        ACCESS_TOKEN=$(curl -sSf -H "Metadata-Flavor: Google" \
            http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token | \
            jq -r '.access_token')
        
        if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
            log_error "Failed to fetch OAuth token"
            exit 1
        fi
        
        log_success "OAuth token obtained successfully"
    fi
}

# Pull and process container images
pull_and_process_images() {
    local auth_mechanism="$1"
    local store_checksums="$2"
    shift 2
    local images=("$@")
    
    log_info "Starting image processing workflow..."
    
    STORE_SNAPSHOT_CHECKSUMS="$store_checksums"
    
    # Setup authentication
    setup_authentication "$auth_mechanism"
    
    # Remove previous snapshot views
    remove_snapshot_views
    
    # Pull all images
    pull_images "${images[@]}"
    
    # Write image metadata
    write_image_metadata "${images[@]}"
    
    # Process snapshots
    process_snapshots
    
    # Remove original images to save space
    cleanup_pulled_images "${images[@]}"
    
    # Unmount the directory
    umount "$MOUNT_POINT"
    
    log_success "Image processing completed successfully"
    echo "Unpacking is completed."  # Signal for caller
}

# Remove previous snapshot views
remove_snapshot_views() {
    log_info "Removing previous snapshot views..."
    
    local views=($(ctr -n k8s.io snapshot list | grep "View" | awk '{print $1}'))
    for view in "${views[@]}"; do
        if [ -n "$view" ]; then
            log_info "Removing view: $view"
            ctr -n k8s.io snapshot rm "$view" 2>/dev/null || true
        fi
    done
}

# Pull container images
pull_images() {
    local images=("$@")
    local total=${#images[@]}
    local current=0
    
    log_info "Pulling $total container images..."
    
    # Create marker file to indicate we're starting image pulling
    touch /tmp/image_pull_started.flag
    
    for image in "${images[@]}"; do
        ((current++))
        log_progress "$current" "$total" "Pulling $image"
        
        # Create marker file for current image
        echo "$image" > "/tmp/pulling_image_${current}.flag"
        
        # Ensure image name has proper registry prefix for Docker Hub images
        local full_image_name="$image"
        if [[ "$image" != *"."* ]]; then
            # If no registry specified, assume it's Docker Hub
            if [[ "$image" != *"/"* ]]; then
                # Official Docker Hub image (e.g., nginx:latest -> docker.io/library/nginx:latest)
                full_image_name="docker.io/library/$image"
            else
                # User/organization Docker Hub image (e.g., myuser/myapp:latest -> docker.io/myuser/myapp:latest)
                full_image_name="docker.io/$image"
            fi
        fi
        
        # Log the full image name we're pulling
        log_info "Pulling full image name: $full_image_name"
        
        if [ "$OAUTH_MECHANISM" = "none" ]; then
            log_info "Executing: ctr -n k8s.io image pull --hosts-dir /etc/containerd/certs.d $full_image_name"
            ctr -n k8s.io image pull --hosts-dir "/etc/containerd/certs.d" "$full_image_name"
        elif [ "$OAUTH_MECHANISM" = "serviceaccounttoken" ]; then
            log_info "Executing: ctr -n k8s.io image pull --hosts-dir /etc/containerd/certs.d --user oauth2accesstoken:*** $full_image_name"
            ctr -n k8s.io image pull --hosts-dir "/etc/containerd/certs.d" \
                --user "oauth2accesstoken:$ACCESS_TOKEN" "$full_image_name"
        else
            log_error "Unknown OAuth mechanism: $OAUTH_MECHANISM"
            # Create error marker file
            echo "$OAUTH_MECHANISM" > /tmp/image_pull_error_unknown_auth.flag
            exit 1
        fi
        
        local pull_result=$?
        if [ $pull_result -ne 0 ]; then
            log_error "Failed to pull image: $full_image_name (from $image) with exit code $pull_result"
            # Create error marker file
            echo "$full_image_name,$pull_result" > "/tmp/image_pull_error_${current}.flag"
            exit 1
        else
            log_success "Successfully pulled image: $full_image_name"
            # Create success marker file
            echo "$full_image_name" > "/tmp/image_pull_success_${current}.flag"
        fi
    done
    
    # Create marker file to indicate all images were pulled successfully
    touch /tmp/all_images_pulled.flag
    log_success "All images pulled successfully"
}

# Write image metadata
write_image_metadata() {
    log_info "Writing image metadata..."
    
    local images_info=$(ctr -n k8s.io images ls)
    echo "$images_info" >> "$MOUNT_POINT/images.metadata"
    
    log_success "Image metadata written"
}

# Process snapshots - core functionality
process_snapshots() {
    log_info "Processing snapshots..."
    
    local snapshots=($(ctr -n k8s.io snapshot list | grep "Committed" | awk '{print $1}'))
    local total=${#snapshots[@]}
    local current=0
    
    for snapshot in "${snapshots[@]}"; do
        ((current++))
        log_progress "$current" "$total" "Processing snapshot: $snapshot"
        
        # Create temporary view with retry mechanism
        local retries=5
        local original_path=""
        
        while [ ${retries} -ge 1 ]; do
            ((retries--))
            
            # Create snapshot view
            ctr -n k8s.io snapshot view "tmp_$snapshot" "$snapshot"
            
            if [ $? -ne 0 ]; then
                log_warn "Failed to create snapshot view for $snapshot. Retries left: $retries"
                continue
            fi
            
            # Get mount point
            original_path=$(ctr -n k8s.io snapshot mount "/tmp_$snapshot" "tmp_$snapshot" | \
                grep -oP '/\S+/snapshots/[0-9]+/fs' | head -n 1)
            
            if [[ -n "$original_path" ]]; then
                break
            fi
            
            log_warn "Failed to get mount point for tmp_$snapshot. Retries left: $retries"
            ctr -n k8s.io snapshot rm "tmp_$snapshot" 2>/dev/null || true
            sleep 1
        done
        
        if [[ -z "$original_path" ]]; then
            log_error "Failed to process snapshot: $snapshot"
            exit 1
        fi
        
        # Copy snapshot data
        local new_path=$(echo "$original_path" | grep -o "snapshots/.*/fs")
        mkdir -p "$MOUNT_POINT/${new_path}"
        cp -r -p "$original_path" "$MOUNT_POINT/${new_path}/.."
        
        if [ $? -ne 0 ]; then
            log_error "Failed to copy snapshot data for: $snapshot"
            exit 1
        fi
        
        # Generate metadata entry
        local mapping="$snapshot $new_path"
        
        if [ "$STORE_SNAPSHOT_CHECKSUMS" = "true" ]; then
            log_info "Calculating checksum for snapshot: $snapshot"
            local checksum=$(find "$MOUNT_POINT/${new_path}" -type f -exec md5sum {} + | \
                cut -d' ' -f1 | LC_ALL=C sort | md5sum | cut -d' ' -f1)
            mapping="$snapshot $new_path $checksum"
        fi
        
        echo "$mapping" >> "$MOUNT_POINT/snapshots.metadata"
        
        if [ $? -ne 0 ]; then
            log_error "Failed to write metadata for snapshot: $snapshot"
            exit 1
        fi
        
        # Cleanup temporary view
        ctr -n k8s.io snapshot rm "tmp_$snapshot" 2>/dev/null || true
    done
    
    log_success "Snapshot processing completed"
    
    # Show metadata summary
    log_info "Snapshots metadata summary:"
    cat "$MOUNT_POINT/snapshots.metadata"
}

# Cleanup pulled images
cleanup_pulled_images() {
    local images=("$@")
    
    log_info "Cleaning up original pulled images..."
    
    for image in "${images[@]}"; do
        log_info "Removing original image: $image"
        ctr -n k8s.io image rm "$image" 2>/dev/null || true
    done
}

# Verify disk image integrity
verify_disk_image() {
    local mount_point="${1:-$MOUNT_POINT}"
    
    log_info "Verifying disk image integrity..."
    
    # Mount the disk if not already mounted
    if ! mountpoint -q "$mount_point"; then
        if [ -n "$DEVICE_NODE" ] && [ -b "$DEVICE_NODE" ]; then
            mkdir -p "$mount_point"
            mount -o discard,defaults "$DEVICE_NODE" "$mount_point"
        else
            log_error "Cannot mount disk for verification"
            exit 1
        fi
    fi
    
    local metadata_file="$mount_point/snapshots.metadata"
    
    if [ ! -f "$metadata_file" ]; then
        log_error "Snapshots metadata file not found: $metadata_file"
        exit 1
    fi
    
    local snapshot_broken=""
    
    while IFS= read -r line; do
        if [ -z "$line" ]; then
            continue
        fi
        
        local snapshot_chainID=$(echo "$line" | cut -d' ' -f1)
        local snapshot_path=$(echo "$line" | cut -d' ' -f2)
        local expected_checksum=$(echo "$line" | cut -d' ' -f3)
        
        if [ -z "$expected_checksum" ]; then
            log_error "Expected checksums not found. Use --store-snapshot-checksum when building"
            exit 1
        fi
        
        log_info "Verifying snapshot: $snapshot_chainID"
        
        local actual_checksum=$(find "$mount_point/${snapshot_path}" -type f -exec md5sum {} + | \
            cut -d' ' -f1 | LC_ALL=C sort | md5sum | cut -d' ' -f1)
        
        if [ "$expected_checksum" = "$actual_checksum" ]; then
            log_success "Verification passed for snapshot: $snapshot_chainID"
        else
            log_error "Verification failed for snapshot: $snapshot_chainID"
            log_error "Expected: $expected_checksum, Got: $actual_checksum"
            snapshot_broken="true"
        fi
    done < "$metadata_file"
    
    if [ -n "$snapshot_broken" ]; then
        log_error "Disk image verification failed - checksum mismatch detected"
        exit 1
    else
        log_success "Disk image verification completed successfully"
    fi
}

# Execute full workflow
execute_full_workflow() {
    local device_name="$1"
    local auth_mechanism="$2"
    local store_checksums="$3"
    shift 3
    local images=("$@")

    log_info "Starting full image cache build workflow..."
    log_info "Device name: $device_name"
    log_info "Auth mechanism: $auth_mechanism"
    log_info "Store checksums: $store_checksums"
    log_info "Images to process: ${images[*]}"

    # Step 1: Setup system environment
    setup_system_environment

    # Step 2: Setup containerd
    setup_containerd

    # Step 3: Prepare disk
    prepare_disk_operations "$device_name" "$MOUNT_POINT"

    # Step 4: Pull and process images
    pull_and_process_images "$auth_mechanism" "$store_checksums" "${images[@]}"

    log_success "Full workflow completed successfully"
}

# Cleanup resources
cleanup_resources() {
    log_info "Cleaning up resources..."
    
    # Unmount if mounted
    if mountpoint -q "$MOUNT_POINT" 2>/dev/null; then
        umount "$MOUNT_POINT" || true
    fi
    
    # Remove snapshot views
    remove_snapshot_views
    
    log_info "Cleanup completed"
}

# Version comparison helper
version_ge() {
    [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$2" ]
}

# Execute main function with all arguments
main "$@"
