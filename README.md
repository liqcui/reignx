# ReignX - Hybrid Infrastructure Management System

ReignX is a powerful, flexible infrastructure management system that supports both SSH-based (agentless) and Agent-based (persistent) management modes, with seamless mode switching capabilities.

## 🎬 Live Demo

**[View Interactive Demo →](https://liqcui.github.io/reignx/)**

Experience ReignX's features through our interactive demonstration page showcasing:
- System architecture and deployment options
- PXE automated OS installation workflow
- Node discovery and agent registration
- Command execution and batch operations
- Real-time monitoring capabilities

## Features

### Dual Mode Architecture
- **SSH Mode**: Agentless management for quick onboarding and restricted environments
- **Agent Mode**: High-performance persistent agents for real-time management
- **Seamless Switching**: Upgrade nodes from SSH to Agent mode with a single command

### Core Capabilities
- **Node Management**: Discover, register, and manage thousands of servers
- **Batch Deployment**: Execute tasks across multiple nodes with configurable concurrency
- **OS Installation**: PXE boot, IPMI/BMC, and cloud API support
- **Firmware Management**: Scan, update, and manage firmware versions
- **OS Reinstallation**: Automated OS provisioning with custom images
- **Remote Execution**: Execute commands across the entire fleet
- **High Availability**: Built-in HA with leader election and automatic failover

### Advanced Features
- **Intelligent Scheduling**: Priority-based task scheduling with resource awareness
- **Real-time Monitoring**: Live status updates and metrics collection
- **Audit Trail**: Complete audit logging for compliance
- **RBAC**: Role-based access control with fine-grained permissions
- **API-First**: RESTful API for automation and integration

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         ReignX CLI                           │
│              (reignxctl - Command Line Tool)                 │
└──────────────────────┬──────────────────────────────────────┘
                       │ REST API
┌──────────────────────┴──────────────────────────────────────┐
│                      ReignX API Layer                         │
│                    (reignx-api service)                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────────────┐
│                   ReignX Control Plane                        │
│                    (reignxd daemon)                           │
│  ┌────────────┬──────────────┬────────────┬──────────────┐  │
│  │ API Server │  Scheduler   │ SSH Engine │ Agent Manager│  │
│  └────────────┴──────────────┴────────────┴──────────────┘  │
└──────┬────────────────────────┬──────────────────────┬──────┘
       │                        │                      │
       │ PostgreSQL        NATS/etcd            gRPC over mTLS
       │                        │                      │
┌──────┴────────────────────────┴──────────────────────┴──────┐
│                      Managed Nodes                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ SSH Mode    │  │ Agent Mode  │  │ Hybrid Mode │          │
│  │ (Agentless) │  │ (Persistent)│  │ (Both)      │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└──────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Docker (for PostgreSQL)
- Go 1.22+ (Go 1.25 has known issues on macOS)
- PostgreSQL client (psql)

**Security Setup (Recommended):**
```bash
# Install pre-commit hook to prevent password commits
cp hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### Installation

**Option 1: Quick Development Setup**

```bash
cd /Users/liqcui/goproject/github.com/liqcui/reignx
./scripts/dev-start.sh
```

This automated script will:
- Start PostgreSQL container
- Run database migrations
- Start ReignX daemon
- Verify system health

**Option 2: Manual Setup**

```bash
# Clone repository
git clone https://github.com/liqcui/reignx.git
cd reignx

# Start PostgreSQL
docker run -d --name reignx-postgres \
  -e POSTGRES_DB=reignx \
  -e POSTGRES_USER=reignx \
  -e POSTGRES_PASSWORD=reignx \
  -p 5432:5432 \
  postgres:16

# Run database migrations
export PGPASSWORD=reignx
psql -h localhost -U reignx -d reignx -f migrations/000001_initial_schema.up.sql

# Start ReignX daemon
go run ./reignxd/cmd/main.go
```

### Verify Installation

```bash
# Check health
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","time":"2026-04-15T10:30:00Z"}
```

### Add Your First Node

```bash
# Create node via API
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "web-01",
    "ip_address": "192.168.1.10",
    "mode": "ssh",
    "os_type": "linux",
    "tags": {"env": "production"},
    "ssh_config": {
      "host": "192.168.1.10",
      "port": 22,
      "user": "root",
      "key_path": "/root/.ssh/id_rsa"
    }
  }'

# List nodes
curl http://localhost:8080/api/v1/nodes
```

### Create and Execute a Job

```bash
# Create a batch job
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "System Health Check",
    "type": "command",
    "mode": "ssh",
    "filter": {"status": "online"},
    "template": {
      "type": "command",
      "command": "uptime",
      "timeout": 30000000000,
      "priority": 1,
      "max_retries": 3
    }
  }'

# Check job status (replace <job-id> with actual ID)
curl http://localhost:8080/api/v1/jobs/<job-id>
```

## CLI Commands

> **Note:** CLI tool (reignxctl) is built but not yet connected to the API. Currently uses mock data for demonstration.

### Node Management
```bash
./bin/reignxctl node list                              # List all nodes
./bin/reignxctl node add <ip> --user <user>            # Add node via SSH
./bin/reignxctl node show <node>                       # Show node details
./bin/reignxctl node delete <node>                     # Remove node
./bin/reignxctl node tag <node> env=prod region=us     # Tag node
./bin/reignxctl node status                            # Show cluster status
```

### Deployment
```bash
./bin/reignxctl deploy --mode=ssh --batch=50           # SSH mode batch deploy
./bin/reignxctl deploy --mode=agent                    # Agent mode deploy
./bin/reignxctl deploy --targets="env=prod" --parallel # Parallel deployment
```

### Mode Switching
```bash
./bin/reignxctl switch <node> ssh agent                # Upgrade to Agent mode
./bin/reignxctl switch <node> agent ssh                # Downgrade to SSH mode
./bin/reignxctl switch --all --mode=agent              # Upgrade all nodes
```

### Remote Execution
```bash
./bin/reignxctl exec --all "uptime"                    # Execute on all nodes
./bin/reignxctl exec --targets="os=linux" "df -h"      # Execute on filtered nodes
./bin/reignxctl exec --node node1 "systemctl status"   # Execute on specific node
```

### Firmware Management
```bash
./bin/reignxctl firmware scan                          # Scan firmware versions
./bin/reignxctl firmware list                          # List firmware inventory
./bin/reignxctl firmware update --version=2.8          # Update firmware
./bin/reignxctl firmware rollback <node>               # Rollback firmware
```

### OS Management
```bash
./bin/reignxctl os reinstall --node node1 --image=rocky9   # Reinstall OS
./bin/reignxctl os list-images                              # List available images
./bin/reignxctl os upload --image ubuntu-22.04.iso          # Upload OS image
```

### Build CLI Tool

```bash
cd /Users/liqcui/goproject/github.com/liqcui/reignx
go build -o bin/reignxctl ./reignxctl
./bin/reignxctl version
```

## Configuration

### Control Plane (reignxd.yaml)
```yaml
server:
  listen: 0.0.0.0:8080
  grpc_port: 50051

database:
  host: localhost
  port: 5432
  database: reignx
  user: reignx
  password: changeme

nats:
  url: nats://localhost:4222

etcd:
  endpoints:
    - localhost:2379

ssh:
  max_connections: 100
  timeout: 30s
  batch_size: 50

agent:
  heartbeat_interval: 30s
  task_timeout: 3600s
```

### Agent (reignx-agent.yaml)
```yaml
server:
  address: reignx.example.com:50051

security:
  tls_enabled: true
  cert_file: /etc/reignx/agent.crt
  key_file: /etc/reignx/agent.key
  ca_file: /etc/reignx/ca.crt

task:
  concurrency: 5
  cache_dir: /var/lib/reignx-agent
```

## Package Structure

```
github.com/reignx/reignx/
├── pkg/
│   ├── core/           # Core interfaces and types
│   ├── sshmode/        # SSH mode implementation
│   ├── agentmode/      # Agent mode implementation
│   ├── database/       # Database access layer
│   ├── config/         # Configuration management
│   ├── api/            # API models and handlers
│   ├── firmware/       # Firmware management
│   ├── osimage/        # OS image management
│   ├── logger/         # Structured logging
│   └── crypto/         # Cryptography utilities
├── reignxctl/          # CLI tool
├── reignxd/            # Control plane daemon
├── reignx-agent/       # Node agent
├── reignx-api/         # API server
└── reignx-web/         # Web console
```

## Import Usage

```go
import "github.com/reignx/reignx/pkg/core"
import "github.com/reignx/reignx/pkg/sshmode"
import "github.com/reignx/reignx/pkg/agentmode"

// Use SSH mode executor
sshExec := sshmode.NewExecutor(config)
result, err := sshExec.Execute(ctx, "192.168.1.10", "uptime")

// Use Agent mode executor
agentExec := agentmode.NewExecutor(config)
result, err := agentExec.Execute(ctx, "node1", task)
```

## Development

```bash
# Setup development environment
make setup-dev

# Run tests
make test

# Run linters
make lint

# Build all binaries
make build

# Run locally
make run-dev
```

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22+ |
| Database | PostgreSQL 16 |
| Message Queue | NATS JetStream |
| Service Discovery | etcd |
| API Protocol | REST + gRPC |
| Security | mTLS + JWT |
| Web Framework | Gin |
| CLI Framework | Cobra |

## Deployment

### Docker Compose (Development)
```bash
docker-compose up -d
```

### Kubernetes (Production)
```bash
helm install reignx ./deployments/kubernetes/helm
```

### Bare Metal
```bash
# Install binaries
sudo make install

# Configure systemd services
sudo systemctl enable reignxd
sudo systemctl start reignxd
```

## Current Status

**Version:** 0.1.0-alpha
**Status:** Core System Complete ✅

### What's Working Now

- ✅ **Complete Control Plane** - HTTP API, gRPC server, job scheduler
- ✅ **Database Layer** - PostgreSQL with 12 tables, repositories, migrations
- ✅ **SSH Mode** - Production-ready agentless execution
- ✅ **Agent Mode Framework** - gRPC client/server (needs proto generation)
- ✅ **CLI Tool** - Full-featured reignxctl with all commands
- ✅ **Job Scheduling** - Automatic task distribution and execution
- ✅ **Documentation** - Quick reference guides and API documentation

### What's Pending

- 🔧 **Agent Binary** - Needs Protocol Buffer code generation
- 🔧 **CLI to API** - Needs HTTP client integration
- 🔧 **Authentication** - JWT and RBAC implementation
- 🔧 **Mode Switching** - Implementation of mode transition logic
- 🔧 **Web Console** - Frontend development
- 🔧 **Monitoring** - Prometheus metrics and Grafana dashboards

### Known Issues

1. **Go 1.25 on macOS**: Built binaries have execution issues. Use `go run` or Go 1.22 instead.
2. **Proto Generation**: Agent mode requires `protoc` compilation before it can be used.

## Documentation

- **[Quick Reference](QUICK_REFERENCE.md)** - Fast command reference and common tasks
- **[Final Implementation Summary](FINAL_IMPLEMENTATION_SUMMARY.md)** - Complete system overview
- **[Project Status](PROJECT_STATUS.md)** - Detailed status and next steps

## Testing

Run the complete integration test suite:

```bash
./scripts/integration-test.sh
```

This will verify:
- Database connectivity
- API endpoints
- Job creation and execution
- Node management
- Database consistency

## Project Location

```
/Users/liqcui/goproject/github.com/liqcui/reignx/
```

## Support

For issues, questions, or contributions, please refer to:
- **Project Status**: [PROJECT_STATUS.md](PROJECT_STATUS.md)
- **Quick Reference**: [QUICK_REFERENCE.md](QUICK_REFERENCE.md)

## License

Copyright 2026 ReignX Project
