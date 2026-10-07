#!/bin/bash
# build-linux.sh - Build Meet Capture for Linux

set -e

echo "🔨 Building Meet Capture for Linux..."
echo ""

# Check if on Linux
if [[ "$OSTYPE" != "linux-gnu"* && "$OSTYPE" != "linux"* ]]; then
    echo "⚠️  WARNING: You are not on Linux. This script is designed for Linux."
    echo "   To build for Linux on another system, ensure you have Rust cross-compilation setup."
    exit 1
fi

# Detect Linux distribution
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    OS="linux"
fi

# Install dependencies based on distro
echo "📦 Installing system dependencies..."
case $OS in
    ubuntu|debian)
        echo "Detected Debian/Ubuntu..."
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
        ;;
    fedora|rhel|centos)
        echo "Detected Fedora/RHEL/CentOS..."
        sudo dnf install -y \
            gcc \
            glibc-devel \
            openssl-devel \
            gtk3-devel \
            librsvg2-devel \
            libappindicator-gtk3-devel
        ;;
    arch)
        echo "Detected Arch Linux..."
        sudo pacman -S --noconfirm base-devel gtk3 librsvg openssl
        ;;
    *)
        echo "⚠️  Unknown Linux distribution: $OS"
        echo "   Please install: build-essential, libssl-dev, libgtk-3-dev, librsvg2-dev"
        ;;
esac

# Install Node dependencies
echo "📦 Installing Node dependencies..."
npm install

# Build frontend and backend
echo "🏗️  Building frontend and backend..."
npm run build:all

# Ask user which bundle format
echo ""
echo "Which bundle format would you like?"
echo "1) AppImage (recommended)"
echo "2) .deb package"
echo "3) Both"
echo ""
read -p "Enter choice (1-3): " CHOICE

case $CHOICE in
    1)
        BUNDLE_OPTS=""
        echo "Building AppImage..."
        ;;
    2)
        BUNDLE_OPTS="--bundles deb"
        echo "Building .deb package..."
        ;;
    3)
        BUNDLE_OPTS="--bundles appimage,deb"
        echo "Building both AppImage and .deb..."
        ;;
    *)
        echo "Invalid choice. Building AppImage..."
        BUNDLE_OPTS=""
        ;;
esac

# Build Tauri app
echo "📦 Building Tauri application..."
npm run tauri build -- --target x86_64-unknown-linux-gnu $BUNDLE_OPTS

# Create output directory
mkdir -p Downloads/linux

# Copy artifacts
echo "📁 Copying artifacts to Downloads/linux..."
BUNDLE_PATH="src-tauri/target/x86_64-unknown-linux-gnu/release/bundle"

if [ -d "$BUNDLE_PATH/appimage" ]; then
    cp "$BUNDLE_PATH/appimage/"*.AppImage "Downloads/linux/" 2>/dev/null || true
    chmod +x "Downloads/linux/"*.AppImage
fi

if [ -d "$BUNDLE_PATH/deb" ]; then
    cp "$BUNDLE_PATH/deb/"*.deb "Downloads/linux/" 2>/dev/null || true
fi

echo ""
echo "✅ Build complete!"
echo "📦 Artifacts saved to: Downloads/linux/"
echo ""
echo "📋 Files created:"
ls -lh "Downloads/linux/"
echo ""
echo "To install .deb package:"
echo "  sudo apt install ./Downloads/linux/*.deb"
echo ""
echo "To run AppImage:"
echo "  ./Downloads/linux/*.AppImage"
