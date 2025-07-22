//go:build windows

package service

import (
	"fmt"
	"os/exec"
	"strings"
)

// installWindows installs the service as a Windows service
func (sm *ServiceManager) installWindows() error {
	// Use sc command to create service
	cmd := exec.Command("sc", "create", sm.ServiceName,
		"binPath=", fmt.Sprintf(`"%s" -force`, sm.ExecutablePath),
		"DisplayName=", sm.DisplayName,
		"start=", "auto",
		"obj=", "LocalSystem")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create Windows service: %w\nOutput: %s", err, string(output))
	}

	// Set service description
	cmd = exec.Command("sc", "description", sm.ServiceName, sm.Description)
	_ = cmd.Run() // Ignore errors for description

	// Set recovery options - restart on failure
	cmd = exec.Command("sc", "failure", sm.ServiceName, "reset=", "86400", "actions=", "restart/30000/restart/60000/restart/60000")
	_ = cmd.Run() // Ignore errors for recovery options

	fmt.Printf("✅ Service installed successfully on Windows\n")
	fmt.Printf("   Service Name: %s\n", sm.ServiceName)
	fmt.Printf("   Display Name: %s\n", sm.DisplayName)
	fmt.Printf("   Service will start automatically on boot\n")

	return nil
}

// uninstallWindows removes the Windows service
func (sm *ServiceManager) uninstallWindows() error {
	// Stop the service first
	_ = sm.stopWindows()

	// Delete the service
	cmd := exec.Command("sc", "delete", sm.ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		// Ignore "service does not exist" error
		if !strings.Contains(outputStr, "1060") && !strings.Contains(outputStr, "does not exist") {
			return fmt.Errorf("failed to delete Windows service: %w\nOutput: %s", err, outputStr)
		}
	}

	fmt.Printf("✅ Service uninstalled successfully from Windows\n")
	return nil
}

// startWindows starts the Windows service
func (sm *ServiceManager) startWindows() error {
	cmd := exec.Command("sc", "start", sm.ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		// Ignore "service already started" error
		if !strings.Contains(outputStr, "1056") && !strings.Contains(outputStr, "already been started") {
			return fmt.Errorf("failed to start Windows service: %w\nOutput: %s", err, outputStr)
		}
	}

	fmt.Printf("✅ Service started on Windows\n")
	return nil
}

// stopWindows stops the Windows service
func (sm *ServiceManager) stopWindows() error {
	cmd := exec.Command("sc", "stop", sm.ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		// Ignore "service not started" error
		if !strings.Contains(outputStr, "1062") && !strings.Contains(outputStr, "not been started") {
			return fmt.Errorf("failed to stop Windows service: %w\nOutput: %s", err, outputStr)
		}
	}

	fmt.Printf("✅ Service stopped on Windows\n")
	return nil
}

// statusWindows checks if the Windows service is running
func (sm *ServiceManager) statusWindows() (bool, error) {
	cmd := exec.Command("sc", "query", sm.ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Service doesn't exist or query failed
		return false, nil
	}

	outputStr := string(output)
	// Check if service is in RUNNING state
	return strings.Contains(outputStr, "RUNNING"), nil
}

// Stub implementations for other platforms on Windows
func (sm *ServiceManager) installMacOS() error {
	return fmt.Errorf("macOS service management not available on Windows")
}

func (sm *ServiceManager) uninstallMacOS() error {
	return fmt.Errorf("macOS service management not available on Windows")
}

func (sm *ServiceManager) startMacOS() error {
	return fmt.Errorf("macOS service management not available on Windows")
}

func (sm *ServiceManager) stopMacOS() error {
	return fmt.Errorf("macOS service management not available on Windows")
}

func (sm *ServiceManager) statusMacOS() (bool, error) {
	return false, fmt.Errorf("macOS service management not available on Windows")
}

func (sm *ServiceManager) installLinux() error {
	return fmt.Errorf("Linux service management not available on Windows")
}

func (sm *ServiceManager) uninstallLinux() error {
	return fmt.Errorf("Linux service management not available on Windows")
}

func (sm *ServiceManager) startLinux() error {
	return fmt.Errorf("Linux service management not available on Windows")
}

func (sm *ServiceManager) stopLinux() error {
	return fmt.Errorf("Linux service management not available on Windows")
}

func (sm *ServiceManager) statusLinux() (bool, error) {
	return false, fmt.Errorf("Linux service management not available on Windows")
}
