#!/bin/bash
# GKE Image Cache Builder - Smart Container Entrypoint
# This script provides intelligent routing between interactive and batch modes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Tool information
TOOL_NAME="gke-image-cache-builder"
TOOL_PATH="/app/${TOOL_NAME}"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Show welcome message for interactive mode
show_welcome() {
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}                ${GREEN}GKE Image Cache Builder${NC}                     ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}          ${BLUE}Container Interactive Mode${NC}                      ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${GREEN}🚀 Welcome to GKE Image Cache Builder!${NC}"
    echo ""
    echo -e "${BLUE}PURPOSE:${NC}"
    echo "   Build container image cache disks to accelerate GKE pod startup"
    echo ""
    echo -e "${BLUE}QUICK START:${NC}"
    echo "   # Show help"
    echo "   ${TOOL_NAME} --help"
    echo ""
    echo "   # Generate configuration template"
    echo "   ${TOOL_NAME} --generate-config basic --output /app/output/my-config.yaml"
    echo ""
    echo "   # Build cache using configuration"
    echo "   ${TOOL_NAME} --config /app/configs/my-config.yaml"
    echo ""
    echo "   # Build cache with command line"
    echo "   ${TOOL_NAME} -L --project-name=my-project --disk-image-name=cache \\"
    echo "       --container-image=nginx:latest"
    echo ""
    echo -e "${BLUE}MOUNTED VOLUMES:${NC}"
    if [ -d "/app/configs" ]; then
        echo "   📁 /app/configs  - Configuration files"
    fi
    if [ -d "/app/output" ]; then
        echo "   📁 /app/output   - Output directory"
    fi
    if [ -f "/app/credentials.json" ]; then
        echo "   🔑 /app/credentials.json - GCP service account credentials"
    fi
    echo ""
    echo -e "${BLUE}EXAMPLES:${NC}"
    echo "   ${TOOL_NAME} --help-examples    # Show detailed examples"
    echo "   ${TOOL_NAME} --help-config      # Configuration file help"
    echo ""
    echo -e "${YELLOW}💡 TIP: Use 'exit' to leave this container${NC}"
    echo ""
}

# Show container help
show_container_help() {
    echo -e "${BLUE}GKE Image Cache Builder - Container Usage${NC}"
    echo ""
    echo -e "${GREEN}INTERACTIVE MODE:${NC}"
    echo "   docker run -it gke-image-cache-builder"
    echo "   docker run -it gke-image-cache-builder interactive"
    echo ""
    echo -e "${GREEN}BATCH/TASK MODE:${NC}"
    echo "   docker run gke-image-cache-builder --help"
    echo "   docker run gke-image-cache-builder --version"
    echo "   docker run gke-image-cache-builder --config=/app/configs/my-config.yaml"
    echo ""
    echo -e "${GREEN}CONFIGURATION GENERATION:${NC}"
    echo "   docker run gke-image-cache-builder --generate-config basic --output=/app/output/config.yaml"
    echo ""
    echo -e "${GREEN}WITH VOLUMES:${NC}"
    echo "   docker run -it \\"
    echo "     -v \$(pwd)/configs:/app/configs:ro \\"
    echo "     -v \$(pwd)/output:/app/output \\"
    echo "     -v \$(pwd)/service-account.json:/app/credentials.json:ro \\"
    echo "     gke-image-cache-builder"
    echo ""
    echo -e "${BLUE}For tool-specific help, run: ${TOOL_NAME} --help${NC}"
}

# Check if running in interactive terminal
is_interactive() {
    [ -t 0 ] && [ -t 1 ]
}

# Signal handler for graceful shutdown
cleanup() {
    log_info "Received shutdown signal, cleaning up..."
    exit 0
}

# Set up signal handlers
trap cleanup SIGTERM SIGINT

# Main entrypoint logic
main() {
    # Handle special container commands first
    case "$1" in
        "help"|"container-help")
            show_container_help
            exit 0
            ;;
        "interactive")
            # Explicit interactive mode
            if is_interactive; then
                show_welcome
                log_info "Starting interactive shell..."
                exec /bin/bash
            else
                log_error "Interactive mode requires TTY. Use: docker run -it <image> interactive"
                exit 1
            fi
            ;;
        "")
            # No arguments - default behavior
            if is_interactive; then
                # Interactive terminal available - start interactive mode
                show_welcome
                log_info "Starting interactive shell..."
                exec /bin/bash
            else
                # No TTY - show help and exit
                log_info "No arguments provided and no TTY available"
                exec "$TOOL_PATH" --help
            fi
            ;;
        *)
            # Pass all arguments to the tool
            log_info "Executing: $TOOL_NAME $*"
            exec "$TOOL_PATH" "$@"
            ;;
    esac
}

# Validate tool exists
if [ ! -x "$TOOL_PATH" ]; then
    log_error "Tool not found at $TOOL_PATH"
    exit 1
fi

# Execute main logic
main "$@"
