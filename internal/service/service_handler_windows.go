//go:build windows

package service

import "fmt"

// IsRunningAsWindowsService always returns false - Windows service mode is disabled
func IsRunningAsWindowsService() bool {
	return false
}

// RunAsWindowsService returns an error - Windows service mode is disabled
func RunAsWindowsService(serviceName, serverPort string) error {
	return fmt.Errorf("Windows service functionality is disabled - run the application directly for proper wallpaper support")
}
