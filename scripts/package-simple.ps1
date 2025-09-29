# AutoCert Simple Package Script
param(
    [string]$Version = "dev",
    [string]$Platform = "windows"
)

$BinaryName = "autocert"
$DistDir = "dist"

Write-Host "AutoCert Package Tool" -ForegroundColor Green
Write-Host "Version: $Version" -ForegroundColor Yellow
Write-Host "Platform: $Platform" -ForegroundColor Yellow

# Create output directory
if (-not (Test-Path $DistDir)) {
    New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
}

# Get build info
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd_HH:mm:ss")
try {
    $CommitHash = (git rev-parse --short HEAD 2>$null)
    if (-not $CommitHash) { $CommitHash = "unknown" }
} catch {
    $CommitHash = "unknown"
}

$BuildFlags = "-ldflags `"-X main.version=$Version -X main.buildTime=$BuildTime -X main.commitHash=$CommitHash`""

# Package function
function New-Package {
    param($OS, $Arch)
    
    $env:GOOS = $OS
    $env:GOARCH = $Arch
    
    $Extension = if ($OS -eq "windows") { ".exe" } else { "" }
    $BinaryPath = "$DistDir/${BinaryName}$Extension"
    
    Write-Host "Building $OS/$Arch..." -ForegroundColor Cyan
    
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
                Write-Warning "Using .zip format instead of .tar.gz (tar command not available)"
                $PackageName = $ZipName
            }
        }
        
        Remove-Item $BinaryPath -Force
        Write-Host "Success: $PackageName" -ForegroundColor Green
        
    } catch {
        Write-Host "Failed: $OS/$Arch - $($_.Exception.Message)" -ForegroundColor Red
    } finally {
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    }
}

# Package based on platform parameter
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
        Write-Host "Unsupported platform: $Platform" -ForegroundColor Red
        Write-Host "Supported platforms: all, linux, windows, darwin" -ForegroundColor Yellow
        exit 1
    }
}

Write-Host ""
Write-Host "Packaging completed! Files:" -ForegroundColor Green
Get-ChildItem $DistDir -Include "*.zip", "*.tar.gz" | ForEach-Object {
    $size = [math]::Round($_.Length / 1MB, 2)
    Write-Host "  $($_.Name) - ${size} MB" -ForegroundColor Cyan
}