#!/bin/bash
set -e

# Build the enhanced AI container image
echo "Building Vibeman AI Assistant Docker image..."

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to the docker directory
cd "$PROJECT_ROOT/docker/ai-assistant"

# Build the image
docker build -t vibeman/ai-assistant:latest .

echo "Successfully built vibeman/ai-assistant:latest"

# Optional: Tag with version
if [ -n "$1" ]; then
    VERSION=$1
    docker tag vibeman/ai-assistant:latest vibeman/ai-assistant:$VERSION
    echo "Also tagged as vibeman/ai-assistant:$VERSION"
fi

echo ""
echo "To test the image, run:"
echo "  docker run -it --rm vibeman/ai-assistant:latest"
echo ""
echo "To push to a registry:"
echo "  docker push vibeman/ai-assistant:latest"