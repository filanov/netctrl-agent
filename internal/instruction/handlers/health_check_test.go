package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
)

func TestNewHealthCheckHandler(t *testing.T) {
	handler := NewHealthCheckHandler()

	if handler == nil {
		t.Fatal("NewHealthCheckHandler returned nil")
	}

	if handler.startTime.IsZero() {
		t.Error("handler startTime is zero")
	}

	// hostname and ipAddress may be empty in some environments, so just check they're initialized
	if handler.hostname == "" {
		t.Log("Warning: hostname is empty")
	}
}

func TestHealthCheckHandler_Execute(t *testing.T) {
	tests := []struct {
		name          string
		instruction   *v1.Instruction
		expectedError bool
	}{
		{
			name: "valid health check",
			instruction: &v1.Instruction{
				Id:      "test-1",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
				Payload: "",
			},
			expectedError: false,
		},
		{
			name:          "nil instruction",
			instruction:   nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHealthCheckHandler()
			ctx := context.Background()

			// Wait a small amount to ensure uptime is non-zero
			time.Sleep(10 * time.Millisecond)

			result, err := handler.Execute(ctx, tt.instruction)

			if (err != nil) != tt.expectedError {
				t.Errorf("Execute() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if !tt.expectedError {
				// Validate result JSON structure
				var healthData map[string]interface{}
				if err := json.Unmarshal([]byte(result), &healthData); err != nil {
					t.Errorf("failed to parse result JSON: %v", err)
					return
				}

				// Check required fields
				requiredFields := []string{"status", "hostname", "ip_address", "uptime_seconds", "timestamp"}
				for _, field := range requiredFields {
					if _, ok := healthData[field]; !ok {
						t.Errorf("missing required field: %s", field)
					}
				}

				// Validate status
				if status, ok := healthData["status"].(string); !ok || status != "active" {
					t.Errorf("status = %v, want 'active'", healthData["status"])
				}

				// Validate uptime_seconds is a number and >= 0
				if uptimeSec, ok := healthData["uptime_seconds"].(float64); !ok || uptimeSec < 0 {
					t.Errorf("uptime_seconds = %v, want >= 0", healthData["uptime_seconds"])
				}

				// Validate timestamp format
				if timestamp, ok := healthData["timestamp"].(string); ok {
					if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
						t.Errorf("invalid timestamp format: %v", err)
					}
				} else {
					t.Error("timestamp is not a string")
				}
			}
		})
	}
}

func TestHealthCheckHandler_Execute_Uptime(t *testing.T) {
	handler := NewHealthCheckHandler()
	ctx := context.Background()

	// Wait 100ms to ensure measurable uptime
	time.Sleep(100 * time.Millisecond)

	instruction := &v1.Instruction{
		Id:      "test-uptime",
		Type:    v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
		Payload: "",
	}

	result, err := handler.Execute(ctx, instruction)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var healthData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &healthData); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	uptimeSec, ok := healthData["uptime_seconds"].(float64)
	if !ok {
		t.Fatal("uptime_seconds is not a number")
	}

	// Uptime should be at least 0.1 seconds (100ms)
	if uptimeSec < 0.1 {
		t.Errorf("uptime_seconds = %v, want >= 0.1", uptimeSec)
	}
}

func TestHealthCheckHandler_Execute_ContextCancellation(t *testing.T) {
	handler := NewHealthCheckHandler()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	instruction := &v1.Instruction{
		Id:      "test-context",
		Type:    v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
		Payload: "",
	}

	// Execute should still work even with cancelled context
	result, err := handler.Execute(ctx, instruction)
	if err != nil {
		t.Errorf("Execute() with cancelled context returned error: %v", err)
	}

	if result == "" {
		t.Error("Execute() returned empty result")
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := getLocalIP()
	// IP may be empty in some test environments, so just log it
	t.Logf("Local IP: %s", ip)
}
