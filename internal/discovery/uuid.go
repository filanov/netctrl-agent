package discovery

import (
	"crypto/sha256"
	"fmt"
)

// GenerateUUID creates a deterministic UUID from hostname and IP address.
// Uses SHA256 hash of "hostname:ip" formatted as a UUID string.
func GenerateUUID(hostname, ip string) string {
	data := hostname + ":" + ip
	hash := sha256.Sum256([]byte(data))

	// Format first 16 bytes as UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}
