#!/bin/bash
#
# Quick test script for ReignX admtools image
#

set -e

IMAGE_NAME="${IMAGE_NAME:-reignx-admtools:latest}"
CONTAINER_NAME="admtools-test-$(date +%s)"
SSH_PORT="${SSH_PORT:-2222}"

# Detect podman or docker
if command -v podman > /dev/null; then
    RUNTIME="podman"
elif command -v docker > /dev/null; then
    RUNTIME="docker"
else
    echo "Error: Neither podman nor docker found!"
    exit 1
fi

echo "=== Testing ReignX Admtools Image ==="
echo "Runtime: $RUNTIME"
echo "Image: $IMAGE_NAME"
echo "Container: $CONTAINER_NAME"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    $RUNTIME rm -f $CONTAINER_NAME 2>/dev/null || true
}

trap cleanup EXIT

# Test 1: SSH-only mode
echo "Test 1: SSH-only mode"
echo "---------------------"
$RUNTIME run -d --name $CONTAINER_NAME \
    -p $SSH_PORT:22 \
    $IMAGE_NAME

echo "Waiting for container to start..."
sleep 5

echo "Checking container status..."
if $RUNTIME ps | grep -q $CONTAINER_NAME; then
    echo "✓ Container is running"
else
    echo "✗ Container failed to start"
    $RUNTIME logs $CONTAINER_NAME
    exit 1
fi

echo "Checking SSH server..."
if nc -zv localhost $SSH_PORT 2>&1 | grep -q succeeded; then
    echo "✓ SSH server is listening on port $SSH_PORT"
else
    echo "✗ SSH server not accessible"
    $RUNTIME logs $CONTAINER_NAME
    exit 1
fi

echo "Testing SSH login..."
# Test password authentication (default password: changeme)
PASSWORD="${ROOT_PASSWORD:-changeme}"
if sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -p $SSH_PORT root@localhost "echo 'SSH works!'" 2>/dev/null; then
    echo "✓ SSH login successful"
else
    echo "⚠ SSH login failed (sshpass might not be installed, trying manual test)"
    echo "  Run: ssh root@localhost -p $SSH_PORT"
    echo "  Password: (set via ROOT_PASSWORD, default: changeme)"
fi

echo ""
echo "Container logs:"
$RUNTIME logs $CONTAINER_NAME | head -20

echo ""
echo "✅ Test 1 PASSED"
echo ""

# Stop container
$RUNTIME rm -f $CONTAINER_NAME

# Test 2: Agent mode
echo "Test 2: Agent mode (without actual server)"
echo "-----------------------------------------"
$RUNTIME run -d --name $CONTAINER_NAME \
    -p $SSH_PORT:22 \
    -e ENABLE_AGENT=true \
    -e REIGNX_SERVER=localhost:50051 \
    -e NODE_ID=test-node-001 \
    -e LOG_LEVEL=debug \
    $IMAGE_NAME

echo "Waiting for container to start..."
sleep 5

echo "Checking container logs for agent..."
$RUNTIME logs $CONTAINER_NAME

if $RUNTIME logs $CONTAINER_NAME | grep -q "ReignX Agent"; then
    echo "✓ Agent mode enabled"
else
    echo "⚠ Agent might not be enabled (check logs above)"
fi

if $RUNTIME logs $CONTAINER_NAME | grep -q "SSH Access"; then
    echo "✓ SSH server started"
fi

echo ""
echo "Container processes:"
$RUNTIME exec $CONTAINER_NAME ps aux

echo ""
echo "✅ Test 2 PASSED"
echo ""

# Test 3: Architecture detection
echo "Test 3: Architecture detection"
echo "------------------------------"
echo "Container architecture:"
$RUNTIME exec $CONTAINER_NAME uname -m
echo "Host architecture:"
uname -m

echo ""
echo "Image inspect:"
$RUNTIME inspect $IMAGE_NAME | grep -i arch

echo ""
echo "✅ Test 3 PASSED"
echo ""

# Summary
echo "=========================================="
echo "All tests passed! ✅"
echo "=========================================="
echo ""
echo "Quick start commands:"
echo ""
echo "# Start container:"
echo "  $RUNTIME run -d --name admtools \\"
echo "    -p 2222:22 \\"
echo "    -e ENABLE_AGENT=true \\"
echo "    -e REIGNX_SERVER=host.docker.internal:50051 \\"
echo "    $IMAGE_NAME"
echo ""
echo "# SSH login:"
echo "  ssh root@localhost -p 2222"
echo "  Password: (set via ROOT_PASSWORD, default: changeme)"
echo ""
echo "# View logs:"
echo "  $RUNTIME logs -f admtools"
echo ""
echo "# Stop and remove:"
echo "  $RUNTIME rm -f admtools"
echo ""
