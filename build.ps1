# build.ps1 - Script to build Wails App (EXE, NSIS Installer, and ZIP)

Write-Host "Checking if Wails CLI is installed..."
if (!(Get-Command wails -ErrorAction SilentlyContinue)) {
    Write-Host "Wails CLI not found. Installing..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
}

Write-Host "Building Wails App (including NSIS Installer)..."
wails build -nsis

if ($LASTEXITCODE -ne 0) {
    Write-Error "Wails build failed."
    exit $LASTEXITCODE
}

Write-Host "Zipping the executable..."
$originalExe = "build\bin\Lymuru.exe"
$originalInstaller = "build\bin\Lymuru-amd64-installer.exe"
$portableExe = "build\bin\Lymuru-Portable.exe"
$installerExe = "build\bin\Lymuru-Installer.exe"
$zipPath = "build\bin\Lymuru-Windows.zip"

if (Test-Path $portableExe) { Remove-Item $portableExe }
if (Test-Path $installerExe) { Remove-Item $installerExe }
if (Test-Path $zipPath) { Remove-Item $zipPath }

if (Test-Path $originalExe) {
    Rename-Item -Path $originalExe -NewName "Lymuru-Portable.exe"
    Compress-Archive -Path $portableExe -DestinationPath $zipPath
    Write-Host "Successfully created $portableExe and $zipPath"
} else {
    Write-Error "Executable not found at $originalExe"
}

if (Test-Path $originalInstaller) {
    Rename-Item -Path $originalInstaller -NewName "Lymuru-Installer.exe"
    Write-Host "Successfully created $installerExe"
} else {
    Write-Host "Installer not found at $originalInstaller (skip if nsis not installed)"
}

Write-Host "Build complete! Check the build\bin folder for outputs."
