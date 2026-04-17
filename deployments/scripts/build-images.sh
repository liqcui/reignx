#!/bin/bash
# Build all Docker images for ReignX platform

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
VERSION="${VERSION:-latest}"

echo "Building ReignX Docker images (version: $VERSION)..."

cd "$PROJECT_ROOT"

# Build reignxd
echo "Building reignxd..."
docker build \
  -f deployments/docker/Dockerfile.reignxd \
  -t reignx/reignxd:${VERSION} \
  -t reignx/reignxd:latest \
  .

# Build reignx-agent
echo "Building reignx-agent..."
docker build \
  -f deployments/docker/Dockerfile.agent \
  -t reignx/reignx-agent:${VERSION} \
  -t reignx/reignx-agent:latest \
  .

# Build migrate
echo "Building migrate..."
docker build \
  -f deployments/docker/Dockerfile.migrate \
  -t reignx/migrate:${VERSION} \
  -t reignx/migrate:latest \
  .

# Build CLI
echo "Building reignxctl..."
docker build \
  -f deployments/docker/Dockerfile.cli \
  -t reignx/reignxctl:${VERSION} \
  -t reignx/reignxctl:latest \
  .

echo ""
echo "✓ All images built successfully!"
echo ""
echo "Images:"
echo "  - reignx/reignxd:${VERSION}"
echo "  - reignx/reignx-agent:${VERSION}"
echo "  - reignx/migrate:${VERSION}"
echo "  - reignx/reignxctl:${VERSION}"
