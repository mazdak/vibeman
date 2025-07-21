#!/bin/bash
# Build script for Vibeman AI Assistant Docker image

set -e

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Image name and tag
IMAGE_NAME="vibeman/ai-assistant"
IMAGE_TAG="${1:-latest}"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

echo "🔨 Building Vibeman AI Assistant Docker image..."
echo "📦 Image: ${FULL_IMAGE}"

# Build the image
docker build -t "${FULL_IMAGE}" "${SCRIPT_DIR}"

# Tag as latest if not already
if [ "${IMAGE_TAG}" != "latest" ]; then
    docker tag "${FULL_IMAGE}" "${IMAGE_NAME}:latest"
fi

echo "✅ Build complete!"
echo ""
echo "📋 Image details:"
docker images "${IMAGE_NAME}"
echo ""
echo "🚀 To test the image:"
echo "   docker run -it --rm ${FULL_IMAGE}"