#!/bin/bash
# build-macos.sh - Build Meet Capture for macOS

set -e

echo "🔨 Building Meet Capture for macOS..."
echo ""

# Check if on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "⚠️  WARNING: You are not on macOS. This script is designed for macOS."
    echo "   To build for macOS on another system, you'll need a Mac with Xcode."
    exit 1
fi

# Install Xcode CLT if needed
if ! command -v xcode-select &> /dev/null; then
    echo "📦 Installing Xcode Command Line Tools..."
    xcode-select --install
fi

# Install dependencies
echo "📦 Installing dependencies..."
npm install

# Add Rust targets
echo "🦀 Adding Rust targets..."
rustup target add x86_64-apple-darwin aarch64-apple-darwin

# Build frontend and backend
echo "🏗️  Building frontend and backend..."
npm run build:all

# Ask user which architecture to build
echo ""
echo "Which architecture would you like to build for?"
echo "1) x86_64 (Intel)"
echo "2) aarch64 (Apple Silicon/M1+)"
echo "3) universal (both)"
echo ""
read -p "Enter choice (1-3): " CHOICE

case $CHOICE in
    1)
        TARGETS=("x86_64-apple-darwin")
        echo "Building for x86_64 (Intel)..."
        ;;
    2)
        TARGETS=("aarch64-apple-darwin")
        echo "Building for aarch64 (Apple Silicon)..."
        ;;
    3)
        TARGETS=("universal-apple-darwin")
        echo "Building universal binary..."
        ;;
    *)
        echo "Invalid choice. Building for current architecture..."
        TARGETS=("$(rustc -vV | grep host | awk '{print $2}')")
        ;;
esac

# Build Tauri app
for target in "${TARGETS[@]}"; do
    echo "📦 Building Tauri application for $target..."
    npm run tauri build -- --target "$target"
done

# Create output directory
mkdir -p Downloads/macos

# Copy artifacts
echo "📁 Copying artifacts to Downloads/macos..."
for target in "${TARGETS[@]}"; do
    BUNDLE_PATH="src-tauri/target/$target/release/bundle"
    
    # Copy app bundle
    if [ -d "$BUNDLE_PATH/macos/Meet Capture.app" ]; then
        cp -r "$BUNDLE_PATH/macos/Meet Capture.app" "Downloads/macos/"
    fi
    
    # Copy DMG
    if [ -f "$BUNDLE_PATH/dmg/"*.dmg ]; then
        cp "$BUNDLE_PATH/dmg/"*.dmg "Downloads/macos/"
    fi
done

echo ""
echo "✅ Build complete!"
echo "📦 Artifacts saved to: Downloads/macos/"
echo ""
echo "📋 Files created:"
ls -lh "Downloads/macos/"
echo ""
echo "To codesign for distribution:"
echo "  codesign --deep --force --verify --verbose --sign \"Developer ID Application\" \"Downloads/macos/Meet Capture.app\""
