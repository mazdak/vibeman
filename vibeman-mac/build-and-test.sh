#!/bin/bash

# Vibeman Swift App Build and Test Script

set -e

echo "🚀 Building Vibeman macOS App..."

# Check if we're in the right directory
if [ ! -f "Package.swift" ]; then
    echo "❌ Error: Package.swift not found. Please run this script from the swift-wrapper directory."
    exit 1
fi

# Build the project
echo "📦 Building Swift package..."
swift build

# Run tests
echo "🧪 Running tests..."
swift test

# Build for release
echo "🎯 Building release version..."
swift build -c release

# Create app bundle if make is available
if command -v make >/dev/null 2>&1; then
    echo "🔨 Creating app bundle..."
    make app
    echo "✅ App bundle created at .build/Vibeman.app"
else
    echo "⚠️  Make not available - skipping app bundle creation"
fi

echo "✅ Build completed successfully!"
echo ""
echo "To run the app:"
echo "  Development: .build/debug/Vibeman"
echo "  Release: .build/release/Vibeman"
echo "  App Bundle: open .build/Vibeman.app"