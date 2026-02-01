package instruction

import (
	"context"
	"fmt"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
)

// Handler defines the interface for instruction handlers.
type Handler interface {
	// Execute processes the instruction and returns result data.
	// Returns result as JSON string or empty string if no result.
	Execute(ctx context.Context, instruction *v1.Instruction) (string, error)
}

// Registry manages instruction handlers and executes instructions.
type Registry struct {
	handlers map[v1.InstructionType]Handler
}

// NewRegistry creates a new instruction handler registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[v1.InstructionType]Handler),
	}
}

// Register adds a handler for a specific instruction type.
func (r *Registry) Register(instructionType v1.InstructionType, handler Handler) {
	r.handlers[instructionType] = handler
}

// Execute processes an instruction using the registered handler.
// Returns result data and error. If the instruction type is not registered,
// returns an error.
func (r *Registry) Execute(ctx context.Context, instruction *v1.Instruction) (string, error) {
	if instruction == nil {
		return "", fmt.Errorf("instruction is nil")
	}

	handler, ok := r.handlers[instruction.Type]
	if !ok {
		return "", fmt.Errorf("no handler registered for instruction type: %v", instruction.Type)
	}

	return handler.Execute(ctx, instruction)
}

// HasHandler returns true if a handler is registered for the given instruction type.
func (r *Registry) HasHandler(instructionType v1.InstructionType) bool {
	_, ok := r.handlers[instructionType]
	return ok
}
