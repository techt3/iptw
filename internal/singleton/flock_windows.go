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
	lockfileExclusiveLock          = 0x00000002
	lockfileFailImmediately        = 0x00000001
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259 // STILL_ACTIVE / STATUS_PENDING
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

// isProcessRunning checks if a process with the given PID is running on Windows.
// os.Process.Wait() cannot be used here because it only works for child processes
// spawned by this process; calling it on an unrelated PID always returns an error,
// which would make every stale lock look like a live process.
// Instead we use OpenProcess + GetExitCodeProcess: if the exit code is STILL_ACTIVE
// (259 / STATUS_PENDING) the process is genuinely alive.
func (l *Lock) isProcessRunning(pid int) bool {
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		// Process does not exist or we lack permission to query it.
		return false
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == stillActive
}
