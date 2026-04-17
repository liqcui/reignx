# ReignX Agent

The ReignX Agent is a lightweight, persistent daemon that runs on managed servers to enable real-time task execution and monitoring.

## Features

- **Persistent Connection**: Maintains gRPC connection to control plane
- **Task Execution**: Executes commands, scripts, and file operations
- **Metrics Collection**: Collects and reports system metrics
- **Idempotency**: Caches task results to prevent duplicate execution
- **Concurrency Control**: Configurable concurrent task execution
- **Auto-Reconnect**: Automatic reconnection with exponential backoff
- **Local Cache**: SQLite cache for task state persistence
- **mTLS Support**: Secure communication with mutual TLS

## Installation

### From Binary

```bash
# Download latest release
curl -L https://github.com/reignx/reignx/releases/latest/download/reignx-agent-linux-amd64 -o reignx-agent

# Make executable
chmod +x reignx-agent

# Move to system path
sudo mv reignx-agent /usr/local/bin/

# Install as systemd service
sudo reignx-agent --install
```

### From Source

```bash
# Build
cd /Users/liqcui/goproject/github.com/liqcui/reignx/reignx-agent
go build -o bin/reignx-agent ./cmd

# Install
sudo cp bin/reignx-agent /usr/local/bin/
sudo reignx-agent --install
```

## Configuration

### Configuration File

Copy the example configuration:

```bash
sudo mkdir -p /etc/reignx
sudo cp agent.yaml.example /etc/reignx/agent.yaml
sudo vi /etc/reignx/agent.yaml
```

### Command-Line Options

```bash
reignx-agent [options]

Options:
  --config string         Path to configuration file (default "/etc/reignx/agent.yaml")
  --server string         ReignX server address (default "localhost:50051")
  --tls                   Enable TLS
  --cert string           Client certificate file (default "/etc/reignx/agent.crt")
  --key string            Client key file (default "/etc/reignx/agent.key")
  --ca string             CA certificate file (default "/etc/reignx/ca.crt")
  --heartbeat int         Heartbeat interval in seconds (default 30)
  --log-level string      Log level: debug, info, warn, error (default "info")
  --cache-dir string      Cache directory (default "/var/lib/reignx-agent")
  --concurrency int       Maximum concurrent tasks (default 5)
  --version               Show version information
  --install               Install agent as systemd service
  --uninstall             Uninstall agent systemd service
```

## Usage

### Start Agent

```bash
# Start directly
sudo reignx-agent

# Start with custom server
sudo reignx-agent --server reignx.example.com:50051 --tls

# Start with debug logging
sudo reignx-agent --log-level debug
```

### Systemd Service

```bash
# Install service
sudo reignx-agent --install

# Enable and start
sudo systemctl enable reignx-agent
sudo systemctl start reignx-agent

# Check status
sudo systemctl status reignx-agent

# View logs
sudo journalctl -u reignx-agent -f

# Stop
sudo systemctl stop reignx-agent

# Uninstall
sudo reignx-agent --uninstall
```

## Architecture

```
┌─────────────────────────────────────┐
│         ReignX Agent                │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  Main Loop                    │ │
│  │  - Connection management      │ │
│  │  - Registration               │ │
│  │  - Graceful shutdown          │ │
│  └───────────────────────────────┘ │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  Heartbeat Loop               │ │
│  │  - Send periodic heartbeats   │ │
│  │  - Report basic metrics       │ │
│  │  - Auto-reconnect on failure  │ │
│  └───────────────────────────────┘ │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  Task Loop                    │ │
│  │  - Receive tasks via stream   │ │
│  │  - Execute asynchronously     │ │
│  │  - Send results back          │ │
│  └───────────────────────────────┘ │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  Metrics Loop                 │ │
│  │  - Collect system metrics     │ │
│  │  - Report periodically        │ │
│  └───────────────────────────────┘ │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  Task Executor                │ │
│  │  - Command execution          │ │
│  │  - Script execution           │ │
│  │  - File operations            │ │
│  │  - Concurrency control        │ │
│  │  - Idempotency checking       │ │
│  └───────────────────────────────┘ │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  Local Cache (SQLite)         │ │
│  │  - Completed tasks            │ │
│  │  - Pending tasks              │ │
│  │  - Configuration              │ │
│  └───────────────────────────────┘ │
└─────────────────────────────────────┘
           │ gRPC over mTLS
           ▼
┌─────────────────────────────────────┐
│      ReignX Control Plane           │
│      (gRPC Server)                  │
└─────────────────────────────────────┘
```

## Task Execution

### Supported Task Types

#### 1. Command Execution

```json
{
  "type": "command",
  "command": "uptime",
  "timeout": 30000000000
}
```

#### 2. Script Execution

```json
{
  "type": "script",
  "script": "#!/bin/bash\napt-get update\napt-get upgrade -y",
  "timeout": 600000000000
}
```

#### 3. File Operations

```json
{
  "type": "file",
  "parameters": {
    "operation": "copy",
    "source": "/tmp/file.txt",
    "destination": "/etc/config/file.txt"
  }
}
```

Supported file operations:
- `copy` - Copy a file
- `move` - Move a file
- `delete` - Delete a file
- `mkdir` - Create a directory

### Task Idempotency

The agent caches successfully completed tasks based on a fingerprint (hash of task type and content). If the same task is received again, it will be skipped and the cached result returned.

This ensures idempotent behavior - running the same task multiple times has the same effect as running it once.

### Concurrency Control

Tasks are executed with configurable concurrency (default: 5). If all execution slots are busy, new tasks wait in a queue.

## Metrics Collection

The agent collects and reports the following metrics:

### System Metrics
- **CPU**: Utilization percentage, core count
- **Memory**: Usage percentage, used/total GB
- **Disk**: Usage percentage, used/total GB (root partition)
- **Network**: RX/TX throughput in MB/s
- **Uptime**: System uptime in seconds

### Process Metrics
- **Goroutines**: Number of active goroutines
- **Running Tasks**: Number of currently executing tasks

### Reporting

- Metrics are included in heartbeats (every 30 seconds by default)
- Detailed metrics reports sent every 60 seconds
- Health status calculated based on resource thresholds

## Local Cache

The agent maintains a local SQLite database for:

1. **Completed Tasks**: Fingerprints of successfully executed tasks (30-day TTL)
2. **Pending Tasks**: Tasks queued for execution (survives restarts)
3. **Configuration**: Agent-specific configuration values

Cache location: `/var/lib/reignx-agent/agent.db`

### Cache Management

```bash
# View cache stats
sqlite3 /var/lib/reignx-agent/agent.db "SELECT COUNT(*) FROM completed_tasks"

# Clear cache
sqlite3 /var/lib/reignx-agent/agent.db "DELETE FROM completed_tasks"

# Vacuum database
sqlite3 /var/lib/reignx-agent/agent.db "VACUUM"
```

## Security

### mTLS

The agent supports mutual TLS authentication:

1. **Client Certificate**: Authenticates agent to server
2. **CA Certificate**: Validates server identity
3. **TLS 1.2+**: Minimum TLS version

### Certificate Generation

```bash
# Generate client certificate (placeholder - actual process TBD)
openssl req -new -newkey rsa:4096 -days 365 -nodes \
  -x509 -keyout agent.key -out agent.crt
```

## Troubleshooting

### Agent Won't Start

```bash
# Check configuration
reignx-agent --config /etc/reignx/agent.yaml

# Test connection
telnet reignx.example.com 50051

# Check logs
journalctl -u reignx-agent -n 100
```

### Connection Issues

```bash
# Verify server address
ping reignx.example.com

# Check TLS certificates
openssl verify -CAfile /etc/reignx/ca.crt /etc/reignx/agent.crt

# Disable TLS for testing
reignx-agent --server localhost:50051
```

### High Resource Usage

```bash
# Check running tasks
# (Check cache stats to see active tasks)

# Reduce concurrency
reignx-agent --concurrency 2

# Check metrics
# (Metrics show CPU/memory usage)
```

### Cache Issues

```bash
# Clear cache
sudo rm -rf /var/lib/reignx-agent/*

# Rebuild cache
sudo reignx-agent
```

## Performance

### Resource Usage

Typical resource usage:
- **Memory**: 20-50MB
- **CPU**: <1% idle, 5-20% during task execution
- **Disk**: 10-100MB for cache
- **Network**: <100KB/s (heartbeats and metrics)

### Scalability

The agent is designed for:
- **Concurrent Tasks**: Up to 10 concurrent tasks (configurable)
- **Task Queue**: Unlimited pending tasks
- **Cache Size**: 10,000+ completed tasks

## Development

### Build from Source

```bash
cd reignx-agent

# Build
go build -o bin/reignx-agent ./cmd

# Run locally
./bin/reignx-agent --server localhost:50051
```

### Dependencies

```bash
go get github.com/shirou/gopsutil/v3
go get github.com/mattn/go-sqlite3
go get go.uber.org/zap
go get google.golang.org/grpc
```

### Testing

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o bin/reignx-agent-linux-amd64 ./cmd
GOOS=darwin GOARCH=amd64 go build -o bin/reignx-agent-darwin-amd64 ./cmd
GOOS=windows GOARCH=amd64 go build -o bin/reignx-agent-windows-amd64.exe ./cmd
```

## Version History

- **0.1.0** (2026-04-15)
  - Initial implementation
  - Core agent loop
  - Task execution engine
  - Metrics collection
  - Local cache
  - mTLS support

## License

Copyright 2026 ReignX Project

## Related Documentation

- [ReignX Main Documentation](../README.md)
- [gRPC Protocol Definition](../api/proto/agent.proto)
- [Control Plane Daemon](../reignxd/README.md)
