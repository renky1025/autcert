#!/bin/bash

# AutoCert Installation Script - Linux
# Supports Ubuntu, CentOS, Debian, AlmaLinux and other major distributions

set -euo pipefail

# Set encoding to support proper text output - Enhanced version
# First try to set Chinese environment
if locale -a 2>/dev/null | grep -q "zh_CN.utf8\|zh_CN.UTF-8"; then
    export LANG=zh_CN.UTF-8
    export LC_ALL=zh_CN.UTF-8
elif locale -a 2>/dev/null | grep -q "en_US.utf8\|en_US.UTF-8"; then
    export LANG=en_US.UTF-8
    export LC_ALL=en_US.UTF-8
else
    # If the system does not support UTF-8 language packages, use C.UTF-8
    export LANG=C.UTF-8
    export LC_ALL=C.UTF-8
fi

# Ensure UTF-8 encoding
if command -v stty >/dev/null 2>&1; then
    stty iutf8 2>/dev/null || true
fi

# Configuration variables
PROGRAM_NAME="autocert"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/autocert"
SERVICE_NAME="autocert"
GITHUB_REPO="renky1025/autocert"
ACVERSION="v1.0.0-final"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Log functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_debug() {
    if [[ "${DEBUG:-}" == "1" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1" >&2
    fi
}

# Error handling
error_exit() {
    log_error "$1"
    exit 1
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error_exit "This script requires root privileges. Please run with sudo."
    fi
}

# Detect operating system - Simplified version
detect_os() {
    # Simplified OS detection - just use "linux" for all Linux distributions
    OS="linux"
    
    # Try to get distribution name for logging only
    local dist_name="unknown"
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        dist_name="$NAME"
    elif type lsb_release >/dev/null 2>&1; then
        dist_name=$(lsb_release -si)
    fi
    
    log_info "Detected Linux distribution: $dist_name"
    log_info "Using unified platform identifier: linux"
}

# Detect architecture
detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm"
            ;;
        *)
            error_exit "Unsupported architecture: $ARCH"
            ;;
    esac
    log_info "Detected architecture: $ARCH"
}

# Install dependencies
install_dependencies() {
    log_info "Installing system dependencies..."
    
    # Detect package manager and install dependencies
    if command -v apt-get >/dev/null 2>&1; then
        log_info "Using apt package manager"
        apt-get update
        apt-get install -y curl wget unzip tar openssl ca-certificates
    elif command -v dnf >/dev/null 2>&1; then
        log_info "Using dnf package manager"
        dnf install -y curl wget unzip tar openssl ca-certificates
    elif command -v yum >/dev/null 2>&1; then
        log_info "Using yum package manager"
        yum install -y curl wget unzip tar openssl ca-certificates
    elif command -v zypper >/dev/null 2>&1; then
        log_info "Using zypper package manager"
        zypper install -y curl wget unzip tar openssl ca-certificates
    elif command -v pacman >/dev/null 2>&1; then
        log_info "Using pacman package manager"
        pacman -S --noconfirm curl wget unzip tar openssl ca-certificates
    else
        log_warn "No supported package manager found, assuming dependencies are installed"
    fi
}

# Download AutoCert binary files
download_binary() {
    log_info "Downloading AutoCert binary files...  $ACVERSION"
    
    local download_url=""
    local temp_file="/tmp/autocert_download.tar.gz"
    
    download_url="https://github.com/${GITHUB_REPO}/releases/download/${ACVERSION}/autocert_${ACVERSION}_linux_${ARCH}.tar.gz"
    
    log_info "Download URL: $download_url"
    
    # Download file with enhanced retry mechanism
    log_info "Downloading: $download_url"
    
    for attempt in 1 2 3; do
        log_debug "Download attempt $attempt"
        
        if curl -L --connect-timeout 15 --max-time 300 \
            --retry 2 --retry-delay 3 --retry-max-time 60 \
            -H "User-Agent: AutoCert-Installer/1.0" \
            -o "$temp_file" "$download_url"; then
            
            # Verify downloaded file
            if [[ -f "$temp_file" && -s "$temp_file" ]]; then
                log_info "Download completed successfully"
                break
            else
                log_warn "Downloaded file is empty or invalid"
                rm -f "$temp_file"
            fi
        fi
        
        if [[ $attempt -eq 3 ]]; then
            error_exit "Download failed after 3 attempts: $download_url"
        else
            log_warn "Download attempt $attempt failed, retrying in 5 seconds..."
            sleep 5
        fi
    done
    
    # Extract to temporary directory
    local temp_dir="/tmp/autocert_extract_$$"
    mkdir -p "$temp_dir"
    
    log_info "Extracting files..."
    if ! tar -xzf "$temp_file" -C "$temp_dir"; then
        rm -rf "$temp_file" "$temp_dir"
        error_exit "Failed to extract downloaded archive"
    fi
    
    # Install binary file
    if [[ -f "$temp_dir/autocert" ]]; then
        install -m 755 "$temp_dir/autocert" "$INSTALL_DIR/autocert"
        log_info "Binary file installation completed: $INSTALL_DIR/autocert"
    else
        # Try to find binary in subdirectories
        local binary_file
        binary_file=$(find "$temp_dir" -name "autocert" -type f 2>/dev/null | head -1)
        if [[ -n "$binary_file" ]]; then
            install -m 755 "$binary_file" "$INSTALL_DIR/autocert"
            log_info "Binary file installation completed: $INSTALL_DIR/autocert"
        else
            rm -rf "$temp_file" "$temp_dir"
            error_exit "autocert binary file not found after extraction"
        fi
    fi
    
    # Clean up temporary files
    rm -rf "$temp_file" "$temp_dir"
}

# Create configuration directory
create_config_dir() {
    log_info "Creating configuration directory..."
    
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$CONFIG_DIR/certs"
    mkdir -p "/var/log"
    
    # Set permissions
    chmod 755 "$CONFIG_DIR"
    chmod 700 "$CONFIG_DIR/certs"
    
    log_info "Configuration directory created: $CONFIG_DIR"
}

# Create default configuration file
create_default_config() {
    log_info "Creating default configuration file..."
    
    local config_file="$CONFIG_DIR/config.yaml"
    
    if [[ ! -f "$config_file" ]]; then
        cat > "$config_file" << EOF
# AutoCert Configuration File
log_level: info
config_dir: $CONFIG_DIR
cert_dir: $CONFIG_DIR/certs
log_dir: /var/log

# ACME Configuration
acme:
  server: https://acme-v02.api.letsencrypt.org/directory
  key_type: rsa
  key_size: 2048

# Web Server Configuration
webserver:
  type: nginx  # nginx, apache
  reload_cmd: systemctl reload nginx

# Notification Configuration
notification:
  email:
    smtp: ""
    port: 587
    username: ""
    password: ""
    from: ""
    to: ""
EOF
        
        chmod 644 "$config_file"
        log_info "Default configuration file created: $config_file"
    else
        log_info "Configuration file already exists, skipping creation"
    fi
}

# Detect and configure web server
detect_webserver() {
    log_info "Detecting web server..."
    
    local webserver=""
    
    if systemctl is-active --quiet nginx 2>/dev/null || command -v nginx >/dev/null; then
        webserver="nginx"
        log_info "Detected Nginx"
    elif systemctl is-active --quiet apache2 2>/dev/null || systemctl is-active --quiet httpd 2>/dev/null; then
        webserver="apache"
        log_info "Detected Apache"
    else
        log_warn "No supported web server detected (nginx/apache)"
        return
    fi
    
    # Update web server type in configuration file
    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        sed -i "s/type: nginx/type: $webserver/" "$CONFIG_DIR/config.yaml"
        log_info "Configuration file updated with web server type: $webserver"
    fi
}

# Setup command line completion
setup_completion() {
    log_info "Setting up command line completion..."
    
    # Bash completion
    if [[ -d /etc/bash_completion.d ]]; then
        "$INSTALL_DIR/autocert" completion bash > /etc/bash_completion.d/autocert
        log_info "Bash completion installed"
    fi
    
    # Zsh completion
    if [[ -d /usr/share/zsh/vendor-completions ]]; then
        "$INSTALL_DIR/autocert" completion zsh > /usr/share/zsh/vendor-completions/_autocert
        log_info "Zsh completion installed"
    fi
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    if [[ ! -f "$INSTALL_DIR/autocert" ]]; then
        error_exit "Installation verification failed: binary file does not exist at $INSTALL_DIR/autocert"
    fi
    
    if [[ ! -x "$INSTALL_DIR/autocert" ]]; then
        error_exit "Installation verification failed: binary file is not executable"
    fi
    
    # Test command execution
    log_info "Testing autocert command..."
    local test_output
    if test_output=$("$INSTALL_DIR/autocert" --version 2>&1); then
        log_info "Command test successful: $test_output"
    else
        # Try help command as fallback
        if test_output=$("$INSTALL_DIR/autocert" --help 2>&1); then
            log_info "Command test successful (via --help)"
        else
            error_exit "Installation verification failed: command execution failed"
        fi
    fi
    
    # Verify file size (should be larger than 1MB for a typical Go binary)
    local file_size
    if file_size=$(stat -c%s "$INSTALL_DIR/autocert" 2>/dev/null); then
        if [[ $file_size -lt 1048576 ]]; then  # 1MB
            log_warn "Binary file seems unusually small ($file_size bytes), installation may be incomplete"
        else
            log_debug "Binary file size: $file_size bytes"
        fi
    fi
    
    log_info "Installation verification successful"
}

# Display post-installation information
show_post_install_info() {
    echo
    echo -e "${GREEN}🎉 AutoCert Installation Successful!${NC}"
    echo
    echo "Installation Information:"
    echo "  - Binary file: $INSTALL_DIR/autocert"
    echo "  - Configuration directory: $CONFIG_DIR"
    echo "  - Configuration file: $CONFIG_DIR/config.yaml"
    echo "  - Certificate directory: $CONFIG_DIR/certs"
    echo "  - Log directory: /var/log"
    echo
    echo "Quick Start:"
    echo "  1. Test installation:"
    echo "     autocert --version"
    echo
    echo "  2. Configure email and domain:"
    echo "     autocert install --domain your-domain.com --email your-email@example.com --nginx"
    echo
    echo "  3. Setup automatic renewal:"
    echo "     autocert schedule install"
    echo
    echo "  4. Check certificate status:"
    echo "     autocert status"
    echo
    echo "  5. View help:"
    echo "     autocert --help"
    echo
    echo -e "${YELLOW}Important Notes:${NC}"
    echo "  - Ensure your domain points to this server"
    echo "  - Ports 80 and 443 must be accessible from the internet"
    echo "  - Configuration file location: $CONFIG_DIR/config.yaml"
    echo
    echo -e "${BLUE}For more information visit: https://github.com/$GITHUB_REPO${NC}"
}

# Main function
main() {
    log_info "Starting AutoCert installation..."
    
    # Check permissions
    check_root
    
    # Detect system environment
    detect_os
    detect_arch
    
    # Install dependencies
    install_dependencies
    
    # Download and install
    download_binary
    
    # Create configuration
    create_config_dir
    create_default_config
    
    # Detect web server
    detect_webserver
    
    # Setup completion
    setup_completion
    
    # Verify installation
    verify_installation
    
    # Display post-installation info
    show_post_install_info
    
    log_info "AutoCert installation completed!"
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi