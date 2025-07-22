//go:build darwin || linux

package singleton

import (
	"os"
	"syscall"
)

// acquireFileLock applies an exclusive lock to the file on Unix systems
func (l *Lock) acquireFileLock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// isProcessRunning checks if a process with the given PID is running on Unix systems
func (l *Lock) isProcessRunning(pid int) bool {
	// Try to send signal 0 to the process (doesn't actually send a signal, just checks if process exists)
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, we can use Signal(0) to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}
	return true
}
