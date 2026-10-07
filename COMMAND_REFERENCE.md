# ⚡ Build & Deploy - Command Reference Card

## Quick Start (Any OS)
```bash
npm install           # Install dependencies
npm run dev:all       # Start development
```

---

## 🪟 WINDOWS - Build & Deploy

### Option 1: Recommended (Build Script)
```powershell
.\build-windows.ps1
```

### Option 2: Manual Commands
```bash
npm run build:all
npm run tauri build -- --target x86_64-pc-windows-msvc
Copy-Item -Path "src-tauri/target/x86_64-pc-windows-msvc/release/bundle/msi/*" -Destination "Downloads/windows/"
```

### Setup Prerequisites
```bash
# Install WebView2: https://developer.microsoft.com/en-us/microsoft-edge/webview2/
rustup target add x86_64-pc-windows-msvc
npm install
```

### Output Files
- **MSI Installer:** `Downloads/windows/Meet Capture_0.1.0_x64_en-US.msi`
- **Executable:** `Downloads/windows/tauri-app.exe`

---

## 🍎 MACOS - Build & Deploy

### Option 1: Recommended (Build Script)
```bash
./build-macos.sh
```

### Option 2: Manual Commands (Intel)
```bash
npm run build:all
npm run tauri build -- --target x86_64-apple-darwin
cp -r "src-tauri/target/x86_64-apple-darwin/release/bundle/macos/Meet Capture.app" "Downloads/macos/"
cp "src-tauri/target/x86_64-apple-darwin/release/bundle/dmg/"*.dmg "Downloads/macos/"
```

### Option 2b: Manual Commands (Apple Silicon)
```bash
npm run build:all
npm run tauri build -- --target aarch64-apple-darwin
cp -r "src-tauri/target/aarch64-apple-darwin/release/bundle/macos/Meet Capture.app" "Downloads/macos/"
```

### Option 2c: Manual Commands (Universal)
```bash
npm run build:all
npm run tauri build -- --target universal-apple-darwin
cp -r "src-tauri/target/universal-apple-darwin/release/bundle/macos/Meet Capture.app" "Downloads/macos/"
```

### Setup Prerequisites
```bash
xcode-select --install
rustup target add x86_64-apple-darwin aarch64-apple-darwin
npm install
```

### Output Files
- **App Bundle:** `Downloads/macos/Meet Capture.app`
- **DMG Installer:** `Downloads/macos/Meet Capture_0.1.0_x64.dmg`

### Code Signing (Production)
```bash
codesign --deep --force --verify --verbose --sign "Developer ID Application" "Downloads/macos/Meet Capture.app"
```

---

## 🐧 LINUX - Build & Deploy

### Option 1: Recommended (Build Script)
```bash
./build-linux.sh
```

### Option 2: Manual Commands (AppImage)
```bash
npm run build:all
npm run tauri build -- --target x86_64-unknown-linux-gnu
cp "src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/appimage/"*.AppImage "Downloads/linux/"
chmod +x "Downloads/linux/"*.AppImage
```

### Option 2b: Manual Commands (.deb)
```bash
npm run build:all
npm run tauri build -- --target x86_64-unknown-linux-gnu --bundles deb
cp "src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/deb/"*.deb "Downloads/linux/"
```

### Setup Prerequisites (Ubuntu/Debian)
```bash
sudo apt-get update && sudo apt-get install -y \
  build-essential curl wget file libssl-dev \
  libayatana-appindicator3-dev libgtk-3-dev librsvg2-dev xdg-utils
npm install
```

### Setup Prerequisites (Fedora/RHEL)
```bash
sudo dnf install -y gcc glibc-devel openssl-devel \
  gtk3-devel librsvg2-devel libappindicator-gtk3-devel
npm install
```

### Setup Prerequisites (Arch)
```bash
sudo pacman -S --noconfirm base-devel gtk3 librsvg openssl
npm install
```

### Output Files
- **AppImage:** `Downloads/linux/Meet Capture_0.1.0_amd64.AppImage` (Recommended)
- **.deb Package:** `Downloads/linux/meet-capture_0.1.0_amd64.deb`

### Install .deb Package
```bash
sudo apt install ./Downloads/linux/*.deb
```

### Run AppImage
```bash
./Downloads/linux/*.AppImage
```

---

## 🔄 Build All Platforms (CI/CD)

### Sequential (Slower but simpler)
```bash
npm run build:all
npm run tauri build -- --target x86_64-pc-windows-msvc
npm run tauri build -- --target x86_64-apple-darwin
npm run tauri build -- --target x86_64-unknown-linux-gnu
```

### Parallel (Faster)
```bash
npm run build:all
npm run tauri build -- --target x86_64-pc-windows-msvc & \
npm run tauri build -- --target x86_64-apple-darwin & \
npm run tauri build -- --target x86_64-unknown-linux-gnu &
wait
```

---

## 📁 New Files & Folders Created

### Documentation Files
- ✅ **BUILD_DEPLOY_GUIDE.md** - Comprehensive guide (30+ KB)
- ✅ **QUICK_COMMANDS.md** - Quick reference guide
- ✅ **SETUP_SUMMARY.md** - Setup overview
- ✅ **COMMAND_REFERENCE.md** - This file

### Build Scripts (Auto-generated)
- ✅ **build-windows.ps1** - Interactive Windows build script
- ✅ **build-windows.sh** - Windows Bash script
- ✅ **build-macos.sh** - Interactive macOS build script
- ✅ **build-linux.sh** - Interactive Linux build script

### Assets
- ✅ **src-tauri/icons/settings-icon.svg** - Settings gear icon (SVG)

### Folders
- ✅ **Downloads/** - Root folder for build artifacts
  - ✅ **Downloads/windows/** - Windows builds
  - ✅ **Downloads/macos/** - macOS builds
  - ✅ **Downloads/linux/** - Linux builds

---

## 🎯 Build Targets Supported

### Windows
| Target | Command |
|--------|---------|
| x86_64 (64-bit) | `--target x86_64-pc-windows-msvc` |

### macOS
| Target | Command |
|--------|---------|
| Intel (x86_64) | `--target x86_64-apple-darwin` |
| Apple Silicon (ARM64) | `--target aarch64-apple-darwin` |
| Universal (both) | `--target universal-apple-darwin` ⭐ |

### Linux
| Target | Command |
|--------|---------|
| x86_64 (64-bit) | `--target x86_64-unknown-linux-gnu` |
| ARM64 | `--target aarch64-unknown-linux-gnu` |
| ARMv7 (32-bit) | `--target armv7-unknown-linux-gnueabihf` |

---

## ✨ Settings Icon (SVG)

**Location:** `src-tauri/icons/settings-icon.svg`

Features:
- Professional gear/settings icon design
- Blue color (#3b82f6)
- Scalable vector format
- Ready to use as app icon or UI element

---

## 📋 Useful npm Scripts

```bash
npm run engine         # Build Go engine
npm run dev           # Start frontend dev server
npm run dev:all       # Start frontend + Go engine
npm run build         # Build frontend only
npm run build:all     # Build frontend + Go engine
npm run preview       # Preview built app
npm run tauri dev     # Tauri dev mode
npm run tauri build   # Tauri production build
```

---

## 🆘 Common Issues & Fixes

### WebView2 Missing (Windows)
```
Download: https://developer.microsoft.com/en-us/microsoft-edge/webview2/
```

### GTK Not Found (Linux)
```bash
# Ubuntu/Debian
sudo apt-get install libgtk-3-dev librsvg2-dev

# Fedora
sudo dnf install gtk3-devel librsvg2-devel
```

### Xcode CLT Missing (macOS)
```bash
xcode-select --install
```

### Rust Target Not Found
```bash
rustup update
rustup target add [target-name]
```

### Build Cache Issues
```bash
rm -rf src-tauri/target
npm install
npm run build:all
```

---

## 📖 Complete Documentation

For detailed information, see:
- **BUILD_DEPLOY_GUIDE.md** - Complete guide with troubleshooting
- **QUICK_COMMANDS.md** - Extended reference with all commands
- **SETUP_SUMMARY.md** - Setup overview and next steps

---

## ✅ Release Checklist

- [ ] Update version in `package.json`
- [ ] Update version in `src-tauri/Cargo.toml`
- [ ] Update version in `src-tauri/tauri.conf.json`
- [ ] Run `npm run build:all`
- [ ] Build for all 3 platforms
- [ ] Test on each platform
- [ ] Copy to `Downloads/[platform]/`
- [ ] Create release notes
- [ ] Test installation
- [ ] Deploy/distribute

---

**Last Updated:** October 7, 2026  
**Version:** 0.1.0  
**Status:** ✅ Ready for Development & Deployment
