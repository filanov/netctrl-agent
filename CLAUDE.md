# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**netctrl-agent** is the client-side component of the netctrl system. It runs on cluster nodes and communicates with **netctrl-server** (https://github.com/filanov/netctrl-server) to register nodes, receive network configurations, and report status.

## System Architecture

### Server-Agent Model

- **netctrl-server**: Central management server exposing gRPC (port 9090) and REST (port 8080) APIs for cluster and agent management
- **netctrl-agent**: Client daemon running on each node that:
  - Registers with the server using a deterministic UUID (SHA256 of hostname:ip)
  - Reports node metadata (hostname, IP address, version)
  - Supports two modes:
    - **One-shot mode** (default): Register once and exit
    - **Daemon mode** (`--daemon`): Continuous polling for instructions
  - Maintains active status through periodic GetInstructions calls (heartbeat)
  - Processes instructions from server (POLL_INTERVAL, HEALTH_CHECK)
  - Applies network configurations received from the server
  - Unregisters gracefully on shutdown (daemon mode)

### gRPC Communication

The agent consumes the `AgentService` from the server, which provides:

- `RegisterAgent`: Register or update agent with cluster ID, hostname, IP, and version
- `GetInstructions`: Poll for pending instructions and send results (also acts as heartbeat)
- `UnregisterAgent`: Remove agent registration
- `GetAgent`: Query agent status by ID (server-to-server, not used by agent)
- `ListAgents`: List all agents (server-to-server, not used by agent)

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

## Daemon Mode Architecture

### Instruction Handler Framework

The agent uses a handler registry pattern for processing instructions:

**Location:** `internal/instruction/`

- `handler.go`: Defines `Handler` interface and `Registry` for managing handlers
- `handlers/`: Individual instruction type implementations

**Adding a new instruction handler:**

1. Create handler in `internal/instruction/handlers/<type>.go`
2. Implement the `Handler` interface with `Execute(ctx, instruction) (result, error)`
3. Register handler in `agent.New()` in `internal/agent/agent.go`

### Polling Loop

**Location:** `internal/agent/agent.go` - `Agent.Run()` and `poll()` methods

The polling loop:
1. Creates a new gRPC connection per poll (resilient to server restarts)
2. Calls `GetInstructions` with agent ID and last instruction ID
3. Processes each instruction via handler registry
4. Updates poll interval if server specifies one
5. Closes connection
6. Sleeps for poll interval (or uses backoff on error)
7. Repeats until context cancellation (SIGTERM/SIGINT)

### Exponential Backoff

**Location:** `internal/agent/backoff.go`

On errors, the agent uses exponential backoff (1s → 2s → 4s → 8s → max 60s) before retrying. Backoff resets to 1s on successful poll.

### Graceful Shutdown

On SIGTERM/SIGINT:
1. Context is cancelled, breaking the polling loop
2. Agent calls `UnregisterAgent` to remove itself from server
3. Process exits cleanly

## Branch Information

This repository uses `master` as the main branch (not `main`).
