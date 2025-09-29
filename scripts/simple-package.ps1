# AutoCert 简化打包脚本
param(
    [string]$Version = "dev",
    [string]$Platform = "windows"
)

$BinaryName = "autocert"
$DistDir = "dist"

Write-Host "🚀 AutoCert 简化打包工具" -ForegroundColor Green
Write-Host "版本: $Version" -ForegroundColor Yellow
Write-Host "平台: $Platform" -ForegroundColor Yellow

# 创建输出目录
if (-not (Test-Path $DistDir)) {
    New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
}

# 获取构建信息
$BuildTime = Get-Date -Format "yyyy-MM-dd_HH:mm:ss" -AsUTC
try {
    $CommitHash = (git rev-parse --short HEAD 2>$null)
    if (-not $CommitHash) { $CommitHash = "unknown" }
} catch {
    $CommitHash = "unknown"
}

$BuildFlags = "-ldflags `"-X main.version=$Version -X main.buildTime=$BuildTime -X main.commitHash=$CommitHash`""

# 打包函数
function New-Package {
    param($OS, $Arch)
    
    $env:GOOS = $OS
    $env:GOARCH = $Arch
    
    $Extension = if ($OS -eq "windows") { ".exe" } else { "" }
    $BinaryPath = "$DistDir/${BinaryName}$Extension"
    
    Write-Host "构建 $OS/$Arch..." -ForegroundColor Cyan
    
    try {
        Invoke-Expression "go build $BuildFlags -o `"$BinaryPath`" ."
        
        if ($OS -eq "windows") {
            $PackageName = "${BinaryName}_${Version}_${OS}_${Arch}.zip"
            Compress-Archive -Path $BinaryPath -DestinationPath "$DistDir/$PackageName" -Force
        } else {
            $PackageName = "${BinaryName}_${Version}_${OS}_${Arch}.tar.gz"
            if (Get-Command "tar" -ErrorAction SilentlyContinue) {
                tar -czf "$DistDir/$PackageName" -C $DistDir "$BinaryName"
            } else {
                $ZipName = "${BinaryName}_${Version}_${OS}_${Arch}.zip"
                Compress-Archive -Path $BinaryPath -DestinationPath "$DistDir/$ZipName" -Force
                Write-Warning "使用 .zip 格式代替 .tar.gz（tar 命令不可用）"
                $PackageName = $ZipName
            }
        }
        
        Remove-Item $BinaryPath -Force
        Write-Host "✅ $PackageName" -ForegroundColor Green
        
    } catch {
        Write-Host "❌ 构建失败: $OS/$Arch - $($_.Exception.Message)" -ForegroundColor Red
    } finally {
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    }
}

# 根据平台参数打包
switch ($Platform.ToLower()) {
    "all" {
        New-Package "linux" "amd64"
        New-Package "linux" "arm64"
        New-Package "windows" "amd64"
        New-Package "windows" "arm64"
        New-Package "darwin" "amd64"
        New-Package "darwin" "arm64"
    }
    "linux" {
        New-Package "linux" "amd64"
        New-Package "linux" "arm64"
    }
    "windows" {
        New-Package "windows" "amd64"
        New-Package "windows" "arm64"
    }
    "darwin" {
        New-Package "darwin" "amd64"
        New-Package "darwin" "arm64"
    }
    default {
        Write-Host "❌ 不支持的平台: $Platform" -ForegroundColor Red
        Write-Host "支持的平台: all, linux, windows, darwin" -ForegroundColor Yellow
        exit 1
    }
}

Write-Host ""
Write-Host "Package completed! File list:" -ForegroundColor Green
Get-ChildItem $DistDir -Include "*.zip", "*.tar.gz" | ForEach-Object {
    $size = [math]::Round($_.Length / 1MB, 2)
    $fileName = $_.Name
    Write-Host "  File: $fileName (Size: $size MB)" -ForegroundColor Cyan
}