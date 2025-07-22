//go:build windows

package singleton

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32     = syscall.MustLoadDLL("kernel32.dll")
	lockFileEx   = kernel32.MustFindProc("LockFileEx")
	unlockFileEx = kernel32.MustFindProc("UnlockFileEx")
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
)

// acquireFileLock applies an exclusive lock to the file on Windows
func (l *Lock) acquireFileLock(file *os.File) error {
	// Use Windows LockFileEx API for proper file locking
	handle := syscall.Handle(file.Fd())

	var overlapped syscall.Overlapped

	r1, _, err := lockFileEx.Call(
		uintptr(handle),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		uintptr(0),
		uintptr(1), // Lock 1 byte
		uintptr(0),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if r1 == 0 {
		return err
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is running on Windows
func (l *Lock) isProcessRunning(pid int) bool {
	// On Windows, we need to use a different approach
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Try to get the process state (this will fail if process doesn't exist)
	_, err = process.Wait()
	if err != nil {
		// If Wait() returns an error, the process might still be running
		// This is a simplified check
		return true
	}
	return false
}
