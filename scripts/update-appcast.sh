#!/bin/bash

# Vibeman Appcast Update Script
# Usage: ./scripts/update-appcast.sh <version> <release-notes>
# Example: ./scripts/update-appcast.sh "1.0.1" "Bug fixes and improvements"

if [ $# -lt 2 ]; then
    echo "Usage: $0 <version> <release-notes>"
    echo "Example: $0 '1.0.1' 'Bug fixes and improvements'"
    exit 1
fi

VERSION="$1"
RELEASE_NOTES="$2"
APPCAST_FILE="appcast.xml"
DATE=$(date -u +"%a, %d %b %Y %H:%M:%S +0000")

echo "Updating appcast for version $VERSION..."

# Create backup
cp "$APPCAST_FILE" "$APPCAST_FILE.backup"

# Create new item XML
NEW_ITEM=$(cat << EOF
        <item>
            <title>Version $VERSION</title>
            <description><![CDATA[
                <h3>What's New in $VERSION:</h3>
                <ul>
                    <li>$RELEASE_NOTES</li>
                </ul>
            ]]></description>
            <pubDate>$DATE</pubDate>
            <sparkle:version>$VERSION</sparkle:version>
            <sparkle:shortVersionString>$VERSION</sparkle:shortVersionString>
            <sparkle:minimumSystemVersion>13.0</sparkle:minimumSystemVersion>
            <enclosure 
                url="https://github.com/mazdak/vibeman/releases/download/v$VERSION/Vibeman.app.zip"
                length="5242880"
                type="application/zip"
                sparkle:edSignature="placeholder-signature"/>
        </item>
        
        <!-- Previous version -->
EOF
)

# Insert new item after the channel opening but before existing items
sed "/<!-- Current version entry/a\\
$NEW_ITEM" "$APPCAST_FILE" > "$APPCAST_FILE.tmp" && mv "$APPCAST_FILE.tmp" "$APPCAST_FILE"

echo "Updated $APPCAST_FILE with version $VERSION"
echo "Don't forget to:"
echo "1. Update version in Info.plist"
echo "2. Build and package the app"
echo "3. Create GitHub release with Vibeman.app.zip"
echo "4. Update the file size in appcast.xml"
echo "5. Commit and push appcast.xml changes"