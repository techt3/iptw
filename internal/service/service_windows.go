//go:build windows

package service

import "fmt"

// Windows service functionality is disabled - wallpaper changes don't work properly in service mode
// Users should run the application directly instead of as a service

// installWindows shows an error message explaining that Windows service is not supported
func (sm *ServiceManager) installWindows() error {
	fmt.Println("❌ Windows service installation is not supported.")
	fmt.Println()
	fmt.Println("🖼️  REASON: Windows services cannot change desktop wallpapers due to session isolation.")
	fmt.Println("   Services run in a different session than the user desktop, preventing")
	fmt.Println("   wallpaper changes and other desktop interactions.")
	fmt.Println()
	fmt.Println("💡 ALTERNATIVE: Run IPTW directly as a regular application:")
	fmt.Println("   ./iptw                    # Run in foreground")
	fmt.Println("   ./iptw -server            # Run with HTTP server")
	fmt.Println()
	fmt.Println("🚀 TIP: Add to Windows startup folder for automatic startup:")
	fmt.Println("   %APPDATA%\\Microsoft\\Windows\\Start Menu\\Programs\\Startup")

	return fmt.Errorf("Windows service installation is not supported")
}

// uninstallWindows shows message that there's nothing to uninstall
func (sm *ServiceManager) uninstallWindows() error {
	fmt.Println("ℹ️  No Windows service to uninstall - service functionality is disabled on Windows.")
	return nil
}

// startWindows shows error message
func (sm *ServiceManager) startWindows() error {
	return fmt.Errorf("Windows service functionality is disabled - run './iptw' directly instead")
}

// stopWindows shows error message
func (sm *ServiceManager) stopWindows() error {
	return fmt.Errorf("Windows service functionality is disabled - no service to stop")
}

// statusWindows shows that no service exists
func (sm *ServiceManager) statusWindows() (bool, error) {
	return false, fmt.Errorf("Windows service functionality is disabled")
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
