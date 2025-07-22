// Package singleton provides utilities to ensure only one instance of the application runs
package singleton

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// Lock represents a singleton lock
type Lock struct {
	lockFile string
	file     *os.File
}

// NewLock creates a new singleton lock
// lockName should be a unique identifier for your application (e.g., "iptw")
func NewLock(lockName string) (*Lock, error) {
	// Get the appropriate lock directory based on OS
	lockDir, err := getLockDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get lock directory: %w", err)
	}

	// Ensure lock directory exists
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	lockFile := filepath.Join(lockDir, fmt.Sprintf("%s.lock", lockName))

	return &Lock{
		lockFile: lockFile,
	}, nil
}

// CleanupStaleLock removes a stale lock file if the process is no longer running
func (l *Lock) CleanupStaleLock() error {
	// Check if lock file exists
	data, err := os.ReadFile(l.lockFile)
	if err != nil {
		// File doesn't exist, nothing to clean up
		return nil
	}

	// Parse PID from file
	pidStr := string(data)
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// Invalid PID, remove the stale file
		return os.Remove(l.lockFile)
	}

	// Check if process with this PID is still running
	if !l.isProcessRunning(pid) {
		// Process is not running, remove the stale lock file
		return os.Remove(l.lockFile)
	}

	return fmt.Errorf("lock file exists and process %d is still running", pid)
}

// Acquire attempts to acquire the singleton lock
// Returns an error if another instance is already running
func (l *Lock) Acquire() error {
	// Check if lock file exists and if the process is still running
	if l.isAnotherInstanceRunning() {
		// Try to cleanup stale lock first
		if err := l.CleanupStaleLock(); err != nil {
			return fmt.Errorf("another instance of the application is already running")
		}
		// If cleanup succeeded, the lock file was stale, so we can proceed
	}

	// Create/open the lock file
	file, err := os.OpenFile(l.lockFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire an exclusive lock on the file
	if err := l.acquireFileLock(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Write our PID to the lock file
	pid := os.Getpid()
	if _, err := file.WriteString(strconv.Itoa(pid)); err != nil {
		file.Close()
		return fmt.Errorf("failed to write PID to lock file: %w", err)
	}

	// Sync to ensure PID is written to disk
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("failed to sync lock file: %w", err)
	}

	l.file = file
	return nil
}

// Release releases the singleton lock
func (l *Lock) Release() error {
	if l.file != nil {
		// Close the file (this also releases the lock)
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("failed to close lock file: %w", err)
		}
		l.file = nil
	}

	// Remove the lock file
	if err := os.Remove(l.lockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	return nil
}

// isAnotherInstanceRunning checks if another instance is already running
func (l *Lock) isAnotherInstanceRunning() bool {
	// Check if lock file exists
	data, err := os.ReadFile(l.lockFile)
	if err != nil {
		// File doesn't exist or can't be read, so no other instance
		return false
	}

	// Parse PID from file
	pidStr := string(data)
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// Invalid PID, treat as stale lock file
		return false
	}

	// Check if process with this PID is still running
	return l.isProcessRunning(pid)
}

// isProcessRunning checks if a process with the given PID is running
// Implementation is platform-specific

// acquireFileLock applies an exclusive lock to the file
// Implementation is platform-specific (see flock_*.go files)

// getLockDirectory returns the appropriate directory for lock files based on OS
func getLockDirectory() (string, error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		// Use /tmp for Unix-like systems, or user's cache directory
		if cacheDir, err := os.UserCacheDir(); err == nil {
			return filepath.Join(cacheDir, "iptw"), nil
		}
		return "/tmp/iptw", nil
	case "windows":
		// Use TEMP directory on Windows
		if tempDir := os.Getenv("TEMP"); tempDir != "" {
			return filepath.Join(tempDir, "iptw"), nil
		}
		if tempDir := os.Getenv("TMP"); tempDir != "" {
			return filepath.Join(tempDir, "iptw"), nil
		}
		return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Temp", "iptw"), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
