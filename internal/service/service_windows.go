//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

// installWindows installs the service as a Windows service
func (sm *ServiceManager) installWindows() error {
	// Validate executable path exists
	if _, err := os.Stat(sm.ExecutablePath); os.IsNotExist(err) {
		return fmt.Errorf("executable not found at %s", sm.ExecutablePath)
	}

	// Get current user to run service in user context (needed for desktop access)
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Format user for Windows service (DOMAIN\username or .\username for local)
	serviceUser := currentUser.Username
	if currentUser.Username != "" {
		// Use local user format if no domain
		if !strings.Contains(currentUser.Username, "\\") {
			serviceUser = ".\\" + currentUser.Username
		}
	}

	// Use sc command to create service running as current user
	cmd := exec.Command("sc", "create", sm.ServiceName,
		"binPath="+fmt.Sprintf(`"%s" -force -server -port %s`, sm.ExecutablePath, sm.ServerPort),
		"DisplayName="+sm.DisplayName,
		"start=auto",
		"obj="+serviceUser,
		"type=own")

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		// Check if service already exists
		if strings.Contains(outputStr, "1073") || strings.Contains(outputStr, "already exists") {
			fmt.Printf("⚠️  Service already exists on Windows\n")
			return nil
		}
		return fmt.Errorf("failed to create Windows service: %w\nOutput: %s", err, outputStr)
	}

	// Set service description
	cmd = exec.Command("sc", "description", sm.ServiceName, sm.Description)
	_ = cmd.Run() // Ignore errors for description

	// Set recovery options - restart on failure
	cmd = exec.Command("sc", "failure", sm.ServiceName, "reset=86400", "actions=restart/30000/restart/60000/restart/60000")
	_ = cmd.Run() // Ignore errors for recovery options

	fmt.Printf("✅ Service installed successfully on Windows\n")
	fmt.Printf("   Service Name: %s\n", sm.ServiceName)
	fmt.Printf("   Display Name: %s\n", sm.DisplayName)
	fmt.Printf("   Service User: %s (desktop access enabled)\n", serviceUser)
	fmt.Printf("   Service will start automatically on boot\n")
	fmt.Printf("   HTTP statistics server will be available on port %s\n", sm.ServerPort)
	fmt.Printf("   Note: Service requires user to be logged in for desktop wallpaper changes\n")
	fmt.Printf("   Note: Windows may prompt for user password when starting the service\n")

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
	// First check if service exists
	statusCmd := exec.Command("sc", "query", sm.ServiceName)
	statusOutput, statusErr := statusCmd.CombinedOutput()
	if statusErr != nil {
		return fmt.Errorf("service '%s' does not exist or cannot be queried: %w\nOutput: %s",
			sm.ServiceName, statusErr, string(statusOutput))
	}

	// Check current status
	statusStr := string(statusOutput)
	if strings.Contains(statusStr, "RUNNING") {
		fmt.Printf("✅ Service is already running on Windows\n")
		return nil
	}

	// Try to start the service
	cmd := exec.Command("sc", "start", sm.ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		// Ignore "service already started" error
		if !strings.Contains(outputStr, "1056") && !strings.Contains(outputStr, "already been started") {
			return fmt.Errorf("failed to start Windows service: %w\nOutput: %s\nService Status: %s",
				err, outputStr, statusStr)
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
		return false, fmt.Errorf("service query failed: %w\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Check various service states
	if strings.Contains(outputStr, "RUNNING") {
		return true, nil
	}

	// If not running, provide more detailed status
	if strings.Contains(outputStr, "STOPPED") {
		return false, nil
	}

	if strings.Contains(outputStr, "START_PENDING") {
		return false, fmt.Errorf("service is starting up (START_PENDING)")
	}

	if strings.Contains(outputStr, "STOP_PENDING") {
		return false, fmt.Errorf("service is shutting down (STOP_PENDING)")
	}

	// Return false with detailed output for any other status
	return false, fmt.Errorf("service status unknown: %s", outputStr)
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
