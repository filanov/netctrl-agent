package agent

import (
	"context"
	"fmt"
	"log"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
	"github.com/filanov/netctrl-agent/internal/client"
	"github.com/filanov/netctrl-agent/internal/discovery"
)

// Agent represents the netctrl agent instance.
type Agent struct {
	clusterID     string
	serverAddress string
}

// New creates a new Agent instance.
func New(clusterID, serverAddress string) *Agent {
	return &Agent{
		clusterID:     clusterID,
		serverAddress: serverAddress,
	}
}

// Register performs the agent registration workflow:
// 1. Discovers hostname and IP address
// 2. Generates deterministic UUID
// 3. Connects to gRPC server
// 4. Sends registration request
func (a *Agent) Register(ctx context.Context) error {
	// Discover hostname
	hostname, err := discovery.GetHostname()
	if err != nil {
		return fmt.Errorf("failed to discover hostname: %w", err)
	}
	log.Printf("Discovered hostname: %s", hostname)

	// Discover primary IP address
	ipAddress, err := discovery.GetPrimaryIPAddress()
	if err != nil {
		return fmt.Errorf("failed to discover IP address: %w", err)
	}
	log.Printf("Discovered IP address: %s", ipAddress)

	// Generate UUID from hostname and IP
	uuid := discovery.GenerateUUID(hostname, ipAddress)
	log.Printf("Generated UUID: %s", uuid)

	// Create gRPC client
	grpcClient, err := client.NewClient(a.serverAddress)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer grpcClient.Close()

	// Prepare registration request
	req := &v1.RegisterAgentRequest{
		Id:        uuid,
		ClusterId: a.clusterID,
		Hostname:  hostname,
		IpAddress: ipAddress,
		Version:   "0.1.0",
	}

	// Send registration request
	log.Printf("Registering agent with server at %s...", a.serverAddress)
	resp, err := grpcClient.RegisterAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	if resp.Agent == nil {
		return fmt.Errorf("registration response missing agent data")
	}

	log.Printf("Registration successful!")
	log.Printf("Agent ID: %s", resp.Agent.Id)
	log.Printf("Cluster ID: %s", resp.Agent.ClusterId)
	log.Printf("Hostname: %s", resp.Agent.Hostname)
	log.Printf("IP Address: %s", resp.Agent.IpAddress)
	log.Printf("Status: %s", resp.Agent.Status.String())

	return nil
}
