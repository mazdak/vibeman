#!/bin/bash

# Vibeman Release Build Script
# For development, use: swift build && swift run
# This script is for creating distributable releases

# Parse command line arguments
NOTARIZE=false
while [[ $# -gt 0 ]]; do
  case $1 in
  --notarize)
    NOTARIZE=true
    shift
    ;;
  *)
    echo "Unknown option: $1"
    echo "Usage: $0 [--notarize]"
    exit 1
    ;;
  esac
done

# Generate version info
GIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date '+%Y-%m-%d')

# Version can be overridden with VIBEMAN_VERSION environment variable
VERSION="${VIBEMAN_VERSION:-1.0.0}"

echo "⚡ Building Vibeman version $VERSION..."

# Clean previous builds
rm -rf .build/release
rm -rf Vibeman.app
rm -rf ../vibeman-server
rm -rf ../vibeman

# Create version file from template if it exists
if [ -f "Sources/VersionInfo.swift.template" ]; then
    sed -e "s/VERSION_PLACEHOLDER/$VERSION/g" \
        -e "s/GIT_HASH_PLACEHOLDER/$GIT_HASH/g" \
        -e "s/BUILD_DATE_PLACEHOLDER/$BUILD_DATE/g" \
        Sources/VersionInfo.swift.template > Sources/VersionInfo.swift
    echo "Generated VersionInfo.swift from template"
else
    echo "Creating fallback VersionInfo.swift..."
    cat >Sources/VersionInfo.swift <<EOF
import Foundation

struct VersionInfo {
    static let version = "$VERSION"
    static let gitHash = "$GIT_HASH"
    static let buildDate = "$BUILD_DATE"
    
    static var displayVersion: String {
        if gitHash != "unknown" && !gitHash.isEmpty {
            let shortHash = String(gitHash.prefix(7))
            return "\(version) (\(shortHash))"
        }
        return version
    }
    
    static var fullVersionInfo: String {
        var info = "Vibeman \(version)"
        if gitHash != "unknown" && !gitHash.isEmpty {
            let shortHash = String(gitHash.prefix(7))
            info += " • \(shortHash)"
        }
        if buildDate.count > 0 {
            info += " • \(buildDate)"
        }
        return info
    }
}
EOF
fi

# Build Swift app for release (universal binary)
echo "📦 Building Swift app for release..."
swift build -c release --arch arm64 --arch x86_64

if [ $? -ne 0 ]; then
  echo "❌ Swift build failed!"
  exit 1
fi

# Build Go server binary (universal binary)
echo "🔧 Building Go server binary..."
cd ..
echo "Building for arm64..."
GOOS=darwin GOARCH=arm64 go build -o vibeman-server-arm64 .
if [ $? -ne 0 ]; then
  echo "❌ Go build failed for arm64!"
  exit 1
fi

echo "Building for amd64..."
GOOS=darwin GOARCH=amd64 go build -o vibeman-server-amd64 .
if [ $? -ne 0 ]; then
  echo "❌ Go build failed for amd64!"
  exit 1
fi

# Create universal binary using lipo
echo "Creating universal Go binary..."
lipo -create -output vibeman-server vibeman-server-arm64 vibeman-server-amd64
if [ $? -ne 0 ]; then
  echo "❌ Failed to create universal Go binary!"
  exit 1
fi

# Create CLI symlink (same as server)
cp vibeman-server vibeman

# Clean up architecture-specific binaries
rm -f vibeman-server-arm64 vibeman-server-amd64

# Build Bun web app for production
echo "🌐 Building web app..."
if [ -d "vibeman-web" ]; then
  cd vibeman-web
  if command -v bun >/dev/null 2>&1; then
    echo "Building web app with Bun..."
    bun install
    
    # Check if this is a standalone Bun app or a web app
    if [ -f "src/index.ts" ] && grep -q "Bun.serve" "src/index.ts" 2>/dev/null; then
      # Standalone Bun server - compile to binary (ARM64 only for now)
      echo "  - Building standalone Bun server (ARM64 only)..."
      bun build --compile --target=bun-darwin-arm64 --minify --sourcemap ./src/index.ts --outfile vibeman-web-server
      if [ $? -ne 0 ]; then
        echo "❌ Bun standalone build failed!"
        exit 1
      fi
      echo "  ℹ️  Note: Bun executable is ARM64-only. Intel Mac support planned for future release."
    else
      # Regular web app - build static assets
      echo "  - Building static web assets..."
      bun run build --minify --outdir=dist --target=browser
      if [ $? -ne 0 ]; then
        echo "❌ Web app build failed!"
        exit 1
      fi
    fi
  else
    echo "❌ Bun not found! Please install Bun: curl -fsSL https://bun.sh/install | bash"
    exit 1
  fi
  cd ..
else
  echo "⚠️ vibeman-web directory not found, skipping web app build"
fi

# Return to swift-wrapper directory
cd swift-wrapper

# Create app bundle
echo "📱 Creating app bundle..."
mkdir -p Vibeman.app/Contents/MacOS
mkdir -p Vibeman.app/Contents/Resources

# Copy Swift executable (universal binary)
cp .build/apple/Products/Release/Vibeman Vibeman.app/Contents/MacOS/

# Copy Go server binary
if [ -f "../vibeman-server" ]; then
  cp ../vibeman-server Vibeman.app/Contents/MacOS/
  echo "Copied vibeman-server binary"
else
  echo "⚠️ vibeman-server not found, server functionality will not work"
fi

# Copy CLI binary (same as server)
if [ -f "../vibeman" ]; then
  cp ../vibeman Vibeman.app/Contents/MacOS/
  echo "Copied vibeman CLI binary"
else
  echo "⚠️ vibeman CLI not found"
fi

# Copy built web app
if [ -d "../vibeman-web/dist" ]; then
  cp -r ../vibeman-web/dist Vibeman.app/Contents/Resources/web-app
  echo "Copied built web app"
else
  echo "⚠️ Built web app not found, web interface will not work"
fi

# Create proper Info.plist with dynamic version
echo "📄 Creating Info.plist..."
cat >Vibeman.app/Contents/Info.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>Vibeman</string>
    <key>CFBundleIdentifier</key>
    <string>com.vibeman.app</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>Vibeman</string>
    <key>CFBundleDisplayName</key>
    <string>Vibeman</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    <key>CFBundleVersion</key>
    <string>$BUILD_DATE</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSPrincipalClass</key>
    <string>NSApplication</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
    <key>NSAppTransportSecurity</key>
    <dict>
        <key>NSAllowsLocalNetworking</key>
        <true/>
        <key>NSExceptionDomains</key>
        <dict>
            <key>localhost</key>
            <dict>
                <key>NSExceptionAllowsInsecureHTTPLoads</key>
                <true/>
                <key>NSExceptionMinimumTLSVersion</key>
                <string>TLSv1.0</string>
            </dict>
        </dict>
    </dict>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.developer-tools</string>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright © 2024 Vibeman. All rights reserved.</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
</dict>
</plist>
EOF

# Generate app icon from source image
if [ -f "Vibeman-Icon.png" ]; then
  echo "🎨 Generating app icons..."
  
  # Create iconset directory
  mkdir -p Vibeman.iconset
  
  # Generate all required icon sizes
  sips -z 16 16 Vibeman-Icon.png --out Vibeman.iconset/icon_16x16.png 2>/dev/null
  sips -z 32 32 Vibeman-Icon.png --out Vibeman.iconset/icon_16x16@2x.png 2>/dev/null
  sips -z 32 32 Vibeman-Icon.png --out Vibeman.iconset/icon_32x32.png 2>/dev/null
  sips -z 64 64 Vibeman-Icon.png --out Vibeman.iconset/icon_32x32@2x.png 2>/dev/null
  sips -z 128 128 Vibeman-Icon.png --out Vibeman.iconset/icon_128x128.png 2>/dev/null
  sips -z 256 256 Vibeman-Icon.png --out Vibeman.iconset/icon_128x128@2x.png 2>/dev/null
  sips -z 256 256 Vibeman-Icon.png --out Vibeman.iconset/icon_256x256.png 2>/dev/null
  sips -z 512 512 Vibeman-Icon.png --out Vibeman.iconset/icon_256x256@2x.png 2>/dev/null
  sips -z 512 512 Vibeman-Icon.png --out Vibeman.iconset/icon_512x512.png 2>/dev/null
  sips -z 1024 1024 Vibeman-Icon.png --out Vibeman.iconset/icon_512x512@2x.png 2>/dev/null

  # Create icns file directly in app bundle
  if command -v iconutil >/dev/null 2>&1; then
    iconutil -c icns Vibeman.iconset -o Vibeman.app/Contents/Resources/AppIcon.icns 2>/dev/null || echo "Note: iconutil failed, app will use default icon"
  fi

  # Clean up temporary files
  rm -rf Vibeman.iconset
else
  echo "⚠️ Vibeman-Icon.png not found, app will use default icon"
fi

# Copy other assets if they exist
if [ -d "Sources/Assets.xcassets" ]; then
  cp -r Sources/Assets.xcassets Vibeman.app/Contents/Resources/
  echo "Copied app assets"
fi

# Make executables
chmod +x Vibeman.app/Contents/MacOS/Vibeman
chmod +x Vibeman.app/Contents/MacOS/vibeman-server 2>/dev/null
chmod +x Vibeman.app/Contents/MacOS/vibeman 2>/dev/null

# Create entitlements file for hardened runtime
echo "🔒 Creating entitlements for hardened runtime..."
cat >Vibeman.entitlements <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.device.audio-input</key>
    <true/>
    <key>com.apple.security.network.client</key>
    <true/>
    <key>com.apple.security.network.server</key>
    <true/>
    <key>com.apple.security.files.user-selected.read-write</key>
    <true/>
    <key>com.apple.security.automation.apple-events</key>
    <true/>
</dict>
</plist>
EOF

# Function to sign the app with a given identity
sign_app() {
  local identity="$1"
  local identity_name="$2"

  if [ -n "$identity_name" ]; then
    echo "🔏 Code signing app with: $identity_name ($identity)"
  else
    echo "🔏 Code signing app with: $identity"
  fi

  # Sign individual binaries first
  if [ -f "Vibeman.app/Contents/MacOS/vibeman-server" ]; then
    codesign --force --sign "$identity" --options runtime --entitlements Vibeman.entitlements Vibeman.app/Contents/MacOS/vibeman-server
  fi
  if [ -f "Vibeman.app/Contents/MacOS/vibeman" ]; then
    codesign --force --sign "$identity" --options runtime --entitlements Vibeman.entitlements Vibeman.app/Contents/MacOS/vibeman
  fi

  # Sign the main app
  codesign --force --deep --sign "$identity" --options runtime --entitlements Vibeman.entitlements Vibeman.app
  if [ $? -eq 0 ]; then
    echo "🔍 Verifying signature..."
    codesign --verify --verbose Vibeman.app
    echo "✅ App signed successfully"
    return 0
  else
    echo "❌ Code signing failed"
    return 1
  fi
}

# Optional: Code sign the app (requires Apple Developer account)
SIGNING_IDENTITY=""
SIGNING_NAME=""

if [ -n "$CODE_SIGN_IDENTITY" ]; then
  SIGNING_IDENTITY="$CODE_SIGN_IDENTITY"
else
  # Try to auto-detect Developer ID (use the first one found)
  DETECTED_HASH=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | awk '{print $2}')
  DETECTED_NAME=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | awk '{print $3}' | tr -d '"')
  if [ -n "$DETECTED_HASH" ]; then
    echo "🔍 Auto-detected signing identity: $DETECTED_NAME"
    SIGNING_IDENTITY="$DETECTED_HASH"
    SIGNING_NAME="$DETECTED_NAME"
  fi
fi

if [ -n "$SIGNING_IDENTITY" ]; then
  sign_app "$SIGNING_IDENTITY" "$SIGNING_NAME"
else
  echo "💡 No Developer ID found. App will be unsigned."
  echo "💡 To sign the app, get a Developer ID certificate from Apple Developer Portal."
fi

# Clean up entitlements file
rm -f Vibeman.entitlements

# Notarization (requires code signing first)
if [ "$NOTARIZE" = true ]; then
  echo ""
  echo "🔐 Starting notarization process..."

  # Check for required environment variables
  if [ -z "$VIBEMAN_APPLE_ID" ] || [ -z "$VIBEMAN_APPLE_PASSWORD" ] || [ -z "$VIBEMAN_TEAM_ID" ]; then
    echo "❌ Notarization requires the following environment variables:"
    echo "   VIBEMAN_APPLE_ID - Your Apple ID email"
    echo "   VIBEMAN_APPLE_PASSWORD - App-specific password for notarization"
    echo "   VIBEMAN_TEAM_ID - Your Apple Developer Team ID"
    echo ""
    echo "To create an app-specific password:"
    echo "1. Go to https://appleid.apple.com/account/manage"
    echo "2. Sign in and go to Security > App-Specific Passwords"
    echo "3. Generate a new password for Vibeman notarization"
    echo ""
    exit 1
  fi

  # Check if app is signed
  if codesign -dvvv Vibeman.app 2>&1 | grep -q "Signature=adhoc"; then
    echo "❌ App must be properly signed before notarization (not adhoc signed)"
    echo "Please ensure CODE_SIGN_IDENTITY is set or a Developer ID is available"
    exit 1
  fi

  # Create a zip file for notarization
  echo "Creating zip for notarization..."
  ditto -c -k --keepParent Vibeman.app Vibeman.zip

  # Submit for notarization
  echo "📤 Submitting to Apple for notarization..."
  xcrun notarytool submit Vibeman.zip \
    --apple-id "$VIBEMAN_APPLE_ID" \
    --password "$VIBEMAN_APPLE_PASSWORD" \
    --team-id "$VIBEMAN_TEAM_ID" \
    --wait 2>&1 | tee notarization.log

  # Check if notarization was successful
  if grep -q "status: Accepted" notarization.log; then
    # Staple the notarization ticket to the app
    echo "📎 Stapling notarization ticket..."
    xcrun stapler staple Vibeman.app

    if [ $? -eq 0 ]; then
      echo "✅ Notarization ticket stapled successfully!"
    else
      echo "⚠️ Failed to staple notarization ticket, but app is notarized"
    fi
  else
    echo "❌ Notarization failed. Check notarization.log for details"
    echo ""
    echo "Common issues:"
    echo "- Ensure your Apple ID has accepted all developer agreements"
    echo "- Check that your app-specific password is correct"
    echo "- Verify your Team ID is correct"
    exit 1
  fi

  # Clean up
  rm -f Vibeman.zip
  rm -f notarization.log
fi

# Create DMG for distribution
if command -v create-dmg >/dev/null 2>&1; then
  echo "💿 Creating DMG..."
  create-dmg \
    --volname "Vibeman" \
    --volicon "Vibeman.app/Contents/Resources/AppIcon.icns" \
    --window-pos 200 120 \
    --window-size 600 300 \
    --icon-size 100 \
    --icon "Vibeman.app" 175 120 \
    --hide-extension "Vibeman.app" \
    --app-drop-link 425 120 \
    "Vibeman-$VERSION.dmg" \
    "Vibeman.app" 2>/dev/null || echo "Note: DMG creation failed, but app bundle is ready"
else
  echo "💡 create-dmg not found. Install with: brew install create-dmg"
  echo "💡 DMG creation skipped, but app bundle is ready"
fi

echo "✅ Build complete!"
echo ""
echo "📁 App bundle: Vibeman.app"
if [ -f "Vibeman-$VERSION.dmg" ]; then
  echo "💿 DMG: Vibeman-$VERSION.dmg"
fi
echo ""
echo "🧪 Testing app bundle structure..."
echo "Swift app: $([ -f "Vibeman.app/Contents/MacOS/Vibeman" ] && echo "✅" || echo "❌")"
echo "Go server: $([ -f "Vibeman.app/Contents/MacOS/vibeman-server" ] && echo "✅" || echo "❌")"
echo "CLI binary: $([ -f "Vibeman.app/Contents/MacOS/vibeman" ] && echo "✅" || echo "❌")"
echo "Web app: $([ -d "Vibeman.app/Contents/Resources/web-app" ] && echo "✅" || echo "❌")"
echo "App icon: $([ -f "Vibeman.app/Contents/Resources/AppIcon.icns" ] && echo "✅" || echo "❌")"
echo ""
open -R Vibeman.app