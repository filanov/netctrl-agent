package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
	"github.com/filanov/netctrl-agent/internal/client"
	"github.com/filanov/netctrl-agent/internal/discovery"
	"github.com/filanov/netctrl-agent/internal/instruction"
	"github.com/filanov/netctrl-agent/internal/instruction/handlers"
)

// Agent represents the netctrl agent instance.
type Agent struct {
	clusterID          string
	serverAddress      string
	agentID            string
	hostname           string
	ipAddress          string
	lastInstructionID  string
	lastResultData     string
	pollInterval       time.Duration
	registry           *instruction.Registry
}

// New creates a new Agent instance with instruction handlers.
func New(clusterID, serverAddress string) *Agent {
	agent := &Agent{
		clusterID:     clusterID,
		serverAddress: serverAddress,
		pollInterval:  60 * time.Second, // Default 60 seconds
		registry:      instruction.NewRegistry(),
	}

	// Register instruction handlers
	agent.registry.Register(
		v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
		handlers.NewPollIntervalHandler(agent.updatePollInterval),
	)
	agent.registry.Register(
		v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
		handlers.NewHealthCheckHandler(),
	)
	agent.registry.Register(
		v1.InstructionType_INSTRUCTION_TYPE_COLLECT_HARDWARE,
		handlers.NewCollectHardwareHandler(),
	)

	return agent
}

// updatePollInterval updates the agent's polling interval.
func (a *Agent) updatePollInterval(interval time.Duration) {
	log.Printf("Updating poll interval to %v", interval)
	a.pollInterval = interval
}

// Run starts the agent in daemon mode:
// 1. Registers with the server
// 2. Enters polling loop to fetch and process instructions
// 3. Handles graceful shutdown
func (a *Agent) Run(ctx context.Context) error {
	// Register first
	if err := a.Register(ctx); err != nil {
		return fmt.Errorf("initial registration failed: %w", err)
	}

	log.Printf("Starting daemon mode with %v poll interval", a.pollInterval)

	backoff := NewBackoff()
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutdown signal received, unregistering agent...")
			// Use a fresh context for unregister since ctx is cancelled
			unregisterCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := a.Unregister(unregisterCtx); err != nil {
				log.Printf("Warning: failed to unregister agent: %v", err)
			}
			return ctx.Err()

		case <-ticker.C:
			if err := a.poll(ctx); err != nil {
				log.Printf("Poll error: %v", err)
				// Apply backoff on error
				sleepDuration := backoff.Next()
				log.Printf("Backing off for %v", sleepDuration)
				time.Sleep(sleepDuration)
			} else {
				// Reset backoff on success
				backoff.Reset()
				// Update ticker with current poll interval (may have changed)
				ticker.Reset(a.pollInterval)
			}
		}
	}
}

// poll fetches and processes instructions from the server.
func (a *Agent) poll(ctx context.Context) error {
	// Create gRPC client for this poll
	grpcClient, err := client.NewClient(a.serverAddress)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer grpcClient.Close()

	// Prepare GetInstructions request
	req := &v1.GetInstructionsRequest{
		AgentId:           a.agentID,
		LastInstructionId: a.lastInstructionID,
		ResultData:        a.lastResultData,
	}

	// Get instructions from server
	resp, err := grpcClient.GetInstructions(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get instructions: %w", err)
	}

	// Update poll interval if server specifies one
	if resp.PollIntervalSeconds > 0 {
		newInterval := time.Duration(resp.PollIntervalSeconds) * time.Second
		if newInterval != a.pollInterval {
			log.Printf("Server updated poll interval to %v", newInterval)
			a.pollInterval = newInterval
		}
	}

	// Clear previous result data after it's been sent
	a.lastResultData = ""

	// Process each instruction
	for _, instruction := range resp.Instructions {
		log.Printf("Processing instruction: id=%s, type=%s", instruction.Id, instruction.Type)

		// Check if handler is registered for this instruction type
		if !a.registry.HasHandler(instruction.Type) {
			log.Printf("Warning: no handler for instruction type %s, skipping", instruction.Type)
			continue
		}

		// Execute instruction
		resultData, err := a.registry.Execute(ctx, instruction)
		if err != nil {
			log.Printf("Error executing instruction %s: %v", instruction.Id, err)
			// Store error as result for next poll
			errorResult := map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
			if errorJSON, err := json.Marshal(errorResult); err == nil {
				a.lastResultData = string(errorJSON)
			}
			a.lastInstructionID = instruction.Id
			// Continue processing other instructions
			continue
		}

		log.Printf("Successfully executed instruction %s", instruction.Id)
		if resultData != "" {
			log.Printf("Result: %s", resultData)
		}

		// Store result data and instruction ID for next poll
		a.lastInstructionID = instruction.Id
		a.lastResultData = resultData
	}

	return nil
}

// Unregister removes the agent registration from the server.
func (a *Agent) Unregister(ctx context.Context) error {
	if a.agentID == "" {
		return fmt.Errorf("agent not registered")
	}

	// Create gRPC client
	grpcClient, err := client.NewClient(a.serverAddress)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer grpcClient.Close()

	// Prepare unregister request
	req := &v1.UnregisterAgentRequest{
		Id: a.agentID,
	}

	// Send unregister request
	log.Printf("Unregistering agent %s...", a.agentID)
	_, err = grpcClient.UnregisterAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("unregister failed: %w", err)
	}

	log.Printf("Agent unregistered successfully")
	return nil
}

// Register performs the agent registration workflow:
// 1. Discovers hostname and IP address
// 2. Generates deterministic UUID
// 3. Connects to gRPC server
// 4. Sends registration request
// 5. Stores agent state for daemon mode
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

	// Store state for daemon mode
	a.hostname = hostname
	a.ipAddress = ipAddress
	a.agentID = uuid

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
