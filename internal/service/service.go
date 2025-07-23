// Package service provides cross-platform background service management
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ServiceManager handles service installation/uninstallation across platforms
type ServiceManager struct {
	ServiceName    string
	DisplayName    string
	Description    string
	ExecutablePath string
	WorkingDir     string
	ServerPort     string
}

// NewServiceManager creates a new service manager instance
func NewServiceManager() (*ServiceManager, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	workingDir := filepath.Dir(execPath)

	return &ServiceManager{
		ServiceName:    "iptw",
		DisplayName:    "IP Travel Wallpaper Server",
		Description:    "IP Travel Wallpaper background service with HTTP statistics server for dynamic desktop wallpapers based on network connections",
		ExecutablePath: execPath,
		WorkingDir:     workingDir,
		ServerPort:     "32782", // Default server port
	}, nil
}

// NewServiceManagerWithPort creates a new service manager instance with custom port
func NewServiceManagerWithPort(port string) (*ServiceManager, error) {
	sm, err := NewServiceManager()
	if err != nil {
		return nil, err
	}
	sm.ServerPort = port
	return sm, nil
}

// Install installs the service on the current platform
func (sm *ServiceManager) Install() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.installMacOS()
	case "windows":
		return sm.installWindows()
	case "linux":
		return sm.installLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Uninstall removes the service from the current platform
func (sm *ServiceManager) Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.uninstallMacOS()
	case "windows":
		return sm.uninstallWindows()
	case "linux":
		return sm.uninstallLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Start starts the installed service
func (sm *ServiceManager) Start() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.startMacOS()
	case "windows":
		return sm.startWindows()
	case "linux":
		return sm.startLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Stop stops the running service
func (sm *ServiceManager) Stop() error {
	switch runtime.GOOS {
	case "darwin":
		return sm.stopMacOS()
	case "windows":
		return sm.stopWindows()
	case "linux":
		return sm.stopLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Status checks if the service is running
func (sm *ServiceManager) Status() (bool, error) {
	switch runtime.GOOS {
	case "darwin":
		return sm.statusMacOS()
	case "windows":
		return sm.statusWindows()
	case "linux":
		return sm.statusLinux()
	default:
		return false, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
