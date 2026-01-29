package discovery

import (
	"testing"
)

func TestGetHostname(t *testing.T) {
	hostname, err := GetHostname()
	if err != nil {
		t.Fatalf("GetHostname() failed: %v", err)
	}

	if hostname == "" {
		t.Error("GetHostname() returned empty hostname")
	}

	t.Logf("Hostname: %s", hostname)
}

func TestGetPrimaryIPAddress(t *testing.T) {
	ip, err := GetPrimaryIPAddress()
	if err != nil {
		t.Fatalf("GetPrimaryIPAddress() failed: %v", err)
	}

	if ip == "" {
		t.Error("GetPrimaryIPAddress() returned empty IP")
	}

	if ip == "127.0.0.1" {
		t.Error("GetPrimaryIPAddress() returned loopback address")
	}

	t.Logf("Primary IP: %s", ip)
}
