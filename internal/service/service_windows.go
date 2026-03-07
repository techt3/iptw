//go:build windows

package service

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// startupDir returns the path to the Windows Startup folder, using APPDATA env var
// with a fallback to constructing the path from the user's home directory.
func startupDir() (string, error) {
	appDataDir := os.Getenv("APPDATA")
	if appDataDir == "" {
		// Fall back to constructing the path from the user's home directory
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("APPDATA not set and failed to get current user: %w", err)
		}
		appDataDir = filepath.Join(currentUser.HomeDir, "AppData", "Roaming")
	}
	return filepath.Join(appDataDir, "Microsoft", "Windows", "Start Menu", "Programs", "Startup"), nil
}

// installWindows creates a shortcut in the Startup folder
func (sm *ServiceManager) installWindows() error {
	startupFolder, err := startupDir()
	if err != nil {
		return fmt.Errorf("failed to determine startup directory: %w", err)
	}

	if err := os.MkdirAll(startupFolder, 0755); err != nil {
		return fmt.Errorf("failed to ensure startup directory: %w", err)
	}

	// For a Go app without external dependencies, we can just create a batch file or symlink
	// A simple .bat file to launch the app
	batPath := filepath.Join(startupFolder, "iptw.bat")
	batContent := fmt.Sprintf("@echo off\nstart \"\" \"%s\"\n", sm.ExecutablePath)

	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("failed to create startup script: %w", err)
	}

	fmt.Printf("✅ Auto-start installed successfully on Windows\n")
	return nil
}

// uninstallWindows removes the startup script
func (sm *ServiceManager) uninstallWindows() error {
	startupFolder, err := startupDir()
	if err != nil {
		return fmt.Errorf("failed to determine startup directory: %w", err)
	}

	batPath := filepath.Join(startupFolder, "iptw.bat")

	if err := os.Remove(batPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove startup script: %w", err)
	}

	fmt.Printf("✅ Auto-start uninstalled successfully from Windows\n")
	return nil
}

// startWindows shows error message
func (sm *ServiceManager) startWindows() error {
	return fmt.Errorf("manual start from service manager not supported. Run './iptw' directly")
}

// stopWindows shows error message
func (sm *ServiceManager) stopWindows() error {
	return fmt.Errorf("manual stop from service manager not supported")
}

// statusWindows
func (sm *ServiceManager) statusWindows() (bool, error) {
	return false, fmt.Errorf("service status not supported for user-space app")
}

// Stub implementations for other platforms on Windows
func (sm *ServiceManager) installMacOS() error {
	return fmt.Errorf("macos service management not available on Windows")
}

func (sm *ServiceManager) uninstallMacOS() error {
	return fmt.Errorf("macos service management not available on Windows")
}

func (sm *ServiceManager) startMacOS() error {
	return fmt.Errorf("macos service management not available on Windows")
}

func (sm *ServiceManager) stopMacOS() error {
	return fmt.Errorf("macos service management not available on Windows")
}

func (sm *ServiceManager) statusMacOS() (bool, error) {
	return false, fmt.Errorf("macos service management not available on Windows")
}

func (sm *ServiceManager) installLinux() error {
	return fmt.Errorf("linux service management not available on Windows")
}

func (sm *ServiceManager) uninstallLinux() error {
	return fmt.Errorf("linux service management not available on Windows")
}

func (sm *ServiceManager) startLinux() error {
	return fmt.Errorf("linux service management not available on Windows")
}

func (sm *ServiceManager) stopLinux() error {
	return fmt.Errorf("linux service management not available on Windows")
}

func (sm *ServiceManager) statusLinux() (bool, error) {
	return false, fmt.Errorf("linux service management not available on Windows")
}
