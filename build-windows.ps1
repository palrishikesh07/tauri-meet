$ErrorActionPreference = "Stop"

Write-Host "========================================="
Write-Host " Meet Capture - Windows Build"
Write-Host "========================================="

Write-Host ""
Write-Host "Checking dependencies..."

node --version
npm --version
go version
rustc --version
cargo --version
ffmpeg -version

Write-Host ""
Write-Host "Installing npm dependencies..."

npm install

Write-Host ""
Write-Host "Building Windows Go engine..."

npm run engine:windows

if (-not (Test-Path "engine\meetrec.exe")) {
    throw "engine\meetrec.exe was not created"
}

Write-Host ""
Write-Host "Checking Whisper..."

if (-not (Test-Path "engine\bin\whisper-server.exe")) {
    Write-Warning "engine\bin\whisper-server.exe was not found."
    Write-Warning "Whisper must be installed before recording can work."
}

if (-not (Test-Path "engine\models\ggml-small.en.bin")) {
    throw "engine\models\ggml-small.en.bin was not found"
}

Write-Host ""
Write-Host "Testing Windows engine..."

Push-Location engine

try {
    .\meetrec.exe devices
}
finally {
    Pop-Location
}

Write-Host ""
Write-Host "Building frontend..."

npm run build

Write-Host ""
Write-Host "Building Tauri Windows application..."

npm run tauri build `
    -- `
    --target x86_64-pc-windows-msvc `
    --config src-tauri/tauri.windows.conf.json

Write-Host ""
Write-Host "========================================="
Write-Host " Build completed"
Write-Host "========================================="

$bundlePath = "src-tauri\target\x86_64-pc-windows-msvc\release\bundle"

if (Test-Path "$bundlePath\msi") {
    Write-Host ""
    Write-Host "MSI:"
    Get-ChildItem "$bundlePath\msi"
}

if (Test-Path "$bundlePath\nsis") {
    Write-Host ""
    Write-Host "NSIS:"
    Get-ChildItem "$bundlePath\nsis"
}