# AutoCert 一键打包脚本 - Windows 版本
# 格式: autocert_${VERSION}_windows_${ARCH}.zip
#       autocert_${VERSION}_linux_${ARCH}.tar.gz

param(
    [string]$Version = "dev",
    [string]$DistDir = "dist",
    [string]$BinaryName = "autocert",
    [string]$Platform = "all",  # all, linux, windows, darwin
    [switch]$Verbose
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
    
    if ($Level -eq "DEBUG" -and -not $Verbose) {
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

# 检查必需工具
function Test-RequiredTools {
    Write-Info "Checking required tools..."
    
    $missingTools = @()
    
    # 检查 Go
    try {
        $null = Get-Command "go" -ErrorAction Stop
        Write-Debug "Go tool found"
    } catch {
        $missingTools += "go"
    }
    
    # 检查 tar（Git for Windows 通常包含）
    try {
        $null = Get-Command "tar" -ErrorAction Stop
        Write-Debug "tar tool found"
    } catch {
        Write-Warn "tar tool not found, will use PowerShell compression"
    }
    
    # 检查 Git（用于获取版本信息）
    try {
        $null = Get-Command "git" -ErrorAction Stop
        Write-Debug "Git tool found"
    } catch {
        Write-Warn "Git tool not found, will use default version info"
    }
    
    if ($missingTools.Count -gt 0) {
        Stop-OnError "Missing required tools: $($missingTools -join ', ')"
    }
    
    Write-Info "Tool check passed"
}

# 获取构建标志
function Get-BuildFlags {
    $buildTime = Get-Date -Format "yyyy-MM-dd_HH:mm:ss" -AsUTC
    
    try {
        $commitHash = (git rev-parse --short HEAD 2>$null)
        if (-not $commitHash) { $commitHash = "unknown" }
    } catch {
        $commitHash = "unknown"
    }
    
    $flags = "-ldflags `"-X main.version=$Version -X main.buildTime=$buildTime -X main.commitHash=$commitHash`""
    Write-Debug "Build flags: $flags"
    return $flags
}

# 构建二进制文件
function Build-Binary {
    param(
        [string]$GOOS,
        [string]$GOARCH,
        [string]$OutputPath,
        [string]$BuildFlags
    )
    
    Write-Info "Building $GOOS/$GOARCH..."
    
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    
    try {
        $buildCmd = "go build $BuildFlags -o `"$OutputPath`" ."
        Write-Debug "Build command: $buildCmd"
        
        Invoke-Expression $buildCmd
        
        if (Test-Path $OutputPath) {
            Write-Info "Build successful: $OutputPath"
            return $true
        } else {
            throw "Build output file does not exist"
        }
    } catch {
        Write-Error "Build failed: $GOOS/$GOARCH - $($_.Exception.Message)"
        return $false
    } finally {
        # 清理环境变量
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    }
}

# 创建 Linux 发布包
function New-LinuxPackage {
    param(
        [string]$Arch,
        [string]$BuildFlags
    )
    
    Write-Info "打包 Linux $Arch..."
    
    $binaryPath = Join-Path $DistDir $BinaryName
    $packageName = "${BinaryName}_${Version}_linux_${Arch}.tar.gz"
    $packagePath = Join-Path $DistDir $packageName
    
    # 构建二进制文件
    if (-not (Build-Binary "linux" $Arch $binaryPath $BuildFlags)) {
        return $false
    }
    
    # 创建 tar.gz 包
    try {
        if (Get-Command "tar" -ErrorAction SilentlyContinue) {
            # 使用 tar 命令
            $tarCmd = "tar -czf `"$packagePath`" -C `"$DistDir`" `"$BinaryName`""
            Invoke-Expression $tarCmd
        } else {
            # 使用 PowerShell 压缩（备用方案）
            Write-Warn "使用 PowerShell 压缩（可能不是标准的 tar.gz 格式）"
            Compress-Archive -Path $binaryPath -DestinationPath ($packagePath -replace '\.tar\.gz$', '.zip') -Force
        }
        
        # 清理临时文件
        Remove-Item $binaryPath -Force -ErrorAction SilentlyContinue
        
        Write-Info "Linux $Arch 打包完成: $packageName"
        Write-Host "  📦 $packagePath" -ForegroundColor Cyan
        return $true
    } catch {
        Write-Error "打包失败: $($_.Exception.Message)"
        return $false
    }
}

# 创建 Windows 发布包
function New-WindowsPackage {
    param(
        [string]$Arch,
        [string]$BuildFlags
    )
    
    Write-Info "打包 Windows $Arch..."
    
    $binaryPath = Join-Path $DistDir "$BinaryName.exe"
    $packageName = "${BinaryName}_${Version}_windows_${Arch}.zip"
    $packagePath = Join-Path $DistDir $packageName
    
    # 构建二进制文件
    if (-not (Build-Binary "windows" $Arch $binaryPath $BuildFlags)) {
        return $false
    }
    
    # 创建 zip 包
    try {
        Compress-Archive -Path $binaryPath -DestinationPath $packagePath -Force
        
        # 清理临时文件
        Remove-Item $binaryPath -Force -ErrorAction SilentlyContinue
        
        Write-Info "Windows $Arch 打包完成: $packageName"
        Write-Host "  📦 $packagePath" -ForegroundColor Cyan
        return $true
    } catch {
        Write-Error "打包失败: $($_.Exception.Message)"
        return $false
    }
}

# 创建 macOS 发布包
function New-DarwinPackage {
    param(
        [string]$Arch,
        [string]$BuildFlags
    )
    
    Write-Info "打包 macOS $Arch..."
    
    $binaryPath = Join-Path $DistDir $BinaryName
    $packageName = "${BinaryName}_${Version}_darwin_${Arch}.tar.gz"
    $packagePath = Join-Path $DistDir $packageName
    
    # 构建二进制文件
    if (-not (Build-Binary "darwin" $Arch $binaryPath $BuildFlags)) {
        return $false
    }
    
    # 创建 tar.gz 包
    try {
        if (Get-Command "tar" -ErrorAction SilentlyContinue) {
            # 使用 tar 命令
            $tarCmd = "tar -czf `"$packagePath`" -C `"$DistDir`" `"$BinaryName`""
            Invoke-Expression $tarCmd
        } else {
            # 使用 PowerShell 压缩（备用方案）
            Write-Warn "使用 PowerShell 压缩（可能不是标准的 tar.gz 格式）"
            Compress-Archive -Path $binaryPath -DestinationPath ($packagePath -replace '\.tar\.gz$', '.zip') -Force
        }
        
        # 清理临时文件
        Remove-Item $binaryPath -Force -ErrorAction SilentlyContinue
        
        Write-Info "macOS $Arch 打包完成: $packageName"
        Write-Host "  📦 $packagePath" -ForegroundColor Cyan
        return $true
    } catch {
        Write-Error "打包失败: $($_.Exception.Message)"
        return $false
    }
}

# 主函数
function main {
    Write-Host "🚀 AutoCert 一键打包工具" -ForegroundColor Green
    Write-Info "版本: $Version"
    Write-Info "输出目录: $DistDir"
    Write-Info "二进制名称: $BinaryName"
    Write-Info "平台: $Platform"
    
    try {
        # 检查工具
        Test-RequiredTools
        
        # 创建输出目录
        if (-not (Test-Path $DistDir)) {
            New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
            Write-Debug "创建输出目录: $DistDir"
        }
        
        # 获取构建标志
        $buildFlags = Get-BuildFlags
        
        # 打包计数器
        $successCount = 0
        $totalCount = 0
        
        # 根据平台参数进行打包
        switch ($Platform.ToLower()) {
            "all" {
                Write-Info "打包所有支持的平台..."
                
                # Linux 平台
                $totalCount += 2
                if (New-LinuxPackage "amd64" $buildFlags) { $successCount++ }
                if (New-LinuxPackage "arm64" $buildFlags) { $successCount++ }
                
                # Windows 平台
                $totalCount += 2
                if (New-WindowsPackage "amd64" $buildFlags) { $successCount++ }
                if (New-WindowsPackage "arm64" $buildFlags) { $successCount++ }
                
                # macOS 平台
                $totalCount += 2
                if (New-DarwinPackage "amd64" $buildFlags) { $successCount++ }
                if (New-DarwinPackage "arm64" $buildFlags) { $successCount++ }
            }
            "linux" {
                Write-Info "打包 Linux 平台..."
                $totalCount += 2
                if (New-LinuxPackage "amd64" $buildFlags) { $successCount++ }
                if (New-LinuxPackage "arm64" $buildFlags) { $successCount++ }
            }
            "windows" {
                Write-Info "打包 Windows 平台..."
                $totalCount += 2
                if (New-WindowsPackage "amd64" $buildFlags) { $successCount++ }
                if (New-WindowsPackage "arm64" $buildFlags) { $successCount++ }
            }
            { $_ -in @("darwin", "macos") } {
                Write-Info "打包 macOS 平台..."
                $totalCount += 2
                if (New-DarwinPackage "amd64" $buildFlags) { $successCount++ }
                if (New-DarwinPackage "arm64" $buildFlags) { $successCount++ }
            }
            default {
                Stop-OnError "不支持的平台: $Platform`n支持的平台: all, linux, windows, darwin"
            }
        }
        
        # 显示结果
        Write-Host ""
        Write-Info "打包完成! 生成的文件:"
        
        $packages = Get-ChildItem -Path $DistDir -Include "*.tar.gz", "*.zip" -ErrorAction SilentlyContinue
        if ($packages) {
            $packages | ForEach-Object {
                $size = [math]::Round($_.Length / 1MB, 2)
                Write-Host "  📦 $($_.Name) (${size} MB)" -ForegroundColor Cyan
            }
        }
        
        # 统计信息
        $totalSize = (Get-ChildItem -Path $DistDir -Include "*.tar.gz", "*.zip" -ErrorAction SilentlyContinue | 
                     Measure-Object -Property Length -Sum).Sum
        $totalSizeMB = [math]::Round($totalSize / 1MB, 2)
        
        Write-Host ""
        Write-Host "📊 统计信息:" -ForegroundColor Yellow
        Write-Host "  📁 输出目录: $DistDir" -ForegroundColor White
        Write-Host "  📦 成功打包: $successCount/$totalCount" -ForegroundColor White
        Write-Host "  💾 总大小: ${totalSizeMB} MB" -ForegroundColor White
        
        if ($successCount -eq $totalCount) {
            Write-Host ""
            Write-Host "🎉 所有发布包已准备就绪!" -ForegroundColor Green
        } else {
            Write-Host ""
            Write-Host "⚠️  部分打包失败，请检查日志" -ForegroundColor Yellow
        }
        
    } catch {
        Write-Error "打包过程中发生错误: $($_.Exception.Message)"
        exit 1
    }
}

# 脚本入口
if ($MyInvocation.InvocationName -ne '.') {
    main
}