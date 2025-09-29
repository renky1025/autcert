@echo off
:: AutoCert Windows Installation Launcher
:: Automatically handles encoding issues

echo ============================================
echo AutoCert Automatic Installation Script
echo ============================================
echo.

:: Set console encoding to UTF-8
chcp 65001 >nul 2>&1

:: Check for administrator privileges
net session >nul 2>&1
if %errorLevel% == 0 (
    echo Administrator privileges detected, continuing installation...
    echo.
) else (
    echo Error: Administrator privileges required to run this script
    echo Please right-click this file and select "Run as administrator"
    pause
    exit /b 1
)

:: Run PowerShell installation script
echo Starting PowerShell installation script...
echo.
powershell -ExecutionPolicy Bypass -File "%~dp0install.ps1"

:: Check installation result
if %errorLevel% == 0 (
    echo.
    echo ============================================
    echo AutoCert installation completed!
    echo ============================================
) else (
    echo.
    echo ============================================
    echo An error occurred during installation, please check the error messages above
    echo ============================================
)

echo.
echo Press any key to close this window...
pause >nul