//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// installMacOS installs the service as a macOS LaunchAgent
func (sm *ServiceManager) installMacOS() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Create LaunchAgents directory if it doesn't exist
	launchAgentsDir := filepath.Join(currentUser.HomeDir, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Create plist file
	plistPath := filepath.Join(launchAgentsDir, fmt.Sprintf("com.%s.plist", sm.ServiceName))
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>-force</string>
        <string>-port</string>
        <string>%s</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>%s/Library/Logs/%s.out.log</string>
    <key>StandardErrorPath</key>
    <string>%s/Library/Logs/%s.err.log</string>
    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>`, sm.ServiceName, sm.ExecutablePath, sm.ServerPort, sm.WorkingDir,
		currentUser.HomeDir, sm.ServiceName,
		currentUser.HomeDir, sm.ServiceName)

	// Write plist file
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the service
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load service with launchctl: %w", err)
	}

	fmt.Printf("✅ Service installed successfully on macOS\n")
	fmt.Printf("   Plist file: %s\n", plistPath)
	fmt.Printf("   Service will start automatically on login\n")
	fmt.Printf("   HTTP statistics server will be available on port %s\n", sm.ServerPort)

	return nil
}

// uninstallMacOS removes the macOS LaunchAgent
func (sm *ServiceManager) uninstallMacOS() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	plistPath := filepath.Join(currentUser.HomeDir, "Library", "LaunchAgents", fmt.Sprintf("com.%s.plist", sm.ServiceName))

	// Unload the service
	cmd := exec.Command("launchctl", "unload", plistPath)
	_ = cmd.Run() // Ignore errors as service might not be loaded

	// Remove plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	fmt.Printf("✅ Service uninstalled successfully from macOS\n")
	return nil
}

// startMacOS starts the macOS LaunchAgent
func (sm *ServiceManager) startMacOS() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	plistPath := filepath.Join(currentUser.HomeDir, "Library", "LaunchAgents", fmt.Sprintf("com.%s.plist", sm.ServiceName))

	// Check if plist file exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return fmt.Errorf("service not installed. Run with -install-service first")
	}

	// Load the service (this also starts it)
	cmd := exec.Command("launchctl", "load", plistPath)
	if err := cmd.Run(); err != nil {
		// If load fails, try start command as fallback
		startCmd := exec.Command("launchctl", "start", fmt.Sprintf("com.%s", sm.ServiceName))
		if startErr := startCmd.Run(); startErr != nil {
			return fmt.Errorf("failed to start service (both load and start failed): load error: %v, start error: %v", err, startErr)
		}
	}

	fmt.Printf("✅ Service started on macOS\n")
	return nil
}

// stopMacOS stops the macOS LaunchAgent
func (sm *ServiceManager) stopMacOS() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	plistPath := filepath.Join(currentUser.HomeDir, "Library", "LaunchAgents", fmt.Sprintf("com.%s.plist", sm.ServiceName))

	// Use unload to stop and prevent restart
	cmd := exec.Command("launchctl", "unload", plistPath)
	if err := cmd.Run(); err != nil {
		// Try the stop command as fallback (though it's less effective)
		stopCmd := exec.Command("launchctl", "stop", fmt.Sprintf("com.%s", sm.ServiceName))
		if stopErr := stopCmd.Run(); stopErr != nil {
			return fmt.Errorf("failed to stop service (both unload and stop failed): unload error: %v, stop error: %v", err, stopErr)
		}
	}

	fmt.Printf("✅ Service stopped on macOS\n")
	return nil
}

// statusMacOS checks if the macOS LaunchAgent is running
func (sm *ServiceManager) statusMacOS() (bool, error) {
	cmd := exec.Command("launchctl", "list", fmt.Sprintf("com.%s", sm.ServiceName))
	err := cmd.Run()
	if err != nil {
		// Service is not loaded
		return false, nil
	}
	return true, nil
}

// Stub implementations for other platforms on macOS
func (sm *ServiceManager) installLinux() error {
	return fmt.Errorf("Linux service management not available on macOS")
}

func (sm *ServiceManager) uninstallLinux() error {
	return fmt.Errorf("Linux service management not available on macOS")
}

func (sm *ServiceManager) startLinux() error {
	return fmt.Errorf("Linux service management not available on macOS")
}

func (sm *ServiceManager) stopLinux() error {
	return fmt.Errorf("Linux service management not available on macOS")
}

func (sm *ServiceManager) statusLinux() (bool, error) {
	return false, fmt.Errorf("Linux service management not available on macOS")
}

func (sm *ServiceManager) installWindows() error {
	return fmt.Errorf("Windows service management not available on macOS")
}

func (sm *ServiceManager) uninstallWindows() error {
	return fmt.Errorf("Windows service management not available on macOS")
}

func (sm *ServiceManager) startWindows() error {
	return fmt.Errorf("Windows service management not available on macOS")
}

func (sm *ServiceManager) stopWindows() error {
	return fmt.Errorf("Windows service management not available on macOS")
}

func (sm *ServiceManager) statusWindows() (bool, error) {
	return false, fmt.Errorf("Windows service management not available on macOS")
}
