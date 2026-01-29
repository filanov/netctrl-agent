package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/filanov/netctrl-agent/internal/agent"
)

func main() {
	// Define command-line flags
	clusterID := flag.String("cluster-id", "", "Cluster ID (required)")
	serverAddr := flag.String("server-address", "localhost:9090", "Server address")
	timeout := flag.Duration("timeout", 10*time.Second, "Operation timeout")
	flag.Parse()

	// Check for cluster ID from environment variable if not provided via flag
	if *clusterID == "" {
		*clusterID = os.Getenv("NETCTRL_CLUSTER_ID")
	}

	// Validate cluster ID is provided
	if *clusterID == "" {
		log.Fatal("cluster-id is required (use --cluster-id flag or NETCTRL_CLUSTER_ID env var)")
	}

	log.Printf("Starting netctrl-agent...")
	log.Printf("Cluster ID: %s", *clusterID)
	log.Printf("Server Address: %s", *serverAddr)

	// Create agent instance
	agentInstance := agent.New(*clusterID, *serverAddr)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Perform registration
	if err := agentInstance.Register(ctx); err != nil {
		log.Fatalf("Agent registration failed: %v", err)
	}

	log.Printf("Agent exiting successfully")
}
