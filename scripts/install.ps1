# AutoCert Installation Script - Windows
# Supports Windows 10/11 and Windows Server

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:ProgramFiles\AutoCert",
    [string]$ConfigDir = "$env:ProgramData\AutoCert",
    [switch]$Force,
    [switch]$Debug
)

# Set encoding to support proper text output
try {
    # Set console encoding
    [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
    [Console]::InputEncoding = [System.Text.Encoding]::UTF8
    
    # Set PowerShell output encoding
    $OutputEncoding = [System.Text.Encoding]::UTF8
    
    # Additional settings for Windows PowerShell 5.1
    if ($PSVersionTable.PSVersion.Major -le 5) {
        $PSDefaultParameterValues['*:Encoding'] = 'utf8'
    }
    
    # Set current process code page to UTF-8
    cmd /c "chcp 65001 > nul 2>&1"
} catch {
    Write-Warning "Encoding setup may not be fully effective, but will not affect installation"
}

# Set error handling
$ErrorActionPreference = "Stop"

# Log functions
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

# Error handling function
function Stop-OnError {
    param([string]$Message)
    Write-Error $Message
    exit 1
}

# Check administrator rights
function Test-AdminRights {
    Write-Info "Checking administrator rights..."
    
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    $isAdmin = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (-not $isAdmin) {
        Stop-OnError "This script requires administrator privileges. Please right-click PowerShell and select 'Run as administrator'."
    }
    
    Write-Info "Administrator rights check passed"
}

# Detect system information
function Get-SystemInfo {
    Write-Info "Detecting system information..."
    
    $os = Get-WmiObject -Class Win32_OperatingSystem
    $arch = $env:PROCESSOR_ARCHITECTURE
    
    Write-Info "Operating System: $($os.Caption)"
    Write-Info "Architecture: $arch"
    
    # Convert architecture names
    $script:Architecture = switch ($arch) {
        "AMD64" { "amd64" }
        "ARM64" { "arm64" }
        default { Stop-OnError "Unsupported system architecture: $arch" }
    }
    
    Write-Debug "Converted architecture: $script:Architecture"
}

# Check and install dependencies
function Install-Dependencies {
    Write-Info "Checking system dependencies..."
    
    # Check PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 3) {
        Stop-OnError "PowerShell 3.0 or higher is required"
    }
    
    # Check .NET Framework
    $dotNetVersion = Get-ItemProperty "HKLM:SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\" -Name Release -ErrorAction SilentlyContinue
    if (-not $dotNetVersion -or $dotNetVersion.Release -lt 461808) {
        Write-Warn ".NET Framework 4.7.2 or higher is not installed, upgrade recommended"
    }
    
    Write-Info "Dependency check completed"
}

# Download AutoCert binary files
function Get-AutoCertBinary {
    Write-Info "Downloading AutoCert binary files..."
    
    $repoUrl = "https://api.github.com/repos/renky1025/autocert"
    $tempDir = "$env:TEMP\AutoCert"
    $tempFile = "$tempDir\autocert.zip"
    
    # Create temporary directory
    if (Test-Path $tempDir) {
        Remove-Item $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    try {
        # Get latest version information
        if ($Version -eq "latest") {
            Write-Info "Getting latest version information..."
            try {
                # Use TLS 1.2 and above protocols
                [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12, [Net.SecurityProtocolType]::Tls13
                
                # Set request parameters
                $headers = @{
                    "User-Agent" = "AutoCert-Installer/1.0"
                    "Accept" = "application/vnd.github.v3+json"
                }
                
                # Get latest version
                $releaseInfo = Invoke-RestMethod -Uri "$repoUrl/releases/latest" -Method Get -Headers $headers -TimeoutSec 30
                
                if ($releaseInfo -and $releaseInfo.tag_name) {
                    $Version = $releaseInfo.tag_name
                    Write-Info "Retrieved latest version: $Version"
                } else {
                    throw "Invalid version information response"
                }
            } catch {
                Write-Warn "Unable to get latest version information ($($_.Exception.Message)), will use default version"
                $Version = "v1.0.0-final"  # Default version
            }
        }
        
        # Build download URL
        $downloadUrl = "https://github.com/renky1025/autocert/releases/download/$Version/autocert_${Version}_windows_$($script:Architecture).zip"
        Write-Debug "Download URL: $downloadUrl"
        
        # Download file
        Write-Info "Downloading: $downloadUrl"
        try {
            # Set progress bar display
            $ProgressPreference = 'SilentlyContinue'
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -UseBasicParsing -TimeoutSec 300
            $ProgressPreference = 'Continue'
        } catch {
            throw "Download failed: $($_.Exception.Message)"
        }
        
        # Extract files
        Write-Info "Extracting files..."
        Expand-Archive -Path $tempFile -DestinationPath $tempDir -Force
        
        # Check if binary file exists
        $binaryPath = "$tempDir\autocert.exe"
        if (-not (Test-Path $binaryPath)) {
            Stop-OnError "autocert.exe not found after extraction"
        }
        
        # Create installation directory
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        
        # Copy binary file
        $targetPath = "$InstallDir\autocert.exe"
        Copy-Item $binaryPath $targetPath -Force
        
        Write-Info "Binary file installation completed: $targetPath"
        
    } catch {
        Stop-OnError "Download failed: $($_.Exception.Message)"
    } finally {
        # Clean up temporary files
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Create configuration directories and files
function New-Configuration {
    Write-Info "Creating configuration directories and files..."
    
    # Create configuration directories
    $certDir = "$ConfigDir\certs"
    $logDir = "$ConfigDir\logs"
    
    @($ConfigDir, $certDir, $logDir) | ForEach-Object {
        if (-not (Test-Path $_)) {
            New-Item -ItemType Directory -Path $_ -Force | Out-Null
            Write-Debug "Created directory: $_"
        }
    }
    
    # Create default configuration file
    $configFile = "$ConfigDir\config.yaml"
    if (-not (Test-Path $configFile) -or $Force) {
        $configContent = @"
# AutoCert Configuration File
log_level: info
config_dir: $ConfigDir
cert_dir: $certDir
log_dir: $logDir

# ACME Configuration
acme:
  server: https://acme-v02.api.letsencrypt.org/directory
  key_type: rsa
  key_size: 2048

# Web Server Configuration
webserver:
  type: iis  # iis, nginx
  reload_cmd: iisreset

# Notification Configuration
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
        Write-Info "Default configuration file created: $configFile"
    } else {
        Write-Info "Configuration file already exists, skipping creation"
    }
}

# Detect Web Server
function Test-WebServer {
    Write-Info "Detecting web server..."
    
    # Detect IIS
    $iisFeature = Get-WindowsFeature -Name IIS-WebServer -ErrorAction SilentlyContinue
    if ($iisFeature -and $iisFeature.InstallState -eq "Installed") {
        Write-Info "Detected IIS"
        return
    }
    
    # Detect Nginx for Windows
    $nginxPaths = @(
        "C:\nginx\nginx.exe",
        "C:\Program Files\nginx\nginx.exe",
        "$env:ProgramFiles\nginx\nginx.exe"
    )
    
    foreach ($path in $nginxPaths) {
        if (Test-Path $path) {
            Write-Info "Detected Nginx: $path"
            
            # Update configuration file
            $configFile = "$ConfigDir\config.yaml"
            if (Test-Path $configFile) {
                $content = Get-Content $configFile -Raw
                $content = $content -replace "type: iis", "type: nginx"
                Set-Content -Path $configFile -Value $content -Encoding UTF8
                Write-Info "Configuration file updated with web server type: nginx"
            }
            return
        }
    }
    
    Write-Warn "No supported web server detected (IIS/Nginx)"
}

# Add to PATH environment variable
function Add-ToPath {
    Write-Info "Adding to PATH environment variable..."
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
    
    if ($currentPath -notlike "*$InstallDir*") {
        $newPath = "$currentPath;$InstallDir"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
        Write-Info "Added to system PATH: $InstallDir"
        
        # Update current session PATH immediately
        $env:PATH += ";$InstallDir"
        
        # Also update current user PATH as backup
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath -notlike "*$InstallDir*") {
            $newUserPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
            [Environment]::SetEnvironmentVariable("PATH", $newUserPath, "User")
            Write-Debug "Also added to user PATH as backup"
        }
    } else {
        Write-Info "Installation directory already in PATH"
        # Ensure current session has the path
        if ($env:PATH -notlike "*$InstallDir*") {
            $env:PATH += ";$InstallDir"
        }
    }
}

# Create firewall rules
function New-FirewallRules {
    Write-Info "Creating firewall rules..."
    
    try {
        # Allow HTTP (80) and HTTPS (443) ports
        $rules = @(
            @{Name="AutoCert-HTTP"; Port=80; Protocol="TCP"},
            @{Name="AutoCert-HTTPS"; Port=443; Protocol="TCP"}
        )
        
        foreach ($rule in $rules) {
            $existingRule = Get-NetFirewallRule -DisplayName $rule.Name -ErrorAction SilentlyContinue
            if (-not $existingRule) {
                New-NetFirewallRule -DisplayName $rule.Name -Direction Inbound -Protocol $rule.Protocol -LocalPort $rule.Port -Action Allow | Out-Null
                Write-Debug "Created firewall rule: $($rule.Name)"
            }
        }
        
        Write-Info "Firewall rules configuration completed"
    } catch {
        Write-Warn "Firewall rule creation failed: $($_.Exception.Message)"
    }
}

# Verify installation
function Test-Installation {
    Write-Info "Verifying installation..."
    
    $binaryPath = "$InstallDir\autocert.exe"
    
    if (-not (Test-Path $binaryPath)) {
        Stop-OnError "Installation verification failed: binary file does not exist"
    }
    
    try {
        # Test using full path
        $output = & $binaryPath --version 2>&1
        Write-Debug "Command output (full path): $output"
        
        # Test using command name from PATH
        try {
            $pathOutput = & autocert --version 2>&1
            Write-Debug "Command output (from PATH): $pathOutput"
            Write-Info "Installation verification successful - command available in PATH"
        } catch {
            Write-Warn "Command not immediately available in PATH, but binary exists. May require new session."
        }
        
        Write-Info "Installation verification successful"
    } catch {
        Stop-OnError "Installation verification failed: command execution failed - $($_.Exception.Message)"
    }
}

# Display post-installation information
function Show-PostInstallInfo {
    Write-Host ""
    Write-Host "🎉 AutoCert Installation Successful!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Installation Information:" -ForegroundColor Cyan
    Write-Host "  - Binary file: $InstallDir\autocert.exe"
    Write-Host "  - Configuration directory: $ConfigDir"
    Write-Host "  - Configuration file: $ConfigDir\config.yaml"
    Write-Host "  - Certificate directory: $ConfigDir\certs"
    Write-Host "  - Log directory: $ConfigDir\logs"
    Write-Host ""
    
    # Test if command is available immediately
    try {
        $testOutput = & autocert --version 2>&1
        Write-Host "✅ AutoCert command is ready to use!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Quick Start (available now):" -ForegroundColor Cyan
        Write-Host "  1. Configure email and domain:"
        Write-Host "     autocert install --domain your-domain.com --email your-email@example.com --iis"
        Write-Host ""
        Write-Host "  2. Setup automatic renewal:"
        Write-Host "     autocert schedule install"
        Write-Host ""
        Write-Host "  3. Check certificate status:"
        Write-Host "     autocert status"
        Write-Host ""
        Write-Host "  4. View help:"
        Write-Host "     autocert --help"
    } catch {
        Write-Host "⚠️  AutoCert command not immediately available" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "If 'autocert' command is not found, try:" -ForegroundColor Cyan
        Write-Host "  Option 1: Close and reopen PowerShell/Command Prompt"
        Write-Host "  Option 2: Use full path: $InstallDir\autocert.exe"
        Write-Host "  Option 3: Refresh environment in current session:"
        Write-Host "           `$env:PATH += ';$InstallDir'"
        Write-Host ""
        Write-Host "Quick Start (after fixing PATH):" -ForegroundColor Cyan
        Write-Host "  1. Configure email and domain:"
        Write-Host "     autocert install --domain your-domain.com --email your-email@example.com --iis"
        Write-Host "     OR: $InstallDir\autocert.exe install --domain your-domain.com --email your-email@example.com --iis"
        Write-Host ""
        Write-Host "  2. Setup automatic renewal:"
        Write-Host "     autocert schedule install"
        Write-Host ""
        Write-Host "  3. Check certificate status:"
        Write-Host "     autocert status"
        Write-Host ""
        Write-Host "  4. View help:"
        Write-Host "     autocert --help"
    }
    
    Write-Host ""
    Write-Host "Important Notes:" -ForegroundColor Yellow
    Write-Host "  - If command not found, close and reopen your terminal"
    Write-Host "  - Ensure firewall allows inbound connections on ports 80 and 443"
    Write-Host "  - Configuration file location: $ConfigDir\config.yaml"
    Write-Host ""
    Write-Host "For more information visit: https://github.com/renky1025/autocert" -ForegroundColor Blue
}

# Main function
function main {
    Write-Info "Starting AutoCert installation..."
    
    try {
        # Check permissions
        Test-AdminRights
        
        # Detect system
        Get-SystemInfo
        
        # Install dependencies
        Install-Dependencies
        
        # Download and install
        Get-AutoCertBinary
        
        # Create configuration
        New-Configuration
        
        # Detect web server
        Test-WebServer
        
        # Add to PATH
        Add-ToPath
        
        # Configure firewall
        New-FirewallRules
        
        # Verify installation
        Test-Installation
        
        # Display post-installation info
        Show-PostInstallInfo
        
        Write-Info "AutoCert installation completed!"
        
    } catch {
        Write-Error "Installation failed: $($_.Exception.Message)"
        exit 1
    }
}

# Script entry point
if ($MyInvocation.InvocationName -ne '.') {
    main
}