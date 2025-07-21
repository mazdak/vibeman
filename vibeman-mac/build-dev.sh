#!/bin/bash

# Vibeman Development Build Script
# Quick build for development and testing
set -e

echo "🚀 Building Vibeman for development..."

# Clean previous builds
rm -rf .build/debug
rm -rf Vibeman.app

# Build Swift app for debug
echo "📦 Building Swift app for debug..."
swift build

if [ $? -ne 0 ]; then
  echo "❌ Swift build failed!"
  exit 1
fi

# Build Go server binary for current architecture only
echo "🔧 Building Go server binary (debug)..."
cd ..
go build -o vibeman-server .
if [ $? -ne 0 ]; then
  echo "❌ Go build failed!"
  exit 1
fi

# Create CLI symlink
cp vibeman-server vibeman

# Build web app for development
echo "🌐 Building web app (development)..."
if [ -d "vibeman-web" ]; then
  cd vibeman-web
  if command -v bun >/dev/null 2>&1; then
    echo "Building web app with Bun (dev mode)..."
    bun install
    bun run build --outdir=dist --target=browser --sourcemap=linked
    if [ $? -ne 0 ]; then
      echo "❌ Web app build failed!"
      exit 1
    fi
  else
    echo "⚠️ Bun not found, skipping web app build"
  fi
  cd ..
fi

# Return to swift-wrapper directory
cd swift-wrapper

# Create app bundle
echo "📱 Creating app bundle..."
mkdir -p Vibeman.app/Contents/MacOS
mkdir -p Vibeman.app/Contents/Resources

# Copy Swift executable (debug build)
cp .build/debug/Vibeman Vibeman.app/Contents/MacOS/

# Copy Go binaries
if [ -f "../vibeman-server" ]; then
  cp ../vibeman-server Vibeman.app/Contents/MacOS/
fi
if [ -f "../vibeman" ]; then
  cp ../vibeman Vibeman.app/Contents/MacOS/
fi

# Copy built web app
if [ -d "../vibeman-web/dist" ]; then
  cp -r ../vibeman-web/dist Vibeman.app/Contents/Resources/web-app
fi

# Copy Info.plist
cp Info.plist Vibeman.app/Contents/

# Copy assets
if [ -d "Sources/Assets.xcassets" ]; then
  cp -r Sources/Assets.xcassets Vibeman.app/Contents/Resources/
fi

# Make executables
chmod +x Vibeman.app/Contents/MacOS/Vibeman
chmod +x Vibeman.app/Contents/MacOS/vibeman-server 2>/dev/null
chmod +x Vibeman.app/Contents/MacOS/vibeman 2>/dev/null

echo "✅ Development build complete!"
echo "📁 App bundle: Vibeman.app"
echo ""
echo "To run: open Vibeman.app"
echo "To debug: lldb Vibeman.app/Contents/MacOS/Vibeman"