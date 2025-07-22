package network

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Connection represents a network connection
type Connection struct {
	RemoteIP   string
	RemotePort string
	LocalIP    string
	LocalPort  string
	Protocol   string
}

// Monitor monitors network connections
type Monitor struct {
	connections []Connection
}

// NewMonitor creates a new network monitor
func NewMonitor() *Monitor {
	return &Monitor{
		connections: make([]Connection, 0),
	}
}

// GetConnections returns current network connections
func (m *Monitor) GetConnections() []Connection {
	return m.connections
}

// GetSupportedPlatforms returns a list of supported operating systems
func GetSupportedPlatforms() []string {
	return []string{"darwin", "linux", "windows"}
}

// IsSupported checks if the current platform is supported
func IsSupported() bool {
	supportedPlatforms := GetSupportedPlatforms()
	for _, platform := range supportedPlatforms {
		if runtime.GOOS == platform {
			return true
		}
	}
	return false
}

// RefreshConnections updates the list of active connections (cross-platform)
func (m *Monitor) RefreshConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var connections []Connection
	var err error

	switch runtime.GOOS {
	case "darwin": // macOS
		connections, err = m.getConnectionsMacOS(ctx)
	case "linux":
		connections, err = m.getConnectionsLinux(ctx)
	case "windows":
		connections, err = m.getConnectionsWindows(ctx)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		return err
	}

	m.connections = connections
	return nil
}

// getConnectionsMacOS gets connections using netstat on macOS
func (m *Monitor) getConnectionsMacOS(ctx context.Context) ([]Connection, error) {
	cmd := exec.CommandContext(ctx, "netstat", "-an", "-f", "inet")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute netstat on macOS: %w", err)
	}

	connections := make([]Connection, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Regex to parse netstat output on macOS
	// Example: tcp4       0      0  192.168.1.100.50123    93.184.216.34.80       ESTABLISHED
	connRegex := regexp.MustCompile(`^(tcp4|udp4)\s+\d+\s+\d+\s+(\S+)\.(\d+)\s+(\S+)\.(\d+)\s+ESTABLISHED`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := connRegex.FindStringSubmatch(line)

		if len(matches) == 6 {
			protocol := matches[1]
			localIP := matches[2]
			localPort := matches[3]
			remoteIP := matches[4]
			remotePort := matches[5]

			if m.shouldIncludeConnection(remoteIP) {
				connections = append(connections, Connection{
					RemoteIP:   remoteIP,
					RemotePort: remotePort,
					LocalIP:    localIP,
					LocalPort:  localPort,
					Protocol:   protocol,
				})
			}
		}
	}

	return connections, nil
}

// getConnectionsLinux gets connections using ss on Linux
func (m *Monitor) getConnectionsLinux(ctx context.Context) ([]Connection, error) {
	// Try ss first (preferred on modern Linux)
	cmd := exec.CommandContext(ctx, "ss", "-tuln", "state", "established")
	output, err := cmd.Output()
	if err != nil {
		// Fall back to netstat if ss is not available
		return m.getConnectionsLinuxNetstat(ctx)
	}

	connections := make([]Connection, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Regex to parse ss output
	// Example: tcp   ESTAB  0      0      192.168.1.100:50123   93.184.216.34:80
	connRegex := regexp.MustCompile(`^(tcp|udp)\s+ESTAB\s+\d+\s+\d+\s+(\S+):(\d+)\s+(\S+):(\d+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := connRegex.FindStringSubmatch(line)

		if len(matches) == 6 {
			protocol := matches[1]
			localIP := matches[2]
			localPort := matches[3]
			remoteIP := matches[4]
			remotePort := matches[5]

			if m.shouldIncludeConnection(remoteIP) {
				connections = append(connections, Connection{
					RemoteIP:   remoteIP,
					RemotePort: remotePort,
					LocalIP:    localIP,
					LocalPort:  localPort,
					Protocol:   protocol,
				})
			}
		}
	}

	return connections, nil
}

// getConnectionsLinuxNetstat gets connections using netstat on Linux (fallback)
func (m *Monitor) getConnectionsLinuxNetstat(ctx context.Context) ([]Connection, error) {
	cmd := exec.CommandContext(ctx, "netstat", "-an", "--inet")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute netstat on Linux: %w", err)
	}

	connections := make([]Connection, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Regex to parse netstat output on Linux
	// Example: tcp        0      0 192.168.1.100:50123    93.184.216.34:80       ESTABLISHED
	connRegex := regexp.MustCompile(`^(tcp|udp)\s+\d+\s+\d+\s+(\S+):(\d+)\s+(\S+):(\d+)\s+ESTABLISHED`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := connRegex.FindStringSubmatch(line)

		if len(matches) == 6 {
			protocol := matches[1]
			localIP := matches[2]
			localPort := matches[3]
			remoteIP := matches[4]
			remotePort := matches[5]

			if m.shouldIncludeConnection(remoteIP) {
				connections = append(connections, Connection{
					RemoteIP:   remoteIP,
					RemotePort: remotePort,
					LocalIP:    localIP,
					LocalPort:  localPort,
					Protocol:   protocol,
				})
			}
		}
	}

	return connections, nil
}

// getConnectionsWindows gets connections using netstat on Windows
func (m *Monitor) getConnectionsWindows(ctx context.Context) ([]Connection, error) {
	cmd := exec.CommandContext(ctx, "netstat", "-an", "-p", "TCP")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute netstat on Windows: %w", err)
	}

	connections := make([]Connection, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Regex to parse netstat output on Windows
	// Example: TCP    192.168.1.100:50123    93.184.216.34:80       ESTABLISHED
	connRegex := regexp.MustCompile(`^\s*(TCP|UDP)\s+(\S+):(\d+)\s+(\S+):(\d+)\s+ESTABLISHED`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := connRegex.FindStringSubmatch(line)

		if len(matches) == 6 {
			protocol := strings.ToLower(matches[1])
			localIP := matches[2]
			localPort := matches[3]
			remoteIP := matches[4]
			remotePort := matches[5]

			if m.shouldIncludeConnection(remoteIP) {
				connections = append(connections, Connection{
					RemoteIP:   remoteIP,
					RemotePort: remotePort,
					LocalIP:    localIP,
					LocalPort:  localPort,
					Protocol:   protocol,
				})
			}
		}
	}

	return connections, nil
}

// shouldIncludeConnection determines if a connection should be included
func (m *Monitor) shouldIncludeConnection(remoteIP string) bool {
	// Skip localhost connections
	if strings.HasPrefix(remoteIP, "127.") || strings.HasPrefix(remoteIP, "::1") || remoteIP == "localhost" {
		return false
	}

	// Skip private network ranges
	if isPrivateIP(remoteIP) {
		return false
	}

	return true
}

// isPrivateIP checks if an IP address is in a private network range
func isPrivateIP(ipStr string) bool {
	// Handle common hostname cases
	if ipStr == "localhost" || ipStr == "0.0.0.0" {
		return true
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check for IPv4 private ranges
	privateRanges := []string{
		"10.0.0.0/8",     // 10.0.0.0 - 10.255.255.255
		"172.16.0.0/12",  // 172.16.0.0 - 172.31.255.255
		"192.168.0.0/16", // 192.168.0.0 - 192.168.255.255
		"169.254.0.0/16", // Link-local addresses (169.254.0.0 - 169.254.255.255)
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	// Check for IPv6 private ranges
	if ip.To4() == nil { // This is IPv6
		ipv6PrivateRanges := []string{
			"fc00::/7",  // Unique local addresses
			"fe80::/10", // Link-local addresses
			"::1/128",   // Loopback
		}

		for _, cidr := range ipv6PrivateRanges {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				return true
			}
		}
	}

	return false
}
