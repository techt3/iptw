//go:build !windows

package service

import "fmt"

// IsRunningAsWindowsService always returns false on non-Windows platforms
func IsRunningAsWindowsService() bool {
	return false
}

// RunAsWindowsService returns an error on non-Windows platforms
func RunAsWindowsService(serviceName, serverPort string) error {
	return fmt.Errorf("Windows service functionality not available on this platform")
}
