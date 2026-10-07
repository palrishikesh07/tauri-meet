# Build & Deploy Commands for Tauri Meet - All Platforms

## Prerequisites for All Platforms

```bash
# Install Node dependencies
npm install

# Install Tauri CLI
npm install -g @tauri-apps/cli

# Install Rust (if not already installed)
# Windows: Download from https://www.rust-lang.org/tools/install
# macOS/Linux: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

---

## 🪟 WINDOWS BUILD & DEPLOY

### Prerequisites
- Windows 10/11
- Visual Studio 2022 with C++ build tools (or install via `rustup`)
- Node.js & npm
- Rust toolchain

### Install Windows Dependencies
```bash
# Install WebView2 Runtime (required for Tauri)
# Download from: https://developer.microsoft.com/en-us/microsoft-edge/webview2/

# Run in PowerShell as Administrator:
npm install -g @tauri-apps/cli
rustup target add x86_64-pc-windows-msvc
```

### Development Build (Windows)
```bash
npm run dev:all
# OR
npm run tauri dev
```

### Production Build (Windows)
```bash
# Build the app
npm run build:all

# Create installer
npm run tauri build -- --target x86_64-pc-windows-msvc
```

### Deploy (Windows)
```bash
# The installer will be at:
# src-tauri/target/x86_64-pc-windows-msvc/release/bundle/msi/Meet\ Capture_0.1.0_x64_en-US.msi

# Or copy the executable to Downloads folder:
Copy-Item -Path "src-tauri/target/x86_64-pc-windows-msvc/release/tauri-app.exe" -Destination "Downloads/windows/"

# Sign the installer (optional but recommended)
# You'll need a code signing certificate for production
```

### Windows Build Targets
```bash
# x64
npm run tauri build -- --target x86_64-pc-windows-msvc

# ARM64
npm run tauri build -- --target aarch64-pc-windows-msvc

# i686 (32-bit)
npm run tauri build -- --target i686-pc-windows-gnu
```

---

## 🍎 MACOS BUILD & DEPLOY

### Prerequisites
- macOS 11.0+
- Xcode Command Line Tools: `xcode-select --install`
- Node.js & npm
- Rust toolchain

### Install macOS Dependencies
```bash
# Install Xcode CLT
xcode-select --install

# Add Apple Silicon target (if using M1/M2/M3)
rustup target add aarch64-apple-darwin

# Add Intel target
rustup target add x86_64-apple-darwin
```

### Development Build (macOS)
```bash
npm run dev:all
# OR
npm run tauri dev
```

### Production Build (macOS) - Intel
```bash
npm run build:all

# Build for x86_64 (Intel)
npm run tauri build -- --target x86_64-apple-darwin
```

### Production Build (macOS) - Apple Silicon
```bash
npm run build:all

# Build for ARM64 (Apple Silicon)
npm run tauri build -- --target aarch64-apple-darwin
```

### Universal Build (macOS - Intel + Apple Silicon)
```bash
npm run build:all

# Build universal binary
npm run tauri build -- --target universal-apple-darwin
```

### Deploy (macOS)
```bash
# The app bundle will be at:
# src-tauri/target/[target]/release/bundle/macos/Meet\ Capture.app

# Create DMG installer
# Already happens automatically during build

# Copy to Downloads folder
cp -r "src-tauri/target/x86_64-apple-darwin/release/bundle/macos/Meet Capture.app" "Downloads/macos/"

# Or copy DMG
cp "src-tauri/target/x86_64-apple-darwin/release/bundle/dmg/Meet Capture_0.1.0_x64.dmg" "Downloads/macos/"

# Code signing (for App Store / production)
codesign --deep --force --verify --verbose --sign "Developer ID Application" "src-tauri/target/x86_64-apple-darwin/release/bundle/macos/Meet Capture.app"

# Notarize (required for distribution)
xcrun altool --notarize-app --file "Meet Capture.dmg" --primary-bundle-id "com.taurimeet.recorder"
```

---

## 🐧 LINUX BUILD & DEPLOY

### Prerequisites
- Linux (Ubuntu 20.04+, Fedora 36+, Debian 11+)
- Node.js & npm
- Rust toolchain
- Build essentials

### Install Linux Dependencies

#### Ubuntu/Debian:
```bash
sudo apt-get update
sudo apt-get install -y \
  build-essential \
  curl \
  wget \
  file \
  libssl-dev \
  libayatana-appindicator3-dev \
  libgtk-3-dev \
  librsvg2-dev \
  xdg-utils
```

#### Fedora/RHEL:
```bash
sudo dnf install -y \
  gcc \
  glibc-devel \
  openssl-devel \
  gtk3-devel \
  librsvg2-devel \
  libappindicator-gtk3-devel
```

#### Arch:
```bash
sudo pacman -S base-devel gtk3 librsvg openssl
```

### Development Build (Linux)
```bash
npm run dev:all
# OR
npm run tauri dev
```

### Production Build (Linux)
```bash
npm run build:all

# Build AppImage (recommended)
npm run tauri build -- --target x86_64-unknown-linux-gnu

# Or build .deb package
npm run tauri build -- --target x86_64-unknown-linux-gnu --bundles deb
```

### Deploy (Linux)
```bash
# AppImage will be at:
# src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/appimage/Meet\ Capture_0.1.0_amd64.AppImage

# Deb package will be at:
# src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/deb/meet-capture_0.1.0_amd64.deb

# Copy to Downloads folder
cp "src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/appimage/"*.AppImage "Downloads/linux/"
cp "src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/deb/"*.deb "Downloads/linux/"

# Make AppImage executable
chmod +x "Downloads/linux/"*.AppImage

# For .deb installation
sudo dpkg -i "Downloads/linux/"*.deb
# Or
sudo apt install "Downloads/linux/"*.deb
```

### Linux Build Targets
```bash
# x86_64
npm run tauri build -- --target x86_64-unknown-linux-gnu

# ARM64 (for Raspberry Pi, ARM servers)
npm run tauri build -- --target aarch64-unknown-linux-gnu

# ARMv7 (32-bit ARM)
npm run tauri build -- --target armv7-unknown-linux-gnueabihf
```

---

## 📦 QUICK BUILD & DEPLOY SCRIPTS

### All-in-One Build Script (Windows PowerShell)
```powershell
# build-all.ps1
$platforms = @("x86_64-pc-windows-msvc", "x86_64-apple-darwin", "x86_64-unknown-linux-gnu")
foreach ($platform in $platforms) {
    Write-Host "Building for $platform..."
    npm run build:all
    npm run tauri build -- --target $platform
}
```

### All-in-One Build Script (macOS/Linux Bash)
```bash
#!/bin/bash
# build-all.sh

platforms=("x86_64-apple-darwin" "aarch64-apple-darwin" "x86_64-unknown-linux-gnu")

for platform in "${platforms[@]}"; do
    echo "Building for $platform..."
    npm run build:all
    npm run tauri build -- --target "$platform"
done
```

---

## 🚀 CONTINUOUS DEPLOYMENT OPTIONS

### GitHub Actions (Windows + macOS + Linux)
Create `.github/workflows/release.yml`:
```yaml
name: Build and Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    strategy:
      matrix:
        platform: [ubuntu-latest, macos-latest, windows-latest]
    
    runs-on: ${{ matrix.platform }}
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - uses: dtolnay/rust-toolchain@stable
      
      - name: Install dependencies
        run: npm install
      
      - name: Build
        run: npm run build:all
      
      - name: Build Tauri
        run: npm run tauri build
```

---

## 📋 FINAL DEPLOYMENT CHECKLIST

- [ ] Update version in `package.json`
- [ ] Update version in `src-tauri/Cargo.toml`
- [ ] Update version in `src-tauri/tauri.conf.json`
- [ ] Build for all target platforms
- [ ] Test the built packages on each OS
- [ ] Copy artifacts to `Downloads/[os]/` folder
- [ ] Create release notes
- [ ] Upload to hosting/distribution platform
- [ ] Test installation on fresh systems

---

## 📁 Expected Output Structure

```
Downloads/
├── windows/
│   ├── Meet Capture_0.1.0_x64_en-US.msi
│   └── tauri-app.exe
├── macos/
│   ├── Meet Capture_0.1.0_x64.dmg
│   ├── Meet Capture.app/
│   └── universal/
├── linux/
│   ├── Meet Capture_0.1.0_amd64.AppImage
│   └── meet-capture_0.1.0_amd64.deb
```

---

## 🔧 Common Issues & Solutions

### Windows: WebView2 Missing
```bash
# Install WebView2 Runtime
https://developer.microsoft.com/en-us/microsoft-edge/webview2/
```

### macOS: Code Signing Issues
```bash
# Remove code signature
codesign --remove-signature "path/to/binary"

# Or sign with developer certificate
codesign -s - "path/to/binary"
```

### Linux: GTK Not Found
```bash
# Ubuntu/Debian
sudo apt-get install libgtk-3-dev librsvg2-dev

# Fedora
sudo dnf install gtk3-devel librsvg2-devel
```

---

## 📞 Quick Reference

| Task | Command |
|------|---------|
| Dev on current platform | `npm run dev:all` |
| Build for release | `npm run build:all` |
| Build package | `npm run tauri build` |
| Build specific target | `npm run tauri build -- --target [target]` |
| Preview | `npm run preview` |
| Clean build | `rm -rf src-tauri/target && npm run build:all` |
