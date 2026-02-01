package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
)

func TestNewPollIntervalHandler(t *testing.T) {
	callback := func(interval time.Duration) {}
	handler := NewPollIntervalHandler(callback)

	if handler == nil {
		t.Fatal("NewPollIntervalHandler returned nil")
	}
	if handler.callback == nil {
		t.Error("handler callback is nil")
	}
}

func TestPollIntervalHandler_Execute(t *testing.T) {
	tests := []struct {
		name          string
		instruction   *v1.Instruction
		expectedError bool
		expectCalled  bool
		expectedInt   int
	}{
		{
			name: "valid interval - minimum",
			instruction: &v1.Instruction{
				Id:      "test-1",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
				Payload: `{"interval_seconds": 10}`,
			},
			expectedError: false,
			expectCalled:  true,
			expectedInt:   10,
		},
		{
			name: "valid interval - maximum",
			instruction: &v1.Instruction{
				Id:      "test-2",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
				Payload: `{"interval_seconds": 300}`,
			},
			expectedError: false,
			expectCalled:  true,
			expectedInt:   300,
		},
		{
			name: "valid interval - middle range",
			instruction: &v1.Instruction{
				Id:      "test-3",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
				Payload: `{"interval_seconds": 120}`,
			},
			expectedError: false,
			expectCalled:  true,
			expectedInt:   120,
		},
		{
			name: "interval too low",
			instruction: &v1.Instruction{
				Id:      "test-4",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
				Payload: `{"interval_seconds": 5}`,
			},
			expectedError: true,
			expectCalled:  false,
		},
		{
			name: "interval too high",
			instruction: &v1.Instruction{
				Id:      "test-5",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
				Payload: `{"interval_seconds": 301}`,
			},
			expectedError: true,
			expectCalled:  false,
		},
		{
			name: "invalid JSON payload",
			instruction: &v1.Instruction{
				Id:      "test-6",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
				Payload: `{invalid json}`,
			},
			expectedError: true,
			expectCalled:  false,
		},
		{
			name: "empty payload",
			instruction: &v1.Instruction{
				Id:      "test-7",
				Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
				Payload: `{}`,
			},
			expectedError: true,
			expectCalled:  false,
		},
		{
			name:          "nil instruction",
			instruction:   nil,
			expectedError: true,
			expectCalled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var receivedInterval time.Duration

			callback := func(interval time.Duration) {
				called = true
				receivedInterval = interval
			}

			handler := NewPollIntervalHandler(callback)
			ctx := context.Background()

			result, err := handler.Execute(ctx, tt.instruction)

			if (err != nil) != tt.expectedError {
				t.Errorf("Execute() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if called != tt.expectCalled {
				t.Errorf("callback called = %v, expectCalled %v", called, tt.expectCalled)
			}

			if tt.expectCalled {
				expectedDuration := time.Duration(tt.expectedInt) * time.Second
				if receivedInterval != expectedDuration {
					t.Errorf("callback received interval = %v, want %v", receivedInterval, expectedDuration)
				}

				// Validate result JSON
				var resultMap map[string]interface{}
				if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
					t.Errorf("failed to parse result JSON: %v", err)
				}

				if status, ok := resultMap["status"].(string); !ok || status != "ok" {
					t.Errorf("result status = %v, want 'ok'", resultMap["status"])
				}

				if intervalSec, ok := resultMap["interval_seconds"].(float64); !ok || int(intervalSec) != tt.expectedInt {
					t.Errorf("result interval_seconds = %v, want %d", resultMap["interval_seconds"], tt.expectedInt)
				}
			}
		})
	}
}

func TestPollIntervalHandler_Execute_NilCallback(t *testing.T) {
	handler := NewPollIntervalHandler(nil)
	ctx := context.Background()

	instruction := &v1.Instruction{
		Id:      "test-nil-callback",
		Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
		Payload: `{"interval_seconds": 60}`,
	}

	// Should not panic with nil callback
	result, err := handler.Execute(ctx, instruction)
	if err != nil {
		t.Errorf("Execute() with nil callback returned error: %v", err)
	}

	if result == "" {
		t.Error("Execute() returned empty result")
	}
}

func TestPollIntervalHandler_Execute_ContextCancellation(t *testing.T) {
	called := false
	callback := func(interval time.Duration) {
		called = true
	}

	handler := NewPollIntervalHandler(callback)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	instruction := &v1.Instruction{
		Id:      "test-context",
		Type:    v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
		Payload: `{"interval_seconds": 60}`,
	}

	// Execute should still work even with cancelled context
	_, err := handler.Execute(ctx, instruction)
	if err != nil {
		t.Errorf("Execute() with cancelled context returned error: %v", err)
	}

	if !called {
		t.Error("callback was not called with cancelled context")
	}
}
