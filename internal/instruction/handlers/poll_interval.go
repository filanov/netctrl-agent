package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
)

const (
	// MinPollInterval is the minimum allowed polling interval in seconds.
	MinPollInterval = 10
	// MaxPollInterval is the maximum allowed polling interval in seconds.
	MaxPollInterval = 300
)

// PollIntervalPayload represents the JSON payload for POLL_INTERVAL instruction.
type PollIntervalPayload struct {
	IntervalSeconds int `json:"interval_seconds"`
}

// PollIntervalCallback is called when a valid poll interval instruction is received.
type PollIntervalCallback func(interval time.Duration)

// PollIntervalHandler handles POLL_INTERVAL instructions.
type PollIntervalHandler struct {
	callback PollIntervalCallback
}

// NewPollIntervalHandler creates a new poll interval handler with the given callback.
func NewPollIntervalHandler(callback PollIntervalCallback) *PollIntervalHandler {
	return &PollIntervalHandler{
		callback: callback,
	}
}

// Execute processes a POLL_INTERVAL instruction.
func (h *PollIntervalHandler) Execute(ctx context.Context, instruction *v1.Instruction) (string, error) {
	if instruction == nil {
		return "", fmt.Errorf("instruction is nil")
	}

	// Parse payload
	var payload PollIntervalPayload
	if err := json.Unmarshal([]byte(instruction.Payload), &payload); err != nil {
		return "", fmt.Errorf("failed to parse poll interval payload: %w", err)
	}

	// Validate interval range
	if payload.IntervalSeconds < MinPollInterval || payload.IntervalSeconds > MaxPollInterval {
		return "", fmt.Errorf("poll interval %d seconds is out of range [%d, %d]",
			payload.IntervalSeconds, MinPollInterval, MaxPollInterval)
	}

	// Call the callback to update the agent's poll interval
	if h.callback != nil {
		h.callback(time.Duration(payload.IntervalSeconds) * time.Second)
	}

	// Return success result
	result := map[string]interface{}{
		"status":           "ok",
		"interval_seconds": payload.IntervalSeconds,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}
