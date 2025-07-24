//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// installLinux installs the service as a systemd user service
func (sm *ServiceManager) installLinux() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Create systemd user directory if it doesn't exist
	systemdDir := filepath.Join(currentUser.HomeDir, ".config", "systemd", "user")
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	// Create service file
	servicePath := filepath.Join(systemdDir, fmt.Sprintf("%s.service", sm.ServiceName))
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=graphical-session.target

[Service]
Type=simple
ExecStart=%s -force -port %s
WorkingDirectory=%s
Restart=always
RestartSec=10
KillMode=process
TimeoutStopSec=20

[Install]
WantedBy=default.target`, sm.Description, sm.ExecutablePath, sm.ServerPort, sm.WorkingDir)

	// Write service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd daemon
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable the service
	cmd = exec.Command("systemctl", "--user", "enable", fmt.Sprintf("%s.service", sm.ServiceName))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Printf("✅ Service installed successfully on Linux\n")
	fmt.Printf("   Service file: %s\n", servicePath)
	fmt.Printf("   Service will start automatically on login\n")
	fmt.Printf("   HTTP statistics server will be available on port %s\n", sm.ServerPort)
	fmt.Printf("   To enable lingering (start without login): sudo loginctl enable-linger %s\n", currentUser.Username)

	return nil
}

// uninstallLinux removes the systemd user service
func (sm *ServiceManager) uninstallLinux() error {
	// Stop the service first
	_ = sm.stopLinux()

	// Disable the service
	cmd := exec.Command("systemctl", "--user", "disable", fmt.Sprintf("%s.service", sm.ServiceName))
	_ = cmd.Run() // Ignore errors as service might not be enabled

	// Remove service file
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	servicePath := filepath.Join(currentUser.HomeDir, ".config", "systemd", "user", fmt.Sprintf("%s.service", sm.ServiceName))
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd daemon
	cmd = exec.Command("systemctl", "--user", "daemon-reload")
	_ = cmd.Run()

	fmt.Printf("✅ Service uninstalled successfully from Linux\n")
	return nil
}

// startLinux starts the systemd user service
func (sm *ServiceManager) startLinux() error {
	cmd := exec.Command("systemctl", "--user", "start", fmt.Sprintf("%s.service", sm.ServiceName))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Printf("✅ Service started on Linux\n")
	return nil
}

// stopLinux stops the systemd user service
func (sm *ServiceManager) stopLinux() error {
	cmd := exec.Command("systemctl", "--user", "stop", fmt.Sprintf("%s.service", sm.ServiceName))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Printf("✅ Service stopped on Linux\n")
	return nil
}

// statusLinux checks if the systemd user service is running
func (sm *ServiceManager) statusLinux() (bool, error) {
	cmd := exec.Command("systemctl", "--user", "is-active", fmt.Sprintf("%s.service", sm.ServiceName))
	err := cmd.Run()
	if err != nil {
		// Service is not active
		return false, nil
	}
	return true, nil
}

// Stub implementations for other platforms on Linux
func (sm *ServiceManager) installMacOS() error {
	return fmt.Errorf("macOS service management not available on Linux")
}

func (sm *ServiceManager) uninstallMacOS() error {
	return fmt.Errorf("macOS service management not available on Linux")
}

func (sm *ServiceManager) startMacOS() error {
	return fmt.Errorf("macOS service management not available on Linux")
}

func (sm *ServiceManager) stopMacOS() error {
	return fmt.Errorf("macOS service management not available on Linux")
}

func (sm *ServiceManager) statusMacOS() (bool, error) {
	return false, fmt.Errorf("macOS service management not available on Linux")
}

func (sm *ServiceManager) installWindows() error {
	return fmt.Errorf("Windows service management not available on Linux")
}

func (sm *ServiceManager) uninstallWindows() error {
	return fmt.Errorf("Windows service management not available on Linux")
}

func (sm *ServiceManager) startWindows() error {
	return fmt.Errorf("Windows service management not available on Linux")
}

func (sm *ServiceManager) stopWindows() error {
	return fmt.Errorf("Windows service management not available on Linux")
}

func (sm *ServiceManager) statusWindows() (bool, error) {
	return false, fmt.Errorf("Windows service management not available on Linux")
}
