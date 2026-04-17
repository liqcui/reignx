#!/bin/bash
# Deploy ReignX to local development environment using Docker Compose

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "Deploying ReignX to local development environment..."

cd "$PROJECT_ROOT"

# Build images first
echo "Building Docker images..."
bash deployments/scripts/build-images.sh

# Initialize PKI if certificates don't exist
if [ ! -d "certs/ca" ]; then
  echo "Initializing PKI..."
  bash scripts/init-pki.sh
fi

# Start services
echo "Starting services with Docker Compose..."
cd deployments/docker-compose
docker-compose -f docker-compose.dev.yaml up -d

# Wait for services to be healthy
echo "Waiting for services to be ready..."
sleep 10

# Check service health
echo "Checking service health..."
docker-compose -f docker-compose.dev.yaml ps

echo ""
echo "✓ ReignX development environment is ready!"
echo ""
echo "Services:"
echo "  - ReignX API:    http://localhost:8080"
echo "  - Prometheus:    http://localhost:9090"
echo "  - Grafana:       http://localhost:3000 (admin/admin)"
echo "  - Jaeger:        http://localhost:16686"
echo "  - PostgreSQL:    localhost:5432"
echo "  - etcd:          localhost:2379"
echo ""
echo "Logs: docker-compose -f deployments/docker-compose/docker-compose.dev.yaml logs -f"
echo "Stop: docker-compose -f deployments/docker-compose/docker-compose.dev.yaml down"
