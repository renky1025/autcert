# AutoCert 一键打包指南

本文档介绍 AutoCert 项目的一键打包功能，支持生成标准格式的跨平台发布包。

## 📦 支持的包格式

### Linux/macOS 包格式
```
autocert_${VERSION}_linux_${ARCH}.tar.gz
autocert_${VERSION}_darwin_${ARCH}.tar.gz
```

### Windows 包格式
```
autocert_${VERSION}_windows_${ARCH}.zip
```

### 支持的架构
- `amd64` - x86_64 架构
- `arm64` - ARM64 架构

## 🚀 一键打包方法

### 方法一：使用 Makefile（推荐）

```bash
# 打包所有平台
make package

# 打包特定平台
make package-linux
make package-windows

# 完整发布流程（清理+测试+打包）
make release

# 快速打包（跳过测试）
make quick-package
```

### 方法二：直接使用打包脚本

#### Linux/macOS 环境
```bash
# 打包所有平台
./scripts/package.sh

# 指定版本和平台
./scripts/package.sh v1.0.0 dist autocert all
./scripts/package.sh v1.0.0 dist autocert linux
./scripts/package.sh v1.0.0 dist autocert windows
```

#### Windows 环境
```powershell
# PowerShell 脚本
.\scripts\package.ps1 -Version "v1.0.0" -Platform "all"
.\scripts\package.ps1 -Version "v1.0.0" -Platform "windows"

# 批处理脚本
.\scripts\build-release.bat v1.0.0 all
```

### 方法三：跨平台快捷脚本

```bash
# Linux/macOS/Git Bash
./scripts/build-release.sh v1.0.0 all

# Windows 命令提示符
.\scripts\build-release.bat v1.0.0 all
```

## 📋 脚本参数说明

### Linux 打包脚本 (package.sh)
```bash
./scripts/package.sh [VERSION] [DIST_DIR] [BINARY_NAME] [PLATFORM]
```

**参数：**
- `VERSION`: 版本号（默认：从 git 获取或 "dev"）
- `DIST_DIR`: 输出目录（默认：dist）
- `BINARY_NAME`: 二进制文件名（默认：autocert）
- `PLATFORM`: 目标平台（all/linux/windows/darwin，默认：all）

### Windows 打包脚本 (package.ps1)
```powershell
.\scripts\package.ps1 -Version "v1.0.0" -DistDir "dist" -BinaryName "autocert" -Platform "all" [-Verbose]
```

**参数：**
- `-Version`: 版本号
- `-DistDir`: 输出目录
- `-BinaryName`: 二进制文件名
- `-Platform`: 目标平台
- `-Verbose`: 显示详细日志

## 📁 输出结构

打包完成后，`dist` 目录结构如下：

```
dist/
├── autocert_v1.0.0_linux_amd64.tar.gz
├── autocert_v1.0.0_linux_arm64.tar.gz
├── autocert_v1.0.0_windows_amd64.zip
├── autocert_v1.0.0_windows_arm64.zip
├── autocert_v1.0.0_darwin_amd64.tar.gz
└── autocert_v1.0.0_darwin_arm64.tar.gz
```

## 🔧 自定义配置

### 修改默认设置

在 `Makefile` 中修改默认配置：

```makefile
# 变量定义
BINARY_NAME=autocert
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DIST_DIR=dist
```

### 添加新的架构支持

在打包脚本中添加新的架构：

```bash
# 在 package.sh 中添加
package_linux "arm" "${build_flags}"
package_windows "386" "${build_flags}"
```

## 🎯 使用场景

### 1. 开发版本打包
```bash
# 快速打包当前开发版本
make quick-package
```

### 2. 正式版本发布
```bash
# 完整发布流程
make release
```

### 3. 特定平台打包
```bash
# 只打包 Linux 版本
make package-linux

# 只打包 Windows 版本
make package-windows
```

### 4. CI/CD 集成
```yaml
# GitHub Actions 示例
- name: Package Release
  run: |
    make package
    ls -la dist/
```

### 5. 手动指定版本
```bash
# 指定特定版本号
VERSION=v1.2.3 make package
```

## 🛠️ 故障排除

### 常见问题

1. **权限错误**
   ```bash
   chmod +x scripts/*.sh
   ```

2. **Go 环境未配置**
   ```bash
   go version  # 检查 Go 是否安装
   ```

3. **Git 未安装**
   - Windows: 安装 Git for Windows
   - Linux: `sudo apt install git` 或 `sudo yum install git`

4. **tar 命令未找到 (Windows)**
   - 安装 Git for Windows（包含 tar）
   - 或使用 WSL

### 调试模式

```bash
# Linux - 启用详细输出
DEBUG=1 ./scripts/package.sh

# Windows - 启用详细输出
.\scripts\package.ps1 -Verbose
```

### 检查工具依赖

```bash
# 检查必需工具
go version
git --version
tar --version  # Linux/macOS
zip --version  # 可选
```

## 📊 性能优化

### 并行构建

修改脚本以支持并行构建：

```bash
# 并行构建多个平台
package_linux "amd64" "${build_flags}" &
package_linux "arm64" "${build_flags}" &
wait
```

### 缓存优化

```bash
# 启用 Go 模块缓存
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org
```

## 🔗 相关命令

```bash
# 查看所有可用目标
make help

# 清理构建文件
make clean

# 运行测试
make test

# 格式化代码
make fmt

# 代码检查
make lint
```

## 📚 扩展阅读

- [Go 交叉编译指南](https://golang.org/doc/install/source#environment)
- [Make 使用手册](https://www.gnu.org/software/make/manual/)
- [PowerShell 脚本开发](https://docs.microsoft.com/en-us/powershell/)

---

通过以上打包系统，您可以轻松地为 AutoCert 项目生成标准格式的跨平台发布包，满足不同用户的部署需求。