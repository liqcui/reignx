#!/bin/bash

# ReignX Integration Test Script
# Tests the gRPC communication between control plane and agent

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== ReignX Integration Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if binaries exist
if [ ! -f "$PROJECT_DIR/bin/reignxd" ]; then
    echo -e "${RED}ERROR: reignxd binary not found${NC}"
    echo "Run: go build -o bin/reignxd ./reignxd/cmd"
    exit 1
fi

if [ ! -f "$PROJECT_DIR/bin/reignx-agent" ]; then
    echo -e "${RED}ERROR: reignx-agent binary not found${NC}"
    echo "Run: go build -o bin/reignx-agent ./reignx-agent/cmd"
    exit 1
fi

echo -e "${GREEN}✓ Binaries found${NC}"

# Check if PostgreSQL is running
if ! pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo -e "${RED}ERROR: PostgreSQL is not running${NC}"
    echo "Start with: brew services start postgresql@16"
    exit 1
fi

echo -e "${GREEN}✓ PostgreSQL is running${NC}"

# Check if database exists
if ! psql -h localhost -U reignx -d reignx -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${YELLOW}Database not set up. Running setup script...${NC}"
    bash "$SCRIPT_DIR/setup-db.sh"
fi

echo -e "${GREEN}✓ Database ready${NC}"
echo ""

# Kill any existing instances
echo "Cleaning up any existing processes..."
pkill -f "bin/reignxd" 2>/dev/null || true
pkill -f "bin/reignx-agent" 2>/dev/null || true
sleep 2

# Create log directory
mkdir -p "$PROJECT_DIR/logs"

# Start control plane
echo "Starting control plane (reignxd)..."
"$PROJECT_DIR/bin/reignxd" --http-port 8080 --grpc-port 50051 \
    > "$PROJECT_DIR/logs/reignxd.log" 2>&1 &
REIGNXD_PID=$!

# Wait for control plane to start
echo "Waiting for control plane to start..."
sleep 3

# Check if control plane is running
if ! ps -p $REIGNXD_PID > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Control plane failed to start${NC}"
    echo "Check logs at: $PROJECT_DIR/logs/reignxd.log"
    cat "$PROJECT_DIR/logs/reignxd.log"
    exit 1
fi

echo -e "${GREEN}✓ Control plane started (PID: $REIGNXD_PID)${NC}"

# Check if gRPC port is listening
if lsof -i:50051 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ gRPC server listening on port 50051${NC}"
else
    echo -e "${YELLOW}⚠ gRPC port 50051 not yet listening${NC}"
fi

# Check if HTTP port is listening
if lsof -i:8080 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ HTTP server listening on port 8080${NC}"
else
    echo -e "${YELLOW}⚠ HTTP port 8080 not yet listening${NC}"
fi

echo ""

# Start agent
echo "Starting agent (reignx-agent)..."
"$PROJECT_DIR/bin/reignx-agent" --server localhost:50051 \
    > "$PROJECT_DIR/logs/reignx-agent.log" 2>&1 &
AGENT_PID=$!

# Wait for agent to start
echo "Waiting for agent to register..."
sleep 5

# Check if agent is running
if ! ps -p $AGENT_PID > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Agent failed to start${NC}"
    echo "Check logs at: $PROJECT_DIR/logs/reignx-agent.log"
    cat "$PROJECT_DIR/logs/reignx-agent.log"

    echo ""
    echo "Control plane logs:"
    cat "$PROJECT_DIR/logs/reignxd.log"

    # Cleanup
    kill $REIGNXD_PID 2>/dev/null || true
    exit 1
fi

echo -e "${GREEN}✓ Agent started (PID: $AGENT_PID)${NC}"
echo ""

# Show last 10 lines of logs
echo "=== Control Plane Logs (last 10 lines) ==="
tail -10 "$PROJECT_DIR/logs/reignxd.log"
echo ""

echo "=== Agent Logs (last 10 lines) ==="
tail -10 "$PROJECT_DIR/logs/reignx-agent.log"
echo ""

# Check database for registered agent
echo "Checking database for registered agent..."
AGENT_COUNT=$(psql -h localhost -U reignx -d reignx -tAc "SELECT COUNT(*) FROM nodes WHERE mode='agent';")

if [ "$AGENT_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Found $AGENT_COUNT registered agent(s) in database${NC}"
    echo ""
    echo "Agent details:"
    psql -h localhost -U reignx -d reignx -c "SELECT id, hostname, ip_address, status, mode, agent_version, last_seen FROM nodes WHERE mode='agent';"
else
    echo -e "${YELLOW}⚠ No agents found in database yet${NC}"
    echo "The agent may still be registering. Check logs for details."
fi

echo ""
echo "=== Integration Test Complete ==="
echo ""
echo "Processes running:"
echo "  Control Plane: PID $REIGNXD_PID (logs: logs/reignxd.log)"
echo "  Agent:         PID $AGENT_PID (logs: logs/reignx-agent.log)"
echo ""
echo "To monitor logs in real-time:"
echo "  tail -f logs/reignxd.log"
echo "  tail -f logs/reignx-agent.log"
echo ""
echo "To stop the services:"
echo "  kill $REIGNXD_PID $AGENT_PID"
echo ""
echo "Press Ctrl+C to stop monitoring and exit"
echo ""

# Monitor logs
tail -f "$PROJECT_DIR/logs/reignxd.log" "$PROJECT_DIR/logs/reignx-agent.log"
