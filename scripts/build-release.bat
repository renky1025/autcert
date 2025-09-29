@echo off
REM AutoCert 跨平台一键打包快捷脚本 - Windows 批处理版本

setlocal enabledelayedexpansion

REM 获取参数
set "VERSION=%~1"
set "PLATFORM=%~2"

REM 如果没有指定版本，尝试从 git 获取
if "%VERSION%"=="" (
    for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%i"
    if "!VERSION!"=="" set "VERSION=dev"
)

REM 如果没有指定平台，默认为 all
if "%PLATFORM%"=="" set "PLATFORM=all"

echo 🚀 AutoCert 跨平台一键打包
echo 版本: %VERSION%
echo 平台: %PLATFORM%
echo.

REM 切换到脚本所在目录的上级目录（项目根目录）
cd /d "%~dp0.."

REM 执行 PowerShell 打包脚本
echo 使用 Windows PowerShell 打包脚本...
powershell -ExecutionPolicy Bypass -File "scripts\package.ps1" -Version "%VERSION%" -Platform "%PLATFORM%"

echo.
echo 📁 打包结果:
if exist "dist" (
    dir /b "dist\*.zip" "dist\*.tar.gz" 2>nul
) else (
    echo 打包目录不存在
)

pause