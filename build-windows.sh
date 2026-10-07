#!/bin/bash
# build-windows.sh - Build Meet Capture for Windows

set -e

echo "🔨 Building Meet Capture for Windows (x86_64)..."
echo ""

# Check if on the right system
if [[ "$OSTYPE" != "msys" && "$OSTYPE" != "cygwin" && "$OSTYPE" != "win32" ]]; then
    echo "⚠️  WARNING: You are not on Windows. This script is designed for Windows."
    echo "   To build Windows on macOS/Linux, use: npm run tauri build -- --target x86_64-pc-windows-msvc"
    exit 1
fi

# Install dependencies
echo "📦 Installing dependencies..."
npm install

# Build frontend and backend
echo "🏗️  Building frontend and backend..."
npm run build:all

# Build Tauri app for Windows
echo "📦 Building Tauri application..."
npm run tauri build -- --target x86_64-pc-windows-msvc

# Create output directory
mkdir -p Downloads/windows

# Copy artifacts
echo "📁 Copying artifacts to Downloads/windows..."
if [ -d "src-tauri/target/x86_64-pc-windows-msvc/release/bundle/msi" ]; then
    cp "src-tauri/target/x86_64-pc-windows-msvc/release/bundle/msi/"*.msi "Downloads/windows/" 2>/dev/null || true
fi

# Copy executable
if [ -f "src-tauri/target/x86_64-pc-windows-msvc/release/tauri-app.exe" ]; then
    cp "src-tauri/target/x86_64-pc-windows-msvc/release/tauri-app.exe" "Downloads/windows/"
fi

echo ""
echo "✅ Build complete!"
echo "📦 Artifacts saved to: Downloads/windows/"
echo ""
echo "📋 Files created:"
ls -lh "Downloads/windows/"
