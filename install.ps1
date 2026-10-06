# ClaudeSwap Universal Installer for Windows PowerShell
$ErrorActionPreference = "Stop"

$Repo = "claudeswap/claudeswap"
$Version = "1.0.0"

Write-Host "ClaudeSwap Windows Installer" -ForegroundColor Cyan
Write-Host "============================"

$InstallDir = Join-Path $HOME ".claudeswap\bin"
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "arm64" }
$BinName = "claudeswap-$Version-windows-$Arch.exe"
$TargetExe = Join-Path $InstallDir "claudeswap.exe"

# If Go is installed, build locally
if (Get-Command "go" -ErrorAction SilentlyContinue) {
    Write-Host "Go detected. Installing via 'go install'..." -ForegroundColor Green
    $env:GOBIN = $InstallDir
    go install github.com/claudeswap/claudeswap/cmd/claudeswap@latest
}

if (!(Test-Path $TargetExe)) {
    $DownloadUrl = "https://github.com/$Repo/releases/download/v$Version/$BinName"
    Write-Host "Downloading $BinName from GitHub Releases..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TargetExe
}

Write-Host "Adding $InstallDir to user PATH..." -ForegroundColor Green
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$UserPath", "User")
}

Write-Host "`n✓ ClaudeSwap installed successfully!" -ForegroundColor Green
Write-Host "Please restart your PowerShell window, then run:"
Write-Host "  claudeswap add <ProfileName>" -ForegroundColor Cyan
Write-Host "  claudeswap" -ForegroundColor Cyan
Write-Host "  claudeswap switch" -ForegroundColor Cyan
