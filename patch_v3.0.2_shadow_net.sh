#!/bin/bash

# ==============================================================================
# Shadow-Net Patch Installer for MHSanaei/3x-ui v3.0.2
# Installs Smart Proxy Outbound Bridge, Custom Settings Tabs, i18n Mappings,
# 3-Strikes Penalty Watchdog, and Multi-Bot Integrations.
# Supports both precompiled binary installation and source-level compilation.
# ==============================================================================

set -o errexit
set -o pipefail
set -o nounset

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO] $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[WARN] $1${NC}"
}

log_error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

# 1. ROOT CHECK
if [[ "$EUID" -ne 0 ]]; then
    log_error "This script must be executed with root/sudo privileges."
    exit 1
fi

# CONSTANTS & PATHS
XUI_DIR="/usr/local/x-ui"

# Dynamically detect active database directory and path
if [[ -f "/etc/x-ui/x-ui.db" ]]; then
    DB_PATH="/etc/x-ui/x-ui.db"
    DB_DIR="/etc/x-ui"
elif [[ -f "/etc/3x-ui/3x-ui.db" ]]; then
    DB_PATH="/etc/3x-ui/3x-ui.db"
    DB_DIR="/etc/3x-ui"
else
    DB_PATH="/etc/x-ui/x-ui.db"
    DB_DIR="/etc/x-ui"
fi

BACKUP_DIR="${DB_DIR}/backups/shadow_net_$(date +%Y%m%d_%H%M%S)"
PATCH_URL="https://raw.githubusercontent.com/vahidvl/shadow-net-panel/main/shadow-net-v3.0.2.patch"
PRECOMPILED_URL="https://github.com/vahidvl/shadow-net-panel/releases/download/v3.0.2-stealth/x-ui-linux-amd64"

echo -e "${BLUE}==================================================================${NC}"
echo -e "${BLUE}        Shadow-Net Proxy & Bot Patch Installer for 3x-ui v3.0.2  ${NC}"
echo -e "${BLUE}==================================================================${NC}"

# Check for existing installation
if [[ ! -d "$XUI_DIR" ]]; then
    log_error "Existing 3x-ui installation not found at $XUI_DIR."
    log_error "Please install official 3x-ui first using standard install script."
    exit 1
fi

# Ask user for installation mode
echo -e "\nChoose installation mode:"
echo -e "  1) Precompiled Binary Overlay (Fastest, zero dependencies)"
echo -e "  2) Compile from Source (Requires Go >= 1.22 and Node.js >= 22)"
read -rp "Enter choice [1-2]: " INSTALL_MODE

# Create Backup directory
log_info "Creating fallback backups in $BACKUP_DIR..."
mkdir -p "$BACKUP_DIR"
if [[ -f "$DB_PATH" ]]; then
    cp "$DB_PATH" "$BACKUP_DIR/3x-ui.db"
    log_info "Database backed up to $BACKUP_DIR/3x-ui.db"
fi
if [[ -f "$XUI_DIR/x-ui" ]]; then
    cp "$XUI_DIR/x-ui" "$BACKUP_DIR/x-ui"
    log_info "Original binary backed up to $BACKUP_DIR/x-ui"
fi

if [[ "$INSTALL_MODE" == "1" ]]; then
    # Mode 1: Precompiled Binary Overlay
    if [[ -f "/root/x-ui-linux-amd64" ]]; then
        log_info "Using local precompiled binary from /root/x-ui-linux-amd64..."
        cp "/root/x-ui-linux-amd64" "$XUI_DIR/x-ui.new"
    elif [[ -f "$(dirname "$0")/x-ui-linux-amd64" ]]; then
        log_info "Using local precompiled binary from $(dirname "$0")/x-ui-linux-amd64..."
        cp "$(dirname "$0")/x-ui-linux-amd64" "$XUI_DIR/x-ui.new"
    else
        log_info "Downloading precompiled Shadow-Net binary..."
        if ! wget -q --show-progress -O "$XUI_DIR/x-ui.new" "$PRECOMPILED_URL"; then
            # Try curl if wget failed
            if ! curl -L -o "$XUI_DIR/x-ui.new" "$PRECOMPILED_URL"; then
                log_error "Failed to download precompiled binary."
                exit 1
            fi
        fi
    fi
    
    log_info "Stopping active 3x-ui systemd services..."
    systemctl stop x-ui || systemctl stop 3x-ui || true
    
    # Overwrite binary
    mv "$XUI_DIR/x-ui.new" "$XUI_DIR/x-ui"
    chmod +x "$XUI_DIR/x-ui"
    log_info "Precompiled binary installed successfully."

else
    # Mode 2: Compile from Source
    log_info "Starting source-level compilation pipeline..."
    
    # Check dependencies
    if ! command -v go &> /dev/null; then
        log_error "Go compiler (golang) is not installed. Please install Go >= 1.22."
        exit 1
    fi
    if ! command -v npm &> /dev/null; then
        log_error "NodeJS (npm) is not installed. Please install Node.js >= 22."
        exit 1
    fi
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed. Please install git."
        exit 1
    fi
    
    # Create temp compile dir
    TEMP_SRC_DIR=$(mktemp -d -t x-ui-compile-XXXXXX)
    log_info "Cloning official MHSanaei/3x-ui repository to $TEMP_SRC_DIR..."
    git clone https://github.com/MHSanaei/3x-ui.git "$TEMP_SRC_DIR"
    cd "$TEMP_SRC_DIR"
    
    log_info "Checking out version v3.0.2..."
    git checkout tags/v3.0.2 || git checkout main
    
    # Obtain patch file
    if [[ -f "/root/shadow-net-v3.0.2.patch" ]]; then
        log_info "Using local patch file from /root/shadow-net-v3.0.2.patch..."
        cp "/root/shadow-net-v3.0.2.patch" ./shadow-net.patch
    elif [[ -f "$(dirname "$0")/shadow-net-v3.0.2.patch" ]]; then
        log_info "Using local patch file from $(dirname "$0")/shadow-net-v3.0.2.patch..."
        cp "$(dirname "$0")/shadow-net-v3.0.2.patch" ./shadow-net.patch
    else
        log_info "Downloading patch file from Github..."
        if ! wget -q -O ./shadow-net.patch "$PATCH_URL"; then
            curl -L -o ./shadow-net.patch "$PATCH_URL" || { log_error "Failed to download patch file."; exit 1; }
        fi
    fi
    
    log_info "Applying Shadow-Net source patch..."
    git apply --whitespace=fix ./shadow-net.patch || { log_error "Patch application failed. Please inspect source conflict manually."; exit 1; }
    
    # Build frontend
    log_info "Building frontend SPA assets..."
    cd frontend
    npm install
    npm run build
    cd ..
    
    # Build backend binary
    log_info "Compiling backend Go binary..."
    go build -tags=jsoniter -ldflags="-s -w" -o bin/x-ui main.go
    
    log_info "Stopping active 3x-ui systemd services..."
    systemctl stop x-ui || systemctl stop 3x-ui || true
    
    # Install binary
    cp bin/x-ui "$XUI_DIR/x-ui"
    chmod +x "$XUI_DIR/x-ui"
    
    # Clean up temp build folder
    rm -rf "$TEMP_SRC_DIR"
    log_info "Source compilation and installation complete."
fi

# 7. SERVICE RESTORE & VERIFICATION
log_info "Starting panel services..."
systemctl daemon-reload
systemctl start x-ui || systemctl start 3x-ui || { log_error "Failed to start x-ui service."; exit 1; }

# Verification Ping
sleep 3
if systemctl is-active --quiet x-ui || systemctl is-active --quiet 3x-ui; then
    echo -e "${GREEN}==================================================================${NC}"
    echo -e "${GREEN} SUCCESS: Shadow-Net Panel Patch has been installed and is active!${NC}"
    echo -e "${GREEN} Original backup database preserved at: $BACKUP_DIR/3x-ui.db${NC}"
    echo -e "${GREEN}==================================================================${NC}"
else
    log_error "System service failed to stabilize after start. Check logs via 'journalctl -u x-ui -n 50'"
    exit 1
fi
