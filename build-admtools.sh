#!/bin/bash
#
# Build multi-arch ReignX admtools image for macOS
# Supports both arm64 (Apple Silicon) and amd64 (Intel)
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

IMAGE_NAME="${IMAGE_NAME:-reignx-admtools}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-localhost:5000}"

echo "=== Building ReignX Admtools Image ==="
echo "Image: $REGISTRY/$IMAGE_NAME:$IMAGE_TAG"
echo "Platform: macOS (linux/arm64, linux/amd64)"
echo ""

# Step 1: Build ReignX Agent for both architectures
echo "Step 1: Building ReignX Agent binaries..."

# Build for arm64
if [ ! -f bin/agent-linux-arm64 ]; then
    echo "Building agent for linux/arm64..."
    GOOS=linux GOARCH=arm64 go build -o bin/agent-linux-arm64 ./cmd/agent
    echo "✓ agent-linux-arm64 built"
fi

# Build for amd64
if [ ! -f bin/agent-linux-amd64 ]; then
    echo "Building agent for linux/amd64..."
    GOOS=linux GOARCH=amd64 go build -o bin/agent-linux-amd64 ./cmd/agent
    echo "✓ agent-linux-amd64 built"
fi

echo ""

# Step 2: Build multi-arch image
echo "Step 2: Building multi-arch container image..."

# Check if using podman or docker
if command -v podman > /dev/null; then
    BUILDER="podman"
    echo "Using Podman"
elif command -v docker > /dev/null; then
    BUILDER="docker"
    echo "Using Docker"
else
    echo "Error: Neither podman nor docker found!"
    exit 1
fi

# Build for both architectures
echo ""
echo "Building for linux/arm64 and linux/amd64..."

if [ "$BUILDER" = "podman" ]; then
    # Podman build
    podman build \
        --platform linux/arm64,linux/amd64 \
        --manifest "$IMAGE_NAME:$IMAGE_TAG" \
        --build-arg ROOT_PASSWORD="${ROOT_PASSWORD:-changeme}" \
        --build-arg USER_PASSWORD="${USER_PASSWORD:-changeme}" \
        -f Containerfile.admtools \
        .

    # Tag and push to registry if specified
    if [ "$REGISTRY" != "localhost:5000" ]; then
        echo ""
        echo "Pushing to registry: $REGISTRY"
        podman manifest push "$IMAGE_NAME:$IMAGE_TAG" \
            "docker://$REGISTRY/$IMAGE_NAME:$IMAGE_TAG"
    fi

elif [ "$BUILDER" = "docker" ]; then
    # Docker buildx (multi-arch support)
    if ! docker buildx version > /dev/null 2>&1; then
        echo "Error: docker buildx not available!"
        echo "Install buildx: https://docs.docker.com/buildx/working-with-buildx/"
        exit 1
    fi

    # Create builder if it doesn't exist
    if ! docker buildx inspect reignx-builder > /dev/null 2>&1; then
        echo "Creating buildx builder..."
        docker buildx create --name reignx-builder --use
    else
        docker buildx use reignx-builder
    fi

    # Build multi-arch
    docker buildx build \
        --platform linux/arm64,linux/amd64 \
        -t "$IMAGE_NAME:$IMAGE_TAG" \
        --build-arg ROOT_PASSWORD="${ROOT_PASSWORD:-changeme}" \
        --build-arg USER_PASSWORD="${USER_PASSWORD:-changeme}" \
        -f Containerfile.admtools \
        --load \
        .

    # Push to registry if specified
    if [ "$REGISTRY" != "localhost:5000" ]; then
        echo ""
        echo "Pushing to registry: $REGISTRY"
        docker tag "$IMAGE_NAME:$IMAGE_TAG" "$REGISTRY/$IMAGE_NAME:$IMAGE_TAG"
        docker push "$REGISTRY/$IMAGE_NAME:$IMAGE_TAG"
    fi
fi

echo ""
echo "✅ Build complete!"
echo ""
echo "Image: $IMAGE_NAME:$IMAGE_TAG"
echo "Platforms: linux/arm64, linux/amd64"
echo ""
echo "To run:"
echo "  # SSH only mode"
echo "  $BUILDER run -d -p 2222:22 $IMAGE_NAME:$IMAGE_TAG"
echo ""
echo "  # With ReignX Agent"
echo "  $BUILDER run -d -p 2222:22 \\"
echo "    -e ENABLE_AGENT=true \\"
echo "    -e REIGNX_SERVER=host.docker.internal:50051 \\"
echo "    -e NODE_ID=admtools-001 \\"
echo "    $IMAGE_NAME:$IMAGE_TAG"
echo ""
echo "  # SSH login (default password: changeme, set via ROOT_PASSWORD)"
echo "  ssh root@localhost -p 2222"
echo ""
