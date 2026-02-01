package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
)

// HealthCheckHandler handles HEALTH_CHECK instructions.
type HealthCheckHandler struct {
	startTime time.Time
	hostname  string
	ipAddress string
}

// NewHealthCheckHandler creates a new health check handler.
func NewHealthCheckHandler() *HealthCheckHandler {
	hostname, _ := os.Hostname()
	ipAddress := getLocalIP()

	return &HealthCheckHandler{
		startTime: time.Now(),
		hostname:  hostname,
		ipAddress: ipAddress,
	}
}

// Execute processes a HEALTH_CHECK instruction.
func (h *HealthCheckHandler) Execute(ctx context.Context, instruction *v1.Instruction) (string, error) {
	if instruction == nil {
		return "", fmt.Errorf("instruction is nil")
	}

	// Calculate uptime
	uptime := time.Since(h.startTime)

	// Collect health data
	healthData := map[string]interface{}{
		"status":       "active",
		"hostname":     h.hostname,
		"ip_address":   h.ipAddress,
		"uptime_seconds": int(uptime.Seconds()),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	// Marshal to JSON
	resultJSON, err := json.Marshal(healthData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal health data: %w", err)
	}

	return string(resultJSON), nil
}

// getLocalIP returns the non-loopback local IP address.
// If multiple IPs are available, returns the first one found.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}
