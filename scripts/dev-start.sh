#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "ReignX Development Environment Setup"
echo "=========================================="
echo ""

# Step 1: Start PostgreSQL
echo -e "${YELLOW}→${NC} Starting PostgreSQL..."
if docker ps | grep -q reignx-postgres; then
    echo -e "${GREEN}✓${NC} PostgreSQL already running"
else
    docker run -d \
        --name reignx-postgres \
        -e POSTGRES_DB=reignx \
        -e POSTGRES_USER=reignx \
        -e POSTGRES_PASSWORD=reignx \
        -p 5432:5432 \
        postgres:16 > /dev/null 2>&1

    echo -e "${YELLOW}→${NC} Waiting for PostgreSQL to be ready..."
    sleep 5

    for i in {1..30}; do
        if PGPASSWORD=reignx psql -h localhost -U reignx -d reignx -c "SELECT 1" &> /dev/null; then
            break
        fi
        sleep 1
    done

    echo -e "${GREEN}✓${NC} PostgreSQL started"
fi

# Step 2: Run migrations
echo -e "${YELLOW}→${NC} Checking database schema..."
export PGPASSWORD=reignx

if psql -h localhost -U reignx -d reignx -c "SELECT COUNT(*) FROM nodes" &> /dev/null; then
    echo -e "${GREEN}✓${NC} Database schema exists"
else
    echo -e "${YELLOW}→${NC} Running migrations..."
    cd /Users/liqcui/goproject/github.com/liqcui/reignx
    psql -h localhost -U reignx -d reignx -f migrations/000001_initial_schema.up.sql > /dev/null 2>&1
    echo -e "${GREEN}✓${NC} Database migrations applied"
fi

# Step 3: Start ReignX daemon
echo -e "${YELLOW}→${NC} Starting ReignX daemon..."
cd /Users/liqcui/goproject/github.com/liqcui/reignx

# Kill existing daemon if running
pkill -f "reignxd/cmd/main.go" 2>/dev/null || true
sleep 1

# Start daemon
go run ./reignxd/cmd/main.go > /tmp/reignxd.log 2>&1 &
DAEMON_PID=$!

# Wait for daemon to start
sleep 3

if ps -p $DAEMON_PID > /dev/null; then
    echo -e "${GREEN}✓${NC} ReignX daemon started (PID: $DAEMON_PID)"
else
    echo -e "${RED}✗${NC} Failed to start daemon"
    cat /tmp/reignxd.log
    exit 1
fi

# Test health endpoint
sleep 2
if curl -s http://localhost:8080/health | grep -q "healthy"; then
    echo -e "${GREEN}✓${NC} API server is healthy"
else
    echo -e "${RED}✗${NC} API server health check failed"
    exit 1
fi

echo ""
echo "=========================================="
echo -e "${GREEN}ReignX is ready!${NC}"
echo "=========================================="
echo ""
echo "Services:"
echo "  - PostgreSQL: localhost:5432"
echo "  - HTTP API:   http://localhost:8080"
echo "  - gRPC:       localhost:50051"
echo ""
echo "Quick commands:"
echo "  - Health check: curl http://localhost:8080/health"
echo "  - List nodes:   curl http://localhost:8080/api/v1/nodes"
echo "  - View logs:    tail -f /tmp/reignxd.log"
echo ""
echo "To stop:"
echo "  - Daemon: kill $DAEMON_PID"
echo "  - PostgreSQL: docker stop reignx-postgres"
echo ""
