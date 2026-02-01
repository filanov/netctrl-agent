package agent

import (
	"testing"
	"time"
)

func TestNewBackoff(t *testing.T) {
	backoff := NewBackoff()
	if backoff == nil {
		t.Fatal("NewBackoff returned nil")
	}
	if backoff.current != InitialBackoff {
		t.Errorf("initial backoff = %v, want %v", backoff.current, InitialBackoff)
	}
	if backoff.max != MaxBackoff {
		t.Errorf("max backoff = %v, want %v", backoff.max, MaxBackoff)
	}
}

func TestBackoff_Next(t *testing.T) {
	backoff := NewBackoff()

	tests := []struct {
		name     string
		expected time.Duration
	}{
		{"first call", 1 * time.Second},
		{"second call", 2 * time.Second},
		{"third call", 4 * time.Second},
		{"fourth call", 8 * time.Second},
		{"fifth call", 16 * time.Second},
		{"sixth call", 32 * time.Second},
		{"seventh call", 60 * time.Second}, // max reached
		{"eighth call", 60 * time.Second},  // stays at max
		{"ninth call", 60 * time.Second},   // stays at max
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := backoff.Next()
			if duration != tt.expected {
				t.Errorf("Next() = %v, want %v", duration, tt.expected)
			}
		})
	}
}

func TestBackoff_Current(t *testing.T) {
	backoff := NewBackoff()

	// Current should return initial value without incrementing
	current := backoff.Current()
	if current != InitialBackoff {
		t.Errorf("Current() = %v, want %v", current, InitialBackoff)
	}

	// Current should still return the same value
	current = backoff.Current()
	if current != InitialBackoff {
		t.Errorf("Current() = %v, want %v (should not increment)", current, InitialBackoff)
	}

	// After Next, Current should return the new value
	backoff.Next()
	current = backoff.Current()
	if current != 2*time.Second {
		t.Errorf("Current() after Next = %v, want %v", current, 2*time.Second)
	}
}

func TestBackoff_Reset(t *testing.T) {
	backoff := NewBackoff()

	// Advance backoff several times
	backoff.Next()
	backoff.Next()
	backoff.Next()

	// Current should be 8 seconds now
	if backoff.Current() != 8*time.Second {
		t.Errorf("Current() before reset = %v, want %v", backoff.Current(), 8*time.Second)
	}

	// Reset should bring it back to initial
	backoff.Reset()
	if backoff.Current() != InitialBackoff {
		t.Errorf("Current() after reset = %v, want %v", backoff.Current(), InitialBackoff)
	}

	// Next after reset should return initial value
	duration := backoff.Next()
	if duration != InitialBackoff {
		t.Errorf("Next() after reset = %v, want %v", duration, InitialBackoff)
	}
}

func TestBackoff_MaxCap(t *testing.T) {
	backoff := NewBackoff()

	// Advance backoff many times to ensure it caps at max
	for i := 0; i < 20; i++ {
		backoff.Next()
	}

	// Should be capped at max
	if backoff.Current() != MaxBackoff {
		t.Errorf("Current() after many Next calls = %v, want %v", backoff.Current(), MaxBackoff)
	}

	// Next should still return max
	duration := backoff.Next()
	if duration != MaxBackoff {
		t.Errorf("Next() at max = %v, want %v", duration, MaxBackoff)
	}
}

func TestBackoff_Progression(t *testing.T) {
	backoff := NewBackoff()

	expected := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		32 * time.Second,
		60 * time.Second,
	}

	for i, exp := range expected {
		got := backoff.Next()
		if got != exp {
			t.Errorf("progression[%d] = %v, want %v", i, got, exp)
		}
	}
}
