package instruction

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
)

// mockHandler is a test implementation of the Handler interface.
type mockHandler struct {
	result string
	err    error
}

func (m *mockHandler) Execute(ctx context.Context, instruction *v1.Instruction) (string, error) {
	return m.result, m.err
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if registry.handlers == nil {
		t.Fatal("registry handlers map is nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	handler := &mockHandler{result: "test"}

	registry.Register(v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK, handler)

	if !registry.HasHandler(v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK) {
		t.Error("handler not registered")
	}
}

func TestRegistry_HasHandler(t *testing.T) {
	registry := NewRegistry()
	handler := &mockHandler{result: "test"}

	tests := []struct {
		name           string
		registerType   v1.InstructionType
		checkType      v1.InstructionType
		expectedResult bool
	}{
		{
			name:           "registered handler",
			registerType:   v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
			checkType:      v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
			expectedResult: true,
		},
		{
			name:           "unregistered handler",
			registerType:   v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
			checkType:      v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(tt.registerType, handler)

			result := registry.HasHandler(tt.checkType)
			if result != tt.expectedResult {
				t.Errorf("HasHandler() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestRegistry_Execute(t *testing.T) {
	tests := []struct {
		name          string
		instruction   *v1.Instruction
		handler       Handler
		expectedRes   string
		expectedError bool
	}{
		{
			name: "successful execution",
			instruction: &v1.Instruction{
				Id:   "test-1",
				Type: v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
			},
			handler:       &mockHandler{result: `{"status": "ok"}`, err: nil},
			expectedRes:   `{"status": "ok"}`,
			expectedError: false,
		},
		{
			name: "handler returns error",
			instruction: &v1.Instruction{
				Id:   "test-2",
				Type: v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
			},
			handler:       &mockHandler{result: "", err: errors.New("handler error")},
			expectedRes:   "",
			expectedError: true,
		},
		{
			name:          "nil instruction",
			instruction:   nil,
			handler:       &mockHandler{result: "test", err: nil},
			expectedRes:   "",
			expectedError: true,
		},
		{
			name: "unregistered instruction type",
			instruction: &v1.Instruction{
				Id:   "test-3",
				Type: v1.InstructionType_INSTRUCTION_TYPE_POLL_INTERVAL,
			},
			handler:       nil,
			expectedRes:   "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			if tt.handler != nil {
				registry.Register(v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK, tt.handler)
			}

			ctx := context.Background()
			result, err := registry.Execute(ctx, tt.instruction)

			if (err != nil) != tt.expectedError {
				t.Errorf("Execute() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if result != tt.expectedRes {
				t.Errorf("Execute() result = %v, want %v", result, tt.expectedRes)
			}
		})
	}
}

func TestRegistry_Execute_ContextCancellation(t *testing.T) {
	registry := NewRegistry()

	// Handler that checks context cancellation
	ctxCheckHandler := &mockHandler{result: "test", err: nil}
	registry.Register(v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK, ctxCheckHandler)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	instruction := &v1.Instruction{
		Id:   "test-cancel",
		Type: v1.InstructionType_INSTRUCTION_TYPE_HEALTH_CHECK,
	}

	// Execute should still work even with cancelled context
	// The handler itself needs to respect context cancellation
	_, err := registry.Execute(ctx, instruction)
	if err != nil {
		t.Errorf("Execute() with cancelled context returned error: %v", err)
	}
}
