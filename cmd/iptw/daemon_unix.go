//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// maybeDaemonize re-executes the process detached from the controlling
// terminal so the user can close the launching shell once the tray icon
// appears.  It is a no-op when --foreground is passed.
//
// How it works:
//  1. The original (parent) process re-execs itself with all the same
//     arguments plus "--foreground", using Setsid=true so the child gets a
//     new session and is no longer attached to the terminal.
//  2. The parent then exits with code 0, freeing the terminal.
//  3. The child (which has --foreground set) skips this function and runs
//     the application normally.
func maybeDaemonize(foreground bool) {
	if foreground {
		return
	}

	exe, err := os.Executable()
	if err != nil {
		// Cannot determine the executable path; just run in the foreground.
		return
	}

	// Forward every original argument and add --foreground so the child does
	// not daemonize again.
	args := make([]string, 0, len(os.Args)-1+1)
	args = append(args, os.Args[1:]...)
	args = append(args, "--foreground")

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	// Disconnect all standard streams so the child has no reference to the
	// parent terminal.
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		// If we cannot spawn the background child, fall through and run
		// normally in the foreground rather than failing silently.
		return
	}

	// Parent exits, releasing the terminal.
	os.Exit(0)
}
