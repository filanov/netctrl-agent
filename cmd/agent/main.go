package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/filanov/netctrl-agent/internal/agent"
)

func main() {
	// Define command-line flags
	clusterID := flag.String("cluster-id", "", "Cluster ID (required)")
	serverAddr := flag.String("server-address", "localhost:9090", "Server address")
	timeout := flag.Duration("timeout", 10*time.Second, "Operation timeout")
	daemon := flag.Bool("daemon", false, "Run in daemon mode (continuous polling)")
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
	log.Printf("Daemon Mode: %v", *daemon)

	// Create agent instance
	agentInstance := agent.New(*clusterID, *serverAddr)

	if *daemon {
		// Daemon mode: Run continuously with signal handling
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		log.Printf("Running in daemon mode...")
		if err := agentInstance.Run(ctx); err != nil && err != context.Canceled {
			log.Fatalf("Agent daemon failed: %v", err)
		}

		log.Printf("Agent daemon stopped gracefully")
	} else {
		// One-shot mode: Register once and exit (backward compatible)
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()

		if err := agentInstance.Register(ctx); err != nil {
			log.Fatalf("Agent registration failed: %v", err)
		}

		log.Printf("Agent exiting successfully")
	}
}
