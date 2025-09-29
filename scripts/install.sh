#!/bin/bash

# AutoCert 一键安装脚本 - Linux 版本
# 支持 Ubuntu, CentOS, Debian, AlmaLinux 等主流发行版

set -euo pipefail

# 配置变量
PROGRAM_NAME="autocert"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/autocert"
SERVICE_NAME="autocert"
GITHUB_REPO="renky1025/autcert"  # 替换为实际的 GitHub 仓库
VERSION="latest"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 错误处理
error_exit() {
    log_error "$1"
    exit 1
}

# 检查是否为 root 用户
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error_exit "此脚本需要 root 权限运行。请使用 sudo 执行。"
    fi
}

# 检测操作系统
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        VER=$VERSION_ID
    elif type lsb_release >/dev/null 2>&1; then
        OS=$(lsb_release -si | tr '[:upper:]' '[:lower:]')
        VER=$(lsb_release -sr)
    else
        error_exit "无法检测操作系统类型"
    fi
    
    log_info "检测到操作系统: $OS $VER"
}

# 检测架构
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
            error_exit "不支持的架构: $ARCH"
            ;;
    esac
    log_info "检测到架构: $ARCH"
}

# 安装依赖
install_dependencies() {
    log_info "安装系统依赖..."
    
    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y curl wget unzip tar openssl ca-certificates
            ;;
        centos|rhel|almalinux|rocky)
            if command -v dnf >/dev/null; then
                dnf install -y curl wget unzip tar openssl ca-certificates
            elif command -v yum >/dev/null; then
                yum install -y curl wget unzip tar openssl ca-certificates
            else
                error_exit "无法找到包管理器 (dnf/yum)"
            fi
            ;;
        *)
            log_warn "未知的操作系统，跳过依赖安装"
            ;;
    esac
}

# 下载 AutoCert 二进制文件
download_binary() {
    log_info "下载 AutoCert 二进制文件..."
    
    local download_url=""
    local temp_file="/tmp/autocert_${VERSION}.tar.gz"
    
    if [[ "$VERSION" == "latest" ]]; then
        # 获取最新版本号
        local latest_version
        latest_version=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        VERSION=$latest_version
    fi
    
    download_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/autocert_${VERSION}_linux_${ARCH}.tar.gz"
    
    log_info "下载地址: $download_url"
    
    # 下载文件
    if ! curl -L -o "$temp_file" "$download_url"; then
        error_exit "下载失败: $download_url"
    fi
    
    # 解压到临时目录
    local temp_dir="/tmp/autocert_extract"
    mkdir -p "$temp_dir"
    tar -xzf "$temp_file" -C "$temp_dir"
    
    # 安装二进制文件
    if [[ -f "$temp_dir/autocert" ]]; then
        install -m 755 "$temp_dir/autocert" "$INSTALL_DIR/autocert"
        log_info "二进制文件安装完成: $INSTALL_DIR/autocert"
    else
        error_exit "解压后未找到 autocert 二进制文件"
    fi
    
    # 清理临时文件
    rm -rf "$temp_file" "$temp_dir"
}

# 创建配置目录
create_config_dir() {
    log_info "创建配置目录..."
    
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$CONFIG_DIR/certs"
    mkdir -p "/var/log"
    
    # 设置权限
    chmod 755 "$CONFIG_DIR"
    chmod 700 "$CONFIG_DIR/certs"
    
    log_info "配置目录创建完成: $CONFIG_DIR"
}

# 创建默认配置文件
create_default_config() {
    log_info "创建默认配置文件..."
    
    local config_file="$CONFIG_DIR/config.yaml"
    
    if [[ ! -f "$config_file" ]]; then
        cat > "$config_file" << EOF
# AutoCert 配置文件
log_level: info
config_dir: $CONFIG_DIR
cert_dir: $CONFIG_DIR/certs
log_dir: /var/log

# ACME 配置
acme:
  server: https://acme-v02.api.letsencrypt.org/directory
  key_type: rsa
  key_size: 2048

# Web 服务器配置
webserver:
  type: nginx  # nginx, apache
  reload_cmd: systemctl reload nginx

# 通知配置
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
        log_info "默认配置文件创建完成: $config_file"
    else
        log_info "配置文件已存在，跳过创建"
    fi
}

# 检测并配置 Web 服务器
detect_webserver() {
    log_info "检测 Web 服务器..."
    
    local webserver=""
    
    if systemctl is-active --quiet nginx 2>/dev/null || command -v nginx >/dev/null; then
        webserver="nginx"
        log_info "检测到 Nginx"
    elif systemctl is-active --quiet apache2 2>/dev/null || systemctl is-active --quiet httpd 2>/dev/null; then
        webserver="apache"
        log_info "检测到 Apache"
    else
        log_warn "未检测到支持的 Web 服务器 (nginx/apache)"
        return
    fi
    
    # 更新配置文件中的 Web 服务器类型
    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        sed -i "s/type: nginx/type: $webserver/" "$CONFIG_DIR/config.yaml"
        log_info "配置文件已更新 Web 服务器类型: $webserver"
    fi
}

# 设置命令行补全
setup_completion() {
    log_info "设置命令行补全..."
    
    # Bash 补全
    if [[ -d /etc/bash_completion.d ]]; then
        "$INSTALL_DIR/autocert" completion bash > /etc/bash_completion.d/autocert
        log_info "Bash 补全已安装"
    fi
    
    # Zsh 补全
    if [[ -d /usr/share/zsh/vendor-completions ]]; then
        "$INSTALL_DIR/autocert" completion zsh > /usr/share/zsh/vendor-completions/_autocert
        log_info "Zsh 补全已安装"
    fi
}

# 验证安装
verify_installation() {
    log_info "验证安装..."
    
    if [[ ! -f "$INSTALL_DIR/autocert" ]]; then
        error_exit "安装验证失败: 二进制文件不存在"
    fi
    
    if [[ ! -x "$INSTALL_DIR/autocert" ]]; then
        error_exit "安装验证失败: 二进制文件不可执行"
    fi
    
    # 测试命令
    if ! "$INSTALL_DIR/autocert" --help >/dev/null 2>&1; then
        error_exit "安装验证失败: 命令执行失败"
    fi
    
    log_info "安装验证成功"
}

# 显示安装后信息
show_post_install_info() {
    echo
    echo -e "${GREEN}🎉 AutoCert 安装成功！${NC}"
    echo
    echo "安装信息:"
    echo "  - 二进制文件: $INSTALL_DIR/autocert"
    echo "  - 配置目录: $CONFIG_DIR"
    echo "  - 配置文件: $CONFIG_DIR/config.yaml"
    echo "  - 证书目录: $CONFIG_DIR/certs"
    echo "  - 日志目录: /var/log"
    echo
    echo "快速开始:"
    echo "  1. 配置邮箱和域名:"
    echo "     autocert install --domain your-domain.com --email your-email@example.com --nginx"
    echo
    echo "  2. 设置自动续期:"
    echo "     autocert schedule install"
    echo
    echo "  3. 查看证书状态:"
    echo "     autocert status"
    echo
    echo "  4. 查看帮助:"
    echo "     autocert --help"
    echo
    echo "更多信息请访问: https://github.com/$GITHUB_REPO"
}

# 主函数
main() {
    log_info "开始安装 AutoCert..."
    
    # 检查权限
    check_root
    
    # 检测系统环境
    detect_os
    detect_arch
    
    # 安装依赖
    install_dependencies
    
    # 下载并安装
    download_binary
    
    # 创建配置
    create_config_dir
    create_default_config
    
    # 检测 Web 服务器
    detect_webserver
    
    # 设置补全
    setup_completion
    
    # 验证安装
    verify_installation
    
    # 显示安装后信息
    show_post_install_info
    
    log_info "AutoCert 安装完成！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi