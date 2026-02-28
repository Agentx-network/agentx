# AgentX One-Click Installer for Windows
# Usage: irm https://raw.githubusercontent.com/Agentx-network/agentx/main/install.ps1 | iex
$ErrorActionPreference = "Stop"

$Repo = "Agentx-network/agentx"
$Binary = "agentx.exe"
$InstallDir = "$env:LOCALAPPDATA\agentx"
# Release assets are raw binaries: agentx-windows-amd64.exe
$Asset = "agentx-windows-amd64.exe"
$Url = "https://github.com/$Repo/releases/latest/download/$Asset"

Write-Host ""
Write-Host "  AgentX Installer" -ForegroundColor Cyan
Write-Host "  -----------------"
Write-Host ""

# Create install directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download binary directly
Write-Host "[info]  Downloading $Asset..." -ForegroundColor Cyan
$DestPath = Join-Path $InstallDir $Binary

try {
    Invoke-WebRequest -Uri $Url -OutFile $DestPath -UseBasicParsing
} catch {
    Write-Host "[error] Download failed: $_" -ForegroundColor Red
    Write-Host "[error] Check your internet connection and that a release exists." -ForegroundColor Red
    exit 1
}

# Add to PATH if not already present
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    Write-Host "[info]  Adding $InstallDir to user PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "[info]  PATH updated. Restart your terminal if 'agentx' is not found." -ForegroundColor Yellow
}

# Verify
if (Test-Path $DestPath) {
    Write-Host "[ok]    AgentX installed successfully to $InstallDir" -ForegroundColor Green
} else {
    Write-Host "[error] Installation failed." -ForegroundColor Red
    exit 1
}

Write-Host ""

# Run onboard wizard
Write-Host "[info]  Launching setup wizard..." -ForegroundColor Cyan
Write-Host ""
try {
    & $DestPath onboard
} catch {
    Write-Host "[ok]    Run 'agentx onboard' to complete setup." -ForegroundColor Green
}
