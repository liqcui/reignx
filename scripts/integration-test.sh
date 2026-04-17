#!/bin/bash
set -e

echo "=========================================="
echo "ReignX Integration Test Suite"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
REIGNX_DIR="/Users/liqcui/goproject/github.com/liqcui/reignx"
API_URL="http://localhost:8080"
POSTGRES_READY=false

# Function to print test status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        exit 1
    fi
}

print_info() {
    echo -e "${YELLOW}→${NC} $1"
}

# Step 1: Check prerequisites
echo "Step 1: Checking prerequisites..."
print_info "Checking Docker..."
if command -v docker &> /dev/null; then
    print_status 0 "Docker is installed"
else
    print_status 1 "Docker is not installed"
fi

print_info "Checking Go..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_status 0 "Go is installed ($GO_VERSION)"
else
    print_status 1 "Go is not installed"
fi

print_info "Checking PostgreSQL client..."
if command -v psql &> /dev/null; then
    print_status 0 "PostgreSQL client is installed"
else
    print_status 1 "PostgreSQL client is not installed"
fi

echo ""

# Step 2: Start PostgreSQL if not running
echo "Step 2: Starting PostgreSQL database..."
if docker ps | grep -q reignx-postgres; then
    print_status 0 "PostgreSQL container already running"
    POSTGRES_READY=true
else
    print_info "Starting PostgreSQL container..."
    docker run -d \
        --name reignx-postgres \
        -e POSTGRES_DB=reignx \
        -e POSTGRES_USER=reignx \
        -e POSTGRES_PASSWORD=reignx \
        -p 5432:5432 \
        postgres:16 2>&1 | head -1

    print_info "Waiting for PostgreSQL to be ready..."
    sleep 5

    for i in {1..30}; do
        if PGPASSWORD=reignx psql -h localhost -U reignx -d reignx -c "SELECT 1" &> /dev/null; then
            POSTGRES_READY=true
            break
        fi
        sleep 1
    done

    if [ "$POSTGRES_READY" = true ]; then
        print_status 0 "PostgreSQL is ready"
    else
        print_status 1 "PostgreSQL failed to start"
    fi
fi

echo ""

# Step 3: Run database migrations
echo "Step 3: Running database migrations..."
cd "$REIGNX_DIR"
export PGPASSWORD=reignx

print_info "Checking if migrations already applied..."
if psql -h localhost -U reignx -d reignx -c "SELECT COUNT(*) FROM nodes" &> /dev/null; then
    print_status 0 "Database schema already exists"
else
    print_info "Applying migrations..."
    psql -h localhost -U reignx -d reignx -f migrations/000001_initial_schema.up.sql > /dev/null 2>&1
    print_status $? "Database migrations applied"
fi

echo ""

# Step 4: Build ReignX daemon
echo "Step 4: Building ReignX daemon..."
cd "$REIGNX_DIR"
print_info "Building reignxd..."
go build -o bin/reignxd ./reignxd/cmd 2>&1 | grep -v "missing LC_UUID" || true
if [ -f bin/reignxd ]; then
    print_status 0 "reignxd binary built successfully"
else
    print_status 1 "Failed to build reignxd"
fi

echo ""

# Step 5: Build CLI tool
echo "Step 5: Building CLI tool..."
print_info "Building reignxctl..."
go build -o bin/reignxctl ./reignxctl 2>&1 | grep -v "missing LC_UUID" || true
if [ -f bin/reignxctl ]; then
    print_status 0 "reignxctl binary built successfully"
else
    print_status 1 "Failed to build reignxctl"
fi

echo ""

# Step 6: Start daemon in background
echo "Step 6: Starting ReignX daemon..."
print_info "Killing any existing reignxd process..."
pkill -f reignxd || true
sleep 2

print_info "Starting reignxd daemon..."
cd "$REIGNX_DIR"
go run ./reignxd/cmd/main.go > /tmp/reignxd.log 2>&1 &
DAEMON_PID=$!

print_info "Waiting for daemon to start..."
sleep 5

if ps -p $DAEMON_PID > /dev/null; then
    print_status 0 "reignxd daemon started (PID: $DAEMON_PID)"
else
    print_status 1 "reignxd daemon failed to start"
    cat /tmp/reignxd.log
    exit 1
fi

echo ""

# Step 7: Test API endpoints
echo "Step 7: Testing API endpoints..."

print_info "Testing health endpoint..."
HEALTH=$(curl -s $API_URL/health)
if echo "$HEALTH" | grep -q "healthy"; then
    print_status 0 "Health check passed"
else
    print_status 1 "Health check failed"
fi

print_info "Creating test node..."
CREATE_RESPONSE=$(curl -s -X POST $API_URL/api/v1/nodes \
    -H "Content-Type: application/json" \
    -d '{
        "hostname": "test-server-01",
        "ip_address": "192.168.1.100",
        "mode": "ssh",
        "os_type": "linux",
        "tags": {"env": "test", "region": "local"},
        "ssh_config": {
            "host": "192.168.1.100",
            "port": 22,
            "user": "root",
            "key_path": "/root/.ssh/id_rsa"
        }
    }')

NODE_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$NODE_ID" ]; then
    print_status 0 "Test node created (ID: $NODE_ID)"
else
    print_status 1 "Failed to create test node"
    echo "$CREATE_RESPONSE"
fi

print_info "Listing nodes..."
LIST_RESPONSE=$(curl -s $API_URL/api/v1/nodes)
if echo "$LIST_RESPONSE" | grep -q "test-server-01"; then
    print_status 0 "Node list retrieved successfully"
else
    print_status 1 "Failed to list nodes"
fi

print_info "Creating test job..."
JOB_RESPONSE=$(curl -s -X POST $API_URL/api/v1/jobs \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test System Check",
        "type": "command",
        "mode": "ssh",
        "filter": {"status": "offline"},
        "template": {
            "type": "command",
            "command": "uptime",
            "timeout": 30000000000,
            "priority": 1,
            "max_retries": 3
        },
        "batch_size": 10,
        "concurrency": 10
    }')

JOB_ID=$(echo "$JOB_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$JOB_ID" ]; then
    print_status 0 "Test job created (ID: $JOB_ID)"
else
    print_status 1 "Failed to create test job"
fi

print_info "Testing statistics endpoint..."
STATS_RESPONSE=$(curl -s $API_URL/api/v1/stats)
if echo "$STATS_RESPONSE" | grep -q "nodes"; then
    print_status 0 "Statistics endpoint working"
else
    print_status 1 "Statistics endpoint failed"
fi

echo ""

# Step 8: Test database directly
echo "Step 8: Testing database consistency..."

print_info "Counting nodes in database..."
NODE_COUNT=$(PGPASSWORD=reignx psql -h localhost -U reignx -d reignx -t -c "SELECT COUNT(*) FROM nodes")
if [ "$NODE_COUNT" -gt 0 ]; then
    print_status 0 "Found $NODE_COUNT node(s) in database"
else
    print_status 1 "No nodes found in database"
fi

print_info "Counting jobs in database..."
JOB_COUNT=$(PGPASSWORD=reignx psql -h localhost -U reignx -d reignx -t -c "SELECT COUNT(*) FROM jobs")
if [ "$JOB_COUNT" -gt 0 ]; then
    print_status 0 "Found $JOB_COUNT job(s) in database"
else
    print_status 1 "No jobs found in database"
fi

print_info "Checking audit logs..."
AUDIT_COUNT=$(PGPASSWORD=reignx psql -h localhost -U reignx -d reignx -t -c "SELECT COUNT(*) FROM audit_logs")
print_status 0 "Found $AUDIT_COUNT audit log entries"

echo ""

# Step 9: Cleanup
echo "Step 9: Cleanup..."

print_info "Stopping daemon..."
kill $DAEMON_PID 2>/dev/null || true
sleep 2
print_status 0 "Daemon stopped"

echo ""
echo "=========================================="
echo -e "${GREEN}All tests passed successfully!${NC}"
echo "=========================================="
echo ""
echo "Summary:"
echo "  - PostgreSQL: Running"
echo "  - Database schema: Applied"
echo "  - ReignX daemon: Built and tested"
echo "  - CLI tool: Built"
echo "  - API endpoints: Working"
echo "  - Nodes created: $NODE_COUNT"
echo "  - Jobs created: $JOB_COUNT"
echo "  - Audit logs: $AUDIT_COUNT entries"
echo ""
echo "To start using ReignX:"
echo "  1. Start daemon: go run ./reignxd/cmd/main.go"
echo "  2. Use CLI: ./bin/reignxctl node list"
echo "  3. Access API: curl http://localhost:8080/health"
echo ""
echo "Logs available at: /tmp/reignxd.log"
echo ""
