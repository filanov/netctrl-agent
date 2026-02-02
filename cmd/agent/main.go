package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/filanov/netctrl-agent/internal/agent"
)

func main() {
	// Get configuration from environment variables (can be overridden by flags)
	defaultServerAddr := os.Getenv("NETCTRL_SERVER_ADDRESS")
	if defaultServerAddr == "" {
		defaultServerAddr = "localhost:9090"
	}

	// Define command-line flags
	clusterID := flag.String("cluster-id", "", "Cluster ID (required)")
	serverAddr := flag.String("server-address", defaultServerAddr, "Server address")
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

	// Run agent in daemon mode with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agentInstance.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Agent failed: %v", err)
	}

	log.Printf("Agent stopped gracefully")
}
