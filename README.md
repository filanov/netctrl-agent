# netctrl-agent

The client-side component of the netctrl system. Runs on cluster nodes and communicates with netctrl-server to register nodes, receive network configurations, and report status.

## Features

- Auto-discovery of hostname and IP address
- Deterministic UUID generation based on hostname + IP
- gRPC-based registration with netctrl-server
- Command-line and environment variable configuration
- **Long-lived daemon**: Continuous polling for instructions with graceful shutdown
- **Instruction processing**: Handles POLL_INTERVAL and HEALTH_CHECK instructions
- **Exponential backoff**: Resilient reconnection on network failures
- **Heartbeat mechanism**: Each poll updates agent's last_seen timestamp

## Building

### Using Make (Recommended)

The project supports both Docker-based and local builds. Docker is automatically used if available.

```bash
# Build the binary (uses Docker if available)
make build

# Build using local Go (bypass Docker)
make build-local

# Run tests
make test

# Run linters
make lint

# Clean build artifacts
make clean
```

### Docker Builds

#### Development Image
Build a development image with all tools (Go, golangci-lint, etc.):

```bash
make docker-build-dev
```

Use the dev container for interactive development:

```bash
make docker-shell
```

#### Production Image
Build a multi-stage production image with minimal footprint:

```bash
# Standard build (native architecture)
make docker-build-prod

# Multi-architecture build (amd64, arm64)
make docker-build-multiarch

# Build and push to registry (amd64, arm64, arm/v7)
DOCKER_REGISTRY=docker.io/username make docker-push-multiarch
```

Run the agent in a production container:

```bash
NETCTRL_CLUSTER_ID=my-cluster make docker-run
```

**Multi-Architecture Support:**
The Docker build supports multiple architectures using Docker buildx:
- `linux/amd64` (x86_64)
- `linux/arm64` (ARM 64-bit, e.g., AWS Graviton, Apple Silicon)
- `linux/arm/v7` (ARM 32-bit, push target only)

### Manual Build

```bash
go build -o bin/netctrl-agent cmd/agent/main.go
```

## Usage

### Basic Usage

```bash
./bin/netctrl-agent --cluster-id=my-cluster
```

### Configuration Options

**Command-line flags:**
- `--cluster-id` (required): Cluster ID to register with
- `--server-address`: Server address (default: `localhost:9090`)

**Environment variables:**
- `NETCTRL_CLUSTER_ID`: Alternative way to provide cluster ID
- `NETCTRL_SERVER_ADDRESS`: Alternative way to provide server address

### Examples

The agent runs as a long-lived daemon that continuously polls for instructions:

```bash
# Run with default server (localhost:9090)
./bin/netctrl-agent --cluster-id=production

# Run with custom server
./bin/netctrl-agent --cluster-id=staging --server-address=10.0.0.5:9090

# Use environment variables
export NETCTRL_CLUSTER_ID=production
export NETCTRL_SERVER_ADDRESS=10.0.0.5:9090
./bin/netctrl-agent

# Docker with host networking
docker run --network host \
  -e NETCTRL_CLUSTER_ID=production \
  -e NETCTRL_SERVER_ADDRESS=10.0.0.5:9090 \
  netctrl-agent:latest

# Stop gracefully with SIGTERM or SIGINT (Ctrl+C)
# The agent will unregister before exiting
```

## Architecture

### Directory Structure

```
netctrl-agent/
├── cmd/
│   └── agent/          # Main entry point
├── internal/
│   ├── agent/          # Core agent logic (registration, polling, backoff)
│   ├── discovery/      # Network and UUID discovery
│   ├── client/         # gRPC client wrapper
│   └── instruction/    # Instruction handler framework
│       └── handlers/   # Instruction type implementations
├── bin/                # Built binaries
└── Makefile            # Build automation
```

### How It Works

The agent runs as a long-lived daemon that maintains continuous communication with the server:

1. **Discovery Phase**: Agent discovers hostname using `os.Hostname()` and finds the primary non-loopback IPv4 address
2. **UUID Generation**: Creates a deterministic UUID by hashing `hostname:ip` with SHA256
3. **Registration**: Sends `RegisterAgentRequest` with cluster ID, hostname, IP, and version to netctrl-server
4. **Polling Loop**: Continuously polls server using `GetInstructions` RPC
   - Default poll interval: 60 seconds
   - Server can adjust interval dynamically via `POLL_INTERVAL` instruction
   - Each poll acts as a heartbeat (updates agent's `last_seen` timestamp)
5. **Instruction Processing**: Executes instructions via handler registry
   - `POLL_INTERVAL`: Adjusts polling interval (10-300 seconds)
   - `HEALTH_CHECK`: Collects and reports agent health (uptime, status, hostname, IP)
6. **Error Handling**: Exponential backoff on failures (1s → 2s → 4s → 8s → max 60s)
7. **Graceful Shutdown**: On SIGTERM/SIGINT, sends `UnregisterAgent` request before exit

### UUID Generation

The agent generates a deterministic UUID from hostname and IP address:
```
SHA256(hostname:ip) → format as UUID
```

This ensures the same node always gets the same UUID, making re-registration idempotent.

### Instruction Types

The agent supports the following instruction types:

#### POLL_INTERVAL

Adjusts the agent's polling interval dynamically.

**Payload format:**
```json
{
  "interval_seconds": 120
}
```

**Valid range:** 10-300 seconds

**Example result:**
```json
{
  "status": "ok",
  "interval_seconds": 120
}
```

#### HEALTH_CHECK

Collects and reports agent health status.

**Payload:** Empty (no payload required)

**Example result:**
```json
{
  "status": "active",
  "hostname": "node-1",
  "ip_address": "10.0.1.5",
  "uptime_seconds": 3600,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Development

### Running Tests

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/discovery/
```

### Dependencies

- `github.com/filanov/netctrl-server` - Proto definitions and generated gRPC client
- `google.golang.org/grpc` - gRPC framework
- `google.golang.org/protobuf` - Protocol Buffers support

### Adding Dependencies

```bash
go get package@version
go mod tidy
```

## Testing with netctrl-server

### Using Local Binaries

1. Start the server:
```bash
cd ../netctrl-server
make docker-run
```

2. Run the agent:
```bash
./bin/netctrl-agent --cluster-id=test-cluster
```

3. Verify registration:
```bash
grpcurl -plaintext -d '{"cluster_id":"test-cluster"}' localhost:9090 netctrl.v1.AgentService/ListAgents
```

### Using Docker Containers

1. Start the server in Docker:
```bash
cd ../netctrl-server
make docker-run
```

2. Run the agent in Docker:
```bash
NETCTRL_CLUSTER_ID=test-cluster NETCTRL_SERVER_ADDRESS=host.docker.internal:9090 make docker-run
```

3. Verify registration using the server's REST API:
```bash
curl http://localhost:8080/api/v1/agents?cluster_id=test-cluster
```

## Version

Current version: `0.1.0`

## License

[Add license information]
