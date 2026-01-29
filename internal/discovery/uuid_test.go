package discovery

import (
	"testing"
)

func TestGenerateUUID(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		ip       string
	}{
		{
			name:     "basic test",
			hostname: "test-host",
			ip:       "192.168.1.100",
		},
		{
			name:     "different host",
			hostname: "prod-node-01",
			ip:       "10.0.0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate UUID twice with same inputs
			uuid1 := GenerateUUID(tt.hostname, tt.ip)
			uuid2 := GenerateUUID(tt.hostname, tt.ip)

			// Should be deterministic (same inputs = same output)
			if uuid1 != uuid2 {
				t.Errorf("UUID generation is not deterministic: %s != %s", uuid1, uuid2)
			}

			// Should match UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
			if len(uuid1) != 36 {
				t.Errorf("UUID has incorrect length: got %d, want 36", len(uuid1))
			}

			// Check hyphen positions
			if uuid1[8] != '-' || uuid1[13] != '-' || uuid1[18] != '-' || uuid1[23] != '-' {
				t.Errorf("UUID has incorrect format: %s", uuid1)
			}
		})
	}
}

func TestGenerateUUID_DifferentInputs(t *testing.T) {
	uuid1 := GenerateUUID("host1", "192.168.1.1")
	uuid2 := GenerateUUID("host2", "192.168.1.1")
	uuid3 := GenerateUUID("host1", "192.168.1.2")

	// Different hostnames should produce different UUIDs
	if uuid1 == uuid2 {
		t.Error("Different hostnames produced same UUID")
	}

	// Different IPs should produce different UUIDs
	if uuid1 == uuid3 {
		t.Error("Different IPs produced same UUID")
	}
}
