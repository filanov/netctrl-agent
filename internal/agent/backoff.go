package agent

import (
	"time"
)

const (
	// InitialBackoff is the starting backoff duration.
	InitialBackoff = 1 * time.Second
	// MaxBackoff is the maximum backoff duration.
	MaxBackoff = 60 * time.Second
)

// Backoff manages exponential backoff timing for retry logic.
type Backoff struct {
	current time.Duration
	max     time.Duration
}

// NewBackoff creates a new Backoff instance.
func NewBackoff() *Backoff {
	return &Backoff{
		current: InitialBackoff,
		max:     MaxBackoff,
	}
}

// Next returns the current backoff duration and increases it exponentially.
// The duration doubles each time until it reaches the maximum.
func (b *Backoff) Next() time.Duration {
	current := b.current
	b.current = b.current * 2
	if b.current > b.max {
		b.current = b.max
	}
	return current
}

// Reset resets the backoff to the initial duration.
func (b *Backoff) Reset() {
	b.current = InitialBackoff
}

// Current returns the current backoff duration without incrementing.
func (b *Backoff) Current() time.Duration {
	return b.current
}
