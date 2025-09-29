# AutoCert 一键安装脚本 - Windows 版本
# 支持 Windows 10/11 和 Windows Server

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:ProgramFiles\AutoCert",
    [string]$ConfigDir = "$env:ProgramData\AutoCert",
    [switch]$Force,
    [switch]$Debug
)

# 设置错误处理
$ErrorActionPreference = "Stop"

# 日志函数
function Write-Log {
    param(
        [string]$Message,
        [string]$Level = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "INFO" { "Green" }
        "WARN" { "Yellow" }
        "ERROR" { "Red" }
        "DEBUG" { "Cyan" }
        default { "White" }
    }
    
    if ($Level -eq "DEBUG" -and -not $Debug) {
        return
    }
    
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

function Write-Info { param([string]$Message) Write-Log $Message "INFO" }
function Write-Warn { param([string]$Message) Write-Log $Message "WARN" }
function Write-Error { param([string]$Message) Write-Log $Message "ERROR" }
function Write-Debug { param([string]$Message) Write-Log $Message "DEBUG" }

# 错误处理函数
function Stop-OnError {
    param([string]$Message)
    Write-Error $Message
    exit 1
}

# 检查管理员权限
function Test-AdminRights {
    Write-Info "检查管理员权限..."
    
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    $isAdmin = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (-not $isAdmin) {
        Stop-OnError "此脚本需要管理员权限运行。请右键点击 PowerShell 并选择 '以管理员身份运行'。"
    }
    
    Write-Info "管理员权限检查通过"
}

# 检测系统信息
function Get-SystemInfo {
    Write-Info "检测系统信息..."
    
    $os = Get-WmiObject -Class Win32_OperatingSystem
    $arch = $env:PROCESSOR_ARCHITECTURE
    
    Write-Info "操作系统: $($os.Caption)"
    Write-Info "架构: $arch"
    
    # 转换架构名称
    $script:Architecture = switch ($arch) {
        "AMD64" { "amd64" }
        "ARM64" { "arm64" }
        default { Stop-OnError "不支持的系统架构: $arch" }
    }
    
    Write-Debug "转换后的架构: $script:Architecture"
}

# 检查并安装依赖
function Install-Dependencies {
    Write-Info "检查系统依赖..."
    
    # 检查 PowerShell 版本
    if ($PSVersionTable.PSVersion.Major -lt 3) {
        Stop-OnError "需要 PowerShell 3.0 或更高版本"
    }
    
    # 检查 .NET Framework
    $dotNetVersion = Get-ItemProperty "HKLM:SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\" -Name Release -ErrorAction SilentlyContinue
    if (-not $dotNetVersion -or $dotNetVersion.Release -lt 461808) {
        Write-Warn ".NET Framework 4.7.2 或更高版本未安装，建议升级"
    }
    
    Write-Info "依赖检查完成"
}

# 下载 AutoCert 二进制文件
function Get-AutoCertBinary {
    Write-Info "下载 AutoCert 二进制文件..."
    
    $repoUrl = "https://api.github.com/repos/autocert/autocert"  # 替换为实际仓库
    $tempDir = "$env:TEMP\AutoCert"
    $tempFile = "$tempDir\autocert.zip"
    
    # 创建临时目录
    if (Test-Path $tempDir) {
        Remove-Item $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    try {
        # 获取最新版本信息
        if ($Version -eq "latest") {
            Write-Info "获取最新版本信息..."
            $releaseInfo = Invoke-WebRequest -Uri "$repoUrl/releases/latest" -UseBasicParsing | ConvertFrom-Json
            $Version = $releaseInfo.tag_name
            Write-Info "最新版本: $Version"
        }
        
        # 构建下载 URL
        $downloadUrl = "https://github.com/autocert/autocert/releases/download/$Version/autocert_${Version}_windows_$($script:Architecture).zip"
        Write-Debug "下载 URL: $downloadUrl"
        
        # 下载文件
        Write-Info "正在下载: $downloadUrl"
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing
        
        # 解压文件
        Write-Info "解压文件..."
        Expand-Archive -Path $tempFile -DestinationPath $tempDir -Force
        
        # 检查二进制文件是否存在
        $binaryPath = "$tempDir\autocert.exe"
        if (-not (Test-Path $binaryPath)) {
            Stop-OnError "解压后未找到 autocert.exe"
        }
        
        # 创建安装目录
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        
        # 复制二进制文件
        $targetPath = "$InstallDir\autocert.exe"
        Copy-Item $binaryPath $targetPath -Force
        
        Write-Info "二进制文件安装完成: $targetPath"
        
    } catch {
        Stop-OnError "下载失败: $($_.Exception.Message)"
    } finally {
        # 清理临时文件
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# 创建配置目录和文件
function New-Configuration {
    Write-Info "创建配置目录和文件..."
    
    # 创建配置目录
    $certDir = "$ConfigDir\certs"
    $logDir = "$ConfigDir\logs"
    
    @($ConfigDir, $certDir, $logDir) | ForEach-Object {
        if (-not (Test-Path $_)) {
            New-Item -ItemType Directory -Path $_ -Force | Out-Null
            Write-Debug "创建目录: $_"
        }
    }
    
    # 创建默认配置文件
    $configFile = "$ConfigDir\config.yaml"
    if (-not (Test-Path $configFile) -or $Force) {
        $configContent = @"
# AutoCert 配置文件
log_level: info
config_dir: $ConfigDir
cert_dir: $certDir
log_dir: $logDir

# ACME 配置
acme:
  server: https://acme-v02.api.letsencrypt.org/directory
  key_type: rsa
  key_size: 2048

# Web 服务器配置
webserver:
  type: iis  # iis, nginx
  reload_cmd: iisreset

# 通知配置
notification:
  email:
    smtp: ""
    port: 587
    username: ""
    password: ""
    from: ""
    to: ""
"@
        
        Set-Content -Path $configFile -Value $configContent -Encoding UTF8
        Write-Info "默认配置文件创建完成: $configFile"
    } else {
        Write-Info "配置文件已存在，跳过创建"
    }
}

# 检测 Web 服务器
function Test-WebServer {
    Write-Info "检测 Web 服务器..."
    
    # 检测 IIS
    $iisFeature = Get-WindowsFeature -Name IIS-WebServer -ErrorAction SilentlyContinue
    if ($iisFeature -and $iisFeature.InstallState -eq "Installed") {
        Write-Info "检测到 IIS"
        return
    }
    
    # 检测 Nginx for Windows
    $nginxPaths = @(
        "C:\nginx\nginx.exe",
        "C:\Program Files\nginx\nginx.exe",
        "$env:ProgramFiles\nginx\nginx.exe"
    )
    
    foreach ($path in $nginxPaths) {
        if (Test-Path $path) {
            Write-Info "检测到 Nginx: $path"
            
            # 更新配置文件
            $configFile = "$ConfigDir\config.yaml"
            if (Test-Path $configFile) {
                $content = Get-Content $configFile -Raw
                $content = $content -replace "type: iis", "type: nginx"
                Set-Content -Path $configFile -Value $content -Encoding UTF8
                Write-Info "配置文件已更新 Web 服务器类型: nginx"
            }
            return
        }
    }
    
    Write-Warn "未检测到支持的 Web 服务器 (IIS/Nginx)"
}

# 添加到 PATH 环境变量
function Add-ToPath {
    Write-Info "添加到 PATH 环境变量..."
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
    
    if ($currentPath -notlike "*$InstallDir*") {
        $newPath = "$currentPath;$InstallDir"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
        Write-Info "已添加到系统 PATH: $InstallDir"
        
        # 更新当前会话的 PATH
        $env:PATH += ";$InstallDir"
    } else {
        Write-Info "PATH 中已包含安装目录"
    }
}

# 创建防火墙规则
function New-FirewallRules {
    Write-Info "创建防火墙规则..."
    
    try {
        # 允许 HTTP (80) 和 HTTPS (443) 端口
        $rules = @(
            @{Name="AutoCert-HTTP"; Port=80; Protocol="TCP"},
            @{Name="AutoCert-HTTPS"; Port=443; Protocol="TCP"}
        )
        
        foreach ($rule in $rules) {
            $existingRule = Get-NetFirewallRule -DisplayName $rule.Name -ErrorAction SilentlyContinue
            if (-not $existingRule) {
                New-NetFirewallRule -DisplayName $rule.Name -Direction Inbound -Protocol $rule.Protocol -LocalPort $rule.Port -Action Allow | Out-Null
                Write-Debug "创建防火墙规则: $($rule.Name)"
            }
        }
        
        Write-Info "防火墙规则配置完成"
    } catch {
        Write-Warn "防火墙规则创建失败: $($_.Exception.Message)"
    }
}

# 验证安装
function Test-Installation {
    Write-Info "验证安装..."
    
    $binaryPath = "$InstallDir\autocert.exe"
    
    if (-not (Test-Path $binaryPath)) {
        Stop-OnError "安装验证失败: 二进制文件不存在"
    }
    
    try {
        $output = & $binaryPath --version 2>&1
        Write-Debug "命令输出: $output"
        Write-Info "安装验证成功"
    } catch {
        Stop-OnError "安装验证失败: 命令执行失败"
    }
}

# 显示安装后信息
function Show-PostInstallInfo {
    Write-Host ""
    Write-Host "🎉 AutoCert 安装成功！" -ForegroundColor Green
    Write-Host ""
    Write-Host "安装信息:" -ForegroundColor Cyan
    Write-Host "  - 二进制文件: $InstallDir\autocert.exe"
    Write-Host "  - 配置目录: $ConfigDir"
    Write-Host "  - 配置文件: $ConfigDir\config.yaml"
    Write-Host "  - 证书目录: $ConfigDir\certs"
    Write-Host "  - 日志目录: $ConfigDir\logs"
    Write-Host ""
    Write-Host "快速开始:" -ForegroundColor Cyan
    Write-Host "  1. 打开新的 PowerShell 窗口（以刷新 PATH）"
    Write-Host "  2. 配置邮箱和域名:"
    Write-Host "     autocert install --domain your-domain.com --email your-email@example.com --iis"
    Write-Host ""
    Write-Host "  3. 设置自动续期:"
    Write-Host "     autocert schedule install"
    Write-Host ""
    Write-Host "  4. 查看证书状态:"
    Write-Host "     autocert status"
    Write-Host ""
    Write-Host "  5. 查看帮助:"
    Write-Host "     autocert --help"
    Write-Host ""
    Write-Host "注意事项:" -ForegroundColor Yellow
    Write-Host "  - 请重新打开 PowerShell 窗口以刷新 PATH 环境变量"
    Write-Host "  - 确保防火墙允许 80 和 443 端口的入站连接"
    Write-Host "  - 配置文件位于: $ConfigDir\config.yaml"
    Write-Host ""
    Write-Host "更多信息请访问: https://github.com/autocert/autocert" -ForegroundColor Blue
}

# 主函数
function main {
    Write-Info "开始安装 AutoCert..."
    
    try {
        # 检查权限
        Test-AdminRights
        
        # 检测系统
        Get-SystemInfo
        
        # 安装依赖
        Install-Dependencies
        
        # 下载并安装
        Get-AutoCertBinary
        
        # 创建配置
        New-Configuration
        
        # 检测 Web 服务器
        Test-WebServer
        
        # 添加到 PATH
        Add-ToPath
        
        # 配置防火墙
        New-FirewallRules
        
        # 验证安装
        Test-Installation
        
        # 显示安装后信息
        Show-PostInstallInfo
        
        Write-Info "AutoCert 安装完成！"
        
    } catch {
        Write-Error "安装失败: $($_.Exception.Message)"
        exit 1
    }
}

# 脚本入口
if ($MyInvocation.InvocationName -ne '.') {
    main
}