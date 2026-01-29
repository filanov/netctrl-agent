# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**netctrl-agent** is the client-side component of the netctrl system. It runs on cluster nodes and communicates with **netctrl-server** (https://github.com/filanov/netctrl-server) to register nodes, receive network configurations, and report status.

## System Architecture

### Server-Agent Model

- **netctrl-server**: Central management server exposing gRPC (port 9090) and REST (port 8080) APIs for cluster and agent management
- **netctrl-agent**: Client daemon running on each node that:
  - Registers with the server using a hardware-based UUID
  - Reports node metadata (hostname, IP address, version)
  - Maintains active status through periodic heartbeats
  - Applies network configurations received from the server
  - Unregisters gracefully on shutdown

### gRPC Communication

The agent consumes the `AgentService` from the server, which provides:

- `RegisterAgent`: Register or update agent with cluster ID, hostname, IP, and version
- `GetAgent`: Query agent status by ID
- `ListAgents`: List all agents (optionally filtered by cluster)
- `UnregisterAgent`: Remove agent registration

### Agent State Machine

Agents can be in one of these states (`AgentStatus` enum):
- `AGENT_STATUS_UNSPECIFIED`: Initial/unknown state
- `AGENT_STATUS_ACTIVE`: Agent is running and responsive
- `AGENT_STATUS_INACTIVE`: Agent is not responding

### Protocol Buffers

The API contract is defined in the server repository at `api/proto/v1/agent.proto` and `api/proto/v1/cluster.proto`. The agent needs to import and consume these proto definitions to communicate with the server.

## Development Commands

### Build
```bash
go build ./...

# Build main agent binary
go build -o netctrl-agent cmd/agent/main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a specific test
go test -run TestName ./path/to/package

# Run tests with verbose output
go test -v ./...
```

### Linting
```bash
# Run go vet
go vet ./...

# Format code
go fmt ./...
```

### Dependencies
```bash
# Add a dependency
go get package@version

# Update dependencies
go get -u ./...

# Tidy go.mod
go mod tidy

# Verify dependencies
go mod verify
```

### Protocol Buffers

If proto definitions are vendored or need regeneration:
```bash
# Generate Go code from proto files (requires buf)
buf generate

# Update proto dependencies
buf mod update
```

## Branch Information

This repository uses `master` as the main branch (not `main`).
