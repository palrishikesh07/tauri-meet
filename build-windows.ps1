# build-windows.ps1 - Build Meet Capture for Windows (PowerShell)

Write-Host "🔨 Building Meet Capture for Windows (x86_64)..." -ForegroundColor Green
Write-Host ""

# Check if running on Windows
if ($PSVersionTable.Platform -ne "Win32NT") {
    Write-Host "⚠️  WARNING: You are not on Windows." -ForegroundColor Yellow
    Write-Host "   To build Windows on macOS/Linux, use: npm run tauri build -- --target x86_64-pc-windows-msvc" -ForegroundColor Yellow
    exit 1
}

# Check WebView2
Write-Host "🔍 Checking for WebView2 Runtime..."
if (-not (Test-Path "HKLM:\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}")) {
    Write-Host "⚠️  WebView2 Runtime not found. Installing..." -ForegroundColor Yellow
    Write-Host "   Download from: https://developer.microsoft.com/en-us/microsoft-edge/webview2/" -ForegroundColor Cyan
}

# Install dependencies
Write-Host "📦 Installing Node dependencies..." -ForegroundColor Cyan
npm install

# Build frontend and backend
Write-Host "🏗️  Building frontend and backend..." -ForegroundColor Cyan
npm run build:all

# Build Tauri app for Windows
Write-Host "📦 Building Tauri application..." -ForegroundColor Cyan
npm run tauri build -- --target x86_64-pc-windows-msvc

# Create output directory
New-Item -ItemType Directory -Path "Downloads\windows" -Force | Out-Null

# Copy artifacts
Write-Host "📁 Copying artifacts to Downloads\windows..." -ForegroundColor Cyan

# Copy MSI installer
$msiPath = "src-tauri\target\x86_64-pc-windows-msvc\release\bundle\msi"
if (Test-Path $msiPath) {
    Get-ChildItem "$msiPath\*.msi" | ForEach-Object {
        Copy-Item $_.FullName "Downloads\windows\"
    }
}

# Copy executable
$exePath = "src-tauri\target\x86_64-pc-windows-msvc\release\tauri-app.exe"
if (Test-Path $exePath) {
    Copy-Item $exePath "Downloads\windows\"
}

Write-Host ""
Write-Host "✅ Build complete!" -ForegroundColor Green
Write-Host "📦 Artifacts saved to: Downloads\windows\" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Files created:" -ForegroundColor Cyan
Get-ChildItem "Downloads\windows\" | Select-Object Name, Length

Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Test the installer or executable" -ForegroundColor Yellow
Write-Host "  2. Sign the MSI for production (requires code signing certificate)" -ForegroundColor Yellow
Write-Host "  3. Distribute to users" -ForegroundColor Yellow
