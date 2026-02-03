package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	v1 "github.com/filanov/netctrl-server/pkg/api/v1"
)

// CollectHardwareHandler handles COLLECT_HARDWARE instructions.
type CollectHardwareHandler struct{}

// NewCollectHardwareHandler creates a new hardware collection handler.
func NewCollectHardwareHandler() *CollectHardwareHandler {
	return &CollectHardwareHandler{}
}

// Execute collects Mellanox NIC information.
func (h *CollectHardwareHandler) Execute(ctx context.Context, instruction *v1.Instruction) (string, error) {
	if instruction == nil {
		return "", fmt.Errorf("instruction is nil")
	}

	// Collect Mellanox NICs
	nics, err := collectMellanoxNICs()
	if err != nil {
		return "", fmt.Errorf("failed to collect Mellanox NICs: %w", err)
	}

	// Convert to proto format
	protoNICs := convertToProtoNICs(nics)

	// Wrap in the format expected by the server
	result := map[string]interface{}{
		"instruction_type": "INSTRUCTION_TYPE_COLLECT_HARDWARE",
		"data": map[string]interface{}{
			"network_interfaces": protoNICs,
		},
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal hardware data: %w", err)
	}

	return string(resultJSON), nil
}

// NICInfo represents collected NIC information for JSON serialization
type NICInfo struct {
	DeviceName      string     `json:"device_name"`
	PCIAddress      string     `json:"pci_address"`
	PartNumber      string     `json:"part_number"`
	SerialNumber    string     `json:"serial_number"`
	FirmwareVersion string     `json:"firmware_version"`
	PortCount       int        `json:"port_count"`
	PSID            string     `json:"psid"`
	Ports           []PortInfo `json:"ports"`
}

// PortInfo represents collected port information for JSON serialization
type PortInfo struct {
	Number        int    `json:"number"`
	State         string `json:"state"`
	Speed         string `json:"speed"`
	MACAddress    string `json:"mac_address"`
	MTU           int    `json:"mtu"`
	GUID          string `json:"guid"`
	PCIAddress    string `json:"pci_address"`
	InterfaceName string `json:"interface_name"`
}

// convertToProtoNICs converts internal NICInfo to proto-compatible format.
func convertToProtoNICs(nics []NICInfo) []map[string]interface{} {
	var protoNICs []map[string]interface{}

	for _, nic := range nics {
		protoNIC := map[string]interface{}{
			"device_name":      nic.DeviceName,
			"pci_address":      nic.PCIAddress,
			"part_number":      nic.PartNumber,
			"serial_number":    nic.SerialNumber,
			"firmware_version": nic.FirmwareVersion,
			"port_count":       nic.PortCount,
			"psid":             nic.PSID,
		}

		// Convert ports
		if len(nic.Ports) > 0 {
			var protoPorts []map[string]interface{}
			for _, port := range nic.Ports {
				protoPort := map[string]interface{}{
					"number":         port.Number,
					"state":          convertPortState(port.State),
					"speed":          convertPortSpeed(port.Speed),
					"mac_address":    port.MACAddress,
					"mtu":            port.MTU,
					"guid":           port.GUID,
					"pci_address":    port.PCIAddress,
					"interface_name": port.InterfaceName,
				}
				protoPorts = append(protoPorts, protoPort)
			}
			protoNIC["ports"] = protoPorts
		}

		protoNICs = append(protoNICs, protoNIC)
	}

	return protoNICs
}

// convertPortState converts string state to proto enum value.
func convertPortState(state string) string {
	switch strings.ToLower(state) {
	case "up":
		return "PORT_STATE_UP"
	case "down":
		return "PORT_STATE_DOWN"
	case "testing":
		return "PORT_STATE_TESTING"
	default:
		return "PORT_STATE_UNSPECIFIED"
	}
}

// convertPortSpeed converts speed string (e.g., "100G") to proto enum value.
func convertPortSpeed(speed string) string {
	switch speed {
	case "1G":
		return "PORT_SPEED_1G"
	case "10G":
		return "PORT_SPEED_10G"
	case "25G":
		return "PORT_SPEED_25G"
	case "40G":
		return "PORT_SPEED_40G"
	case "50G":
		return "PORT_SPEED_50G"
	case "100G":
		return "PORT_SPEED_100G"
	case "200G":
		return "PORT_SPEED_200G"
	case "400G":
		return "PORT_SPEED_400G"
	default:
		return "PORT_SPEED_UNSPECIFIED"
	}
}

// collectMellanoxNICs discovers and collects information about Mellanox NICs.
func collectMellanoxNICs() ([]NICInfo, error) {
	// Find Mellanox devices via lspci
	pciDevices, err := findMellanoxPCIDevices()
	if err != nil {
		return nil, fmt.Errorf("failed to find Mellanox devices: %w", err)
	}

	var nics []NICInfo

	for _, pciAddr := range pciDevices {
		nic, err := collectNICInfo(pciAddr)
		if err != nil {
			// Log error but continue with other devices
			continue
		}
		nics = append(nics, nic)
	}

	return nics, nil
}

// findMellanoxPCIDevices finds Mellanox network devices using lspci.
func findMellanoxPCIDevices() ([]string, error) {
	cmd := exec.Command("lspci", "-D", "-d", "15b3:")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lspci command failed: %w", err)
	}

	var devices []string
	lines := strings.Split(string(output), "\n")
	pciRegex := regexp.MustCompile(`^([0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-9])`)

	for _, line := range lines {
		if line == "" {
			continue
		}
		matches := pciRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			devices = append(devices, matches[1])
		}
	}

	return devices, nil
}

// collectNICInfo collects detailed information about a specific NIC.
func collectNICInfo(pciAddr string) (NICInfo, error) {
	nic := NICInfo{
		PCIAddress: pciAddr,
	}

	// Get device name from sysfs
	deviceName, err := getDeviceNameFromPCI(pciAddr)
	if err == nil {
		nic.DeviceName = deviceName
	}

	// Get firmware version and other details using mstflint if available
	if fwVer := getFirmwareVersion(pciAddr); fwVer != "" {
		nic.FirmwareVersion = fwVer
	}

	// Get part number and serial number from sysfs or mstflint
	if partNum := getPartNumber(pciAddr); partNum != "" {
		nic.PartNumber = partNum
	}

	if serialNum := getSerialNumber(pciAddr); serialNum != "" {
		nic.SerialNumber = serialNum
	}

	if psid := getPSID(pciAddr); psid != "" {
		nic.PSID = psid
	}

	// Collect port information
	ports, err := collectPorts(pciAddr, deviceName)
	if err == nil && len(ports) > 0 {
		nic.Ports = ports
		nic.PortCount = len(ports)
	}

	return nic, nil
}

// getDeviceNameFromPCI gets the device name (e.g., mlx5_0) from PCI address.
func getDeviceNameFromPCI(pciAddr string) (string, error) {
	// Look in /sys/bus/pci/devices/<pci>/infiniband/
	ibPath := filepath.Join("/sys/bus/pci/devices", pciAddr, "infiniband")

	entries, err := os.ReadDir(ibPath)
	if err != nil {
		// Try net instead
		netPath := filepath.Join("/sys/bus/pci/devices", pciAddr, "net")
		entries, err = os.ReadDir(netPath)
		if err != nil {
			return "", err
		}
		if len(entries) > 0 {
			return entries[0].Name(), nil
		}
		return "", fmt.Errorf("no device found")
	}

	if len(entries) > 0 {
		return entries[0].Name(), nil
	}

	return "", fmt.Errorf("no infiniband device found")
}

// getFirmwareVersion attempts to get firmware version.
func getFirmwareVersion(pciAddr string) string {
	// Try using mstflint if available
	cmd := exec.Command("mstflint", "-d", pciAddr, "q")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse firmware version from mstflint output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "FW Version:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// getPartNumber attempts to get part number.
func getPartNumber(pciAddr string) string {
	cmd := exec.Command("mstflint", "-d", pciAddr, "q")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Part Number:") || strings.Contains(line, "Product Ver:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// getSerialNumber attempts to get serial number.
func getSerialNumber(pciAddr string) string {
	// Try reading from sysfs
	serialPath := filepath.Join("/sys/bus/pci/devices", pciAddr, "vpd")
	data, err := os.ReadFile(serialPath)
	if err == nil {
		// Simple extraction - VPD format is complex, this is a basic attempt
		if sn := extractSerialFromVPD(string(data)); sn != "" {
			return sn
		}
	}

	return ""
}

// getPSID attempts to get PSID.
func getPSID(pciAddr string) string {
	cmd := exec.Command("mstflint", "-d", pciAddr, "q")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "PSID:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// extractSerialFromVPD extracts serial number from VPD data.
func extractSerialFromVPD(vpd string) string {
	// VPD format is complex; this is a simplified extraction
	// Look for "SN" tag followed by data
	if idx := strings.Index(vpd, "SN"); idx != -1 && idx+2 < len(vpd) {
		// Extract a reasonable length serial number
		end := idx + 2 + 20
		if end > len(vpd) {
			end = len(vpd)
		}
		sn := vpd[idx+2 : end]
		// Clean non-printable characters
		return strings.Map(func(r rune) rune {
			if r >= 32 && r < 127 {
				return r
			}
			return -1
		}, sn)
	}
	return ""
}

// collectPorts collects information about NIC ports.
func collectPorts(pciAddr, deviceName string) ([]PortInfo, error) {
	var ports []PortInfo

	// Find network interfaces associated with this PCI device
	netPath := filepath.Join("/sys/bus/pci/devices", pciAddr, "net")
	entries, err := os.ReadDir(netPath)
	if err != nil {
		return nil, err
	}

	for portNum, entry := range entries {
		ifName := entry.Name()

		port := PortInfo{
			Number:        portNum + 1,
			InterfaceName: ifName,
			PCIAddress:    pciAddr,
		}

		// Get port state
		statePath := filepath.Join(netPath, ifName, "operstate")
		if stateData, err := os.ReadFile(statePath); err == nil {
			state := strings.TrimSpace(string(stateData))
			port.State = state
		}

		// Get MAC address
		macPath := filepath.Join(netPath, ifName, "address")
		if macData, err := os.ReadFile(macPath); err == nil {
			port.MACAddress = strings.TrimSpace(string(macData))
		}

		// Get MTU
		mtuPath := filepath.Join(netPath, ifName, "mtu")
		if mtuData, err := os.ReadFile(mtuPath); err == nil {
			if mtu, err := strconv.Atoi(strings.TrimSpace(string(mtuData))); err == nil {
				port.MTU = mtu
			}
		}

		// Get speed (requires interface to be up)
		speedPath := filepath.Join(netPath, ifName, "speed")
		if speedData, err := os.ReadFile(speedPath); err == nil {
			speedStr := strings.TrimSpace(string(speedData))
			if speedVal, err := strconv.Atoi(speedStr); err == nil {
				// Convert Mbps to Gbps string
				if speedVal >= 1000 {
					port.Speed = fmt.Sprintf("%dG", speedVal/1000)
				} else {
					port.Speed = fmt.Sprintf("%dM", speedVal)
				}
			}
		}

		ports = append(ports, port)
	}

	return ports, nil
}
