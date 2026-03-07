//go:build linux

package service

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// installLinux creates a .desktop file in ~/.config/autostart
func (sm *ServiceManager) installLinux() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	autostartDir := filepath.Join(currentUser.HomeDir, ".config", "autostart")
	if err := os.MkdirAll(autostartDir, 0755); err != nil {
		return fmt.Errorf("failed to create autostart directory: %w", err)
	}

	desktopFile := filepath.Join(autostartDir, fmt.Sprintf("%s.desktop", sm.ServiceName))
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Exec="%s"
Hidden=false
NoDisplay=false
Terminal=false
X-GNOME-Autostart-enabled=true
Name=%s
Comment=%s
`, sm.ExecutablePath, sm.DisplayName, sm.Description)

	if err := os.WriteFile(desktopFile, []byte(desktopContent), 0644); err != nil {
		return fmt.Errorf("failed to write autostart file: %w", err)
	}

	fmt.Printf("✅ Auto-start installed successfully on Linux\n")
	return nil
}

// uninstallLinux removes the .desktop file
func (sm *ServiceManager) uninstallLinux() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	desktopFile := filepath.Join(currentUser.HomeDir, ".config", "autostart", fmt.Sprintf("%s.desktop", sm.ServiceName))

	if err := os.Remove(desktopFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove autostart file: %w", err)
	}

	fmt.Printf("✅ Auto-start uninstalled successfully from Linux\n")
	return nil
}

// startLinux
func (sm *ServiceManager) startLinux() error {
	return fmt.Errorf("manual start from service manager not supported. Run './iptw' directly")
}

// stopLinux
func (sm *ServiceManager) stopLinux() error {
	return fmt.Errorf("manual stop from service manager not supported")
}

// statusLinux
func (sm *ServiceManager) statusLinux() (bool, error) {
	return false, fmt.Errorf("service status not supported for user-space app")
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
	return false, fmt.Errorf("macos service management not available on Linux")
}

func (sm *ServiceManager) installWindows() error {
	return fmt.Errorf("windows service management not available on Linux")
}

func (sm *ServiceManager) uninstallWindows() error {
	return fmt.Errorf("windows service management not available on Linux")
}

func (sm *ServiceManager) startWindows() error {
	return fmt.Errorf("windows service management not available on Linux")
}

func (sm *ServiceManager) stopWindows() error {
	return fmt.Errorf("windows service management not available on Linux")
}

func (sm *ServiceManager) statusWindows() (bool, error) {
	return false, fmt.Errorf("windows service management not available on Linux")
}
