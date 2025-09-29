#!/bin/bash

# AutoCert 跨平台一键打包快捷脚本

# 检测操作系统
OS=$(uname -s)
ARCH=$(uname -m)

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 参数处理
VERSION=${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
PLATFORM=${2:-"all"}

echo "🚀 AutoCert 跨平台一键打包"
echo "操作系统: $OS"
echo "架构: $ARCH" 
echo "版本: $VERSION"
echo "平台: $PLATFORM"
echo ""

# 切换到项目根目录
cd "$PROJECT_ROOT"

case "$OS" in
    "Linux"|"Darwin")
        echo "使用 Linux/macOS 打包脚本..."
        chmod +x scripts/package.sh
        scripts/package.sh "$VERSION" "dist" "autocert" "$PLATFORM"
        ;;
    "MINGW"*|"MSYS"*|"CYGWIN"*)
        echo "使用 Windows 打包脚本..."
        powershell -ExecutionPolicy Bypass -File scripts/package.ps1 -Version "$VERSION" -Platform "$PLATFORM"
        ;;
    *)
        echo "❌ 不支持的操作系统: $OS"
        echo "支持的系统: Linux, macOS, Windows"
        exit 1
        ;;
esac

echo ""
echo "📁 打包结果:"
ls -la dist/ 2>/dev/null || dir dist\ 2>/dev/null || echo "打包目录不存在"