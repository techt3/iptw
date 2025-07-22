//go:build !darwin && !linux && !windows

package service

import "fmt"

// Stub implementations for unsupported platforms

func (sm *ServiceManager) installMacOS() error {
	return fmt.Errorf("macOS service management not supported on this platform")
}

func (sm *ServiceManager) uninstallMacOS() error {
	return fmt.Errorf("macOS service management not supported on this platform")
}

func (sm *ServiceManager) startMacOS() error {
	return fmt.Errorf("macOS service management not supported on this platform")
}

func (sm *ServiceManager) stopMacOS() error {
	return fmt.Errorf("macOS service management not supported on this platform")
}

func (sm *ServiceManager) statusMacOS() (bool, error) {
	return false, fmt.Errorf("macOS service management not supported on this platform")
}

func (sm *ServiceManager) installLinux() error {
	return fmt.Errorf("Linux service management not supported on this platform")
}

func (sm *ServiceManager) uninstallLinux() error {
	return fmt.Errorf("Linux service management not supported on this platform")
}

func (sm *ServiceManager) startLinux() error {
	return fmt.Errorf("Linux service management not supported on this platform")
}

func (sm *ServiceManager) stopLinux() error {
	return fmt.Errorf("Linux service management not supported on this platform")
}

func (sm *ServiceManager) statusLinux() (bool, error) {
	return false, fmt.Errorf("Linux service management not supported on this platform")
}

func (sm *ServiceManager) installWindows() error {
	return fmt.Errorf("Windows service management not supported on this platform")
}

func (sm *ServiceManager) uninstallWindows() error {
	return fmt.Errorf("Windows service management not supported on this platform")
}

func (sm *ServiceManager) startWindows() error {
	return fmt.Errorf("Windows service management not supported on this platform")
}

func (sm *ServiceManager) stopWindows() error {
	return fmt.Errorf("Windows service management not supported on this platform")
}

func (sm *ServiceManager) statusWindows() (bool, error) {
	return false, fmt.Errorf("Windows service management not supported on this platform")
}
