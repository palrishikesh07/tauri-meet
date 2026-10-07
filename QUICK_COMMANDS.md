# 🚀 Quick Build & Deploy Commands

## One-Line Quick Start Commands

### Development
```bash
# Start development server with backend
npm run dev:all
```

### Production Builds

#### Windows (PowerShell)
```powershell
# Interactive build script (recommended)
.\build-windows.ps1

# Or direct Tauri build
npm run build:all && npm run tauri build -- --target x86_64-pc-windows-msvc
```

#### macOS
```bash
# Interactive build script (recommended)
./build-macos.sh

# Or direct Tauri build for Intel
npm run build:all && npm run tauri build -- --target x86_64-apple-darwin

# For Apple Silicon
npm run build:all && npm run tauri build -- --target aarch64-apple-darwin

# Universal binary (both architectures)
npm run build:all && npm run tauri build -- --target universal-apple-darwin
```

#### Linux
```bash
# Interactive build script (recommended)
./build-linux.sh

# Or direct Tauri build (AppImage)
npm run build:all && npm run tauri build -- --target x86_64-unknown-linux-gnu

# Or .deb package
npm run build:all && npm run tauri build -- --target x86_64-unknown-linux-gnu --bundles deb
```

---

## Complete Command Reference for All 3 OS

### 🪟 Windows Commands

#### Setup Prerequisites
```bash
# Install WebView2 (Download and run installer from)
https://developer.microsoft.com/en-us/microsoft-edge/webview2/

# Install Rust target
rustup target add x86_64-pc-windows-msvc

# Install npm dependencies
npm install
```

#### Build & Deploy
```bash
# Development
npm run dev:all

# Production Build
npm run build:all
npm run tauri build -- --target x86_64-pc-windows-msvc

# Copy to Downloads
Copy-Item -Path "src-tauri/target/x86_64-pc-windows-msvc/release/bundle/msi/*" -Destination "Downloads/windows/"

# Alternative: Run build script
.\build-windows.ps1
```

#### Build Artifacts Location
```
src-tauri/target/x86_64-pc-windows-msvc/release/bundle/
├── msi/                          # MSI Installer
│   └── Meet Capture_0.1.0_x64_en-US.msi
└── nsis/                         # NSIS Installer (alternative)
    └── Meet Capture_0.1.0_x64-setup.exe
```

---

### 🍎 macOS Commands

#### Setup Prerequisites
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Add Rust targets
rustup target add x86_64-apple-darwin       # Intel
rustup target add aarch64-apple-darwin      # Apple Silicon
rustup target add universal-apple-darwin    # Both

# Install npm dependencies
npm install
```

#### Build & Deploy (Intel)
```bash
# Development
npm run dev:all

# Production Build
npm run build:all
npm run tauri build -- --target x86_64-apple-darwin

# Copy to Downloads
cp -r "src-tauri/target/x86_64-apple-darwin/release/bundle/macos/Meet Capture.app" "Downloads/macos/"
cp "src-tauri/target/x86_64-apple-darwin/release/bundle/dmg/"*.dmg "Downloads/macos/"
```

#### Build & Deploy (Apple Silicon)
```bash
npm run build:all
npm run tauri build -- --target aarch64-apple-darwin

cp -r "src-tauri/target/aarch64-apple-darwin/release/bundle/macos/Meet Capture.app" "Downloads/macos/"
cp "src-tauri/target/aarch64-apple-darwin/release/bundle/dmg/"*.dmg "Downloads/macos/"
```

#### Build & Deploy (Universal - Recommended)
```bash
npm run build:all
npm run tauri build -- --target universal-apple-darwin

cp -r "src-tauri/target/universal-apple-darwin/release/bundle/macos/Meet Capture.app" "Downloads/macos/Meet Capture-universal.app"
cp "src-tauri/target/universal-apple-darwin/release/bundle/dmg/"*.dmg "Downloads/macos/Meet Capture-universal.dmg"
```

#### Code Signing (Production)
```bash
# Sign the app bundle
codesign --deep --force --verify --verbose --sign "Developer ID Application" "src-tauri/target/x86_64-apple-darwin/release/bundle/macos/Meet Capture.app"

# Notarize (Mac App Store requirement)
xcrun altool --notarize-app --file "Meet Capture.dmg" --primary-bundle-id "com.taurimeet.recorder"

# Alternative: Run build script
./build-macos.sh
```

#### Build Artifacts Location
```
src-tauri/target/[target]/release/bundle/
├── macos/
│   └── Meet Capture.app                    # App bundle
└── dmg/
    └── Meet Capture_0.1.0_x64.dmg         # DMG installer
```

---

### 🐧 Linux Commands

#### Setup Prerequisites

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y \
  build-essential \
  curl wget file \
  libssl-dev \
  libayatana-appindicator3-dev \
  libgtk-3-dev \
  librsvg2-dev \
  xdg-utils

npm install
```

**Fedora/RHEL:**
```bash
sudo dnf install -y \
  gcc glibc-devel openssl-devel \
  gtk3-devel librsvg2-devel \
  libappindicator-gtk3-devel

npm install
```

**Arch:**
```bash
sudo pacman -S --noconfirm base-devel gtk3 librsvg openssl
npm install
```

#### Build & Deploy (AppImage - Recommended)
```bash
# Development
npm run dev:all

# Production Build
npm run build:all
npm run tauri build -- --target x86_64-unknown-linux-gnu

# Copy to Downloads and make executable
cp "src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/appimage/"*.AppImage "Downloads/linux/"
chmod +x "Downloads/linux/"*.AppImage

# Run the AppImage
./Downloads/linux/*.AppImage
```

#### Build & Deploy (.deb Package)
```bash
npm run build:all
npm run tauri build -- --target x86_64-unknown-linux-gnu --bundles deb

cp "src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/deb/"*.deb "Downloads/linux/"

# Install
sudo apt install ./Downloads/linux/*.deb
```

#### Alternative: Run build script
```bash
./build-linux.sh
```

#### Build Artifacts Location
```
src-tauri/target/x86_64-unknown-linux-gnu/release/bundle/
├── appimage/
│   └── Meet Capture_0.1.0_amd64.AppImage   # AppImage bundle
└── deb/
    └── meet-capture_0.1.0_amd64.deb        # Debian package
```

---

## 📋 File Locations Summary

### Icon Assets
```
src-tauri/icons/
├── settings-icon.svg          # ✅ NEW Settings icon (SVG)
├── icon.png                   # Main icon
├── icon.icns                  # macOS icon
├── icon.ico                   # Windows icon
├── 32x32.png
├── 128x128.png
└── ...other icon formats
```

### Build Output
```
Downloads/
├── windows/                   # Windows installers & executables
├── macos/                     # macOS app bundles & DMG installers
└── linux/                     # AppImage & .deb packages
```

### Build Cache
```
src-tauri/target/
├── x86_64-pc-windows-msvc/    # Windows build artifacts
├── x86_64-apple-darwin/       # macOS Intel build artifacts
├── aarch64-apple-darwin/      # macOS Apple Silicon artifacts
├── universal-apple-darwin/    # macOS Universal artifacts
└── x86_64-unknown-linux-gnu/  # Linux build artifacts
```

---

## 🔄 Build Matrix - All Combinations

### Windows
| Target | Command | Output Path |
|--------|---------|-------------|
| x86_64 | `npm run tauri build -- --target x86_64-pc-windows-msvc` | `target/x86_64-pc-windows-msvc/release/bundle/msi/` |

### macOS
| Target | Command | Output Path |
|--------|---------|-------------|
| Intel (x86_64) | `npm run tauri build -- --target x86_64-apple-darwin` | `target/x86_64-apple-darwin/release/bundle/macos/` |
| Apple Silicon (ARM64) | `npm run tauri build -- --target aarch64-apple-darwin` | `target/aarch64-apple-darwin/release/bundle/macos/` |
| Universal | `npm run tauri build -- --target universal-apple-darwin` | `target/universal-apple-darwin/release/bundle/macos/` |

### Linux
| Target | Command | Output Path |
|--------|---------|-------------|
| x86_64 (AppImage) | `npm run tauri build -- --target x86_64-unknown-linux-gnu` | `target/x86_64-unknown-linux-gnu/release/bundle/appimage/` |
| x86_64 (.deb) | `npm run tauri build -- --target x86_64-unknown-linux-gnu --bundles deb` | `target/x86_64-unknown-linux-gnu/release/bundle/deb/` |
| ARM64 | `npm run tauri build -- --target aarch64-unknown-linux-gnu` | `target/aarch64-unknown-linux-gnu/release/bundle/` |

---

## ⚡ Performance Build Tips

### Clean Build (fresh start)
```bash
# Windows
rm -r src-tauri/target
npm run build:all
npm run tauri build

# macOS/Linux
rm -rf src-tauri/target
npm run build:all
npm run tauri build
```

### Incremental Build (faster)
```bash
# Just rebuild without cleaning
npm run tauri build
```

### Parallel Builds (for CI/CD)
```bash
# Build multiple targets simultaneously
npm run build:all
npm run tauri build -- --target x86_64-pc-windows-msvc &
npm run tauri build -- --target x86_64-apple-darwin &
npm run tauri build -- --target x86_64-unknown-linux-gnu &
wait
```

---

## 🆘 Debug Commands

### Check Rust Toolchain
```bash
rustc --version
rustup show
rustup target list --installed
```

### Check Build Environment
```bash
# Windows
node --version
npm --version
cargo --version
rustc --version

# macOS
xcode-select --print-path
node --version
npm --version
cargo --version

# Linux
node --version
npm --version
cargo --version
gcc --version
pkg-config --list-all
```

### Test Frontend Only
```bash
npm run build
npm run preview
```

### Test Backend Only
```bash
cd engine
go build -o meetrec .
./meetrec
```

---

## 📦 Distribution Channels

### Windows
- **MSI Installer**: Copy `Meet Capture_0.1.0_x64_en-US.msi` to distribution
- **Direct Executable**: Copy `tauri-app.exe` with dependencies

### macOS
- **App Bundle**: Drag `Meet Capture.app` to Applications
- **DMG Installer**: Double-click `Meet Capture_0.1.0_x64.dmg`
- **Code Signing Required**: For App Store or distribution outside MAS

### Linux
- **AppImage**: Single executable file `Meet Capture_0.1.0_amd64.AppImage`
- **.deb Package**: Install via `apt` or package manager
- **Universal**: No dependencies, runs on any x86_64 Linux

---

## 🎯 Quick Checklist for Release

- [ ] Update version in `package.json`, `Cargo.toml`, `tauri.conf.json`
- [ ] Test locally on your OS: `npm run dev:all`
- [ ] Build production package: `npm run build:all && npm run tauri build`
- [ ] Test built application before distribution
- [ ] Copy artifacts to `Downloads/[os]/` folder
- [ ] Create release notes with changes and features
- [ ] Sign code (Windows: optional, macOS: required, Linux: optional)
- [ ] Upload to distribution platform
- [ ] Test installation on fresh system
- [ ] Announce release to users
