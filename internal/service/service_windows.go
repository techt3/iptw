//go:build windows

package service

import (
"fmt"
"os"
"path/filepath"
"os/user"
)

// installWindows creates a shortcut in the Startup folder
func (sm *ServiceManager) installWindows() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	startupDir := filepath.Join(currentUser.HomeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	
	if err := os.MkdirAll(startupDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure startup directory: %w", err)
	}

	// For a Go app without external dependencies, we can just create a batch file or symlink
	// A simple .bat file to launch the app
	batPath := filepath.Join(startupDir, "iptw.bat")
	batContent := fmt.Sprintf("@echo off\nstart \"\" \"%s\"\n", sm.ExecutablePath)

	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("failed to create startup script: %w", err)
	}

	fmt.Printf("✅ Auto-start installed successfully on Windows\n")
	return nil
}

// uninstallWindows removes the startup script
func (sm *ServiceManager) uninstallWindows() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	batPath := filepath.Join(currentUser.HomeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "iptw.bat")

	if err := os.Remove(batPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove startup script: %w", err)
	}

	fmt.Printf("✅ Auto-start uninstalled successfully from Windows\n")
	return nil
}

// startWindows shows error message
func (sm *ServiceManager) startWindows() error {
	return fmt.Errorf("Manual start from service manager not supported. Run './iptw' directly")
}

// stopWindows shows error message
func (sm *ServiceManager) stopWindows() error {
	return fmt.Errorf("Manual stop from service manager not supported.")
}

// statusWindows
func (sm *ServiceManager) statusWindows() (bool, error) {
	return false, fmt.Errorf("Service status not supported for user-space app")
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
