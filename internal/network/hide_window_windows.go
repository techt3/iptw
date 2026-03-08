//go:build windows

package network

import (
	"os/exec"
	"syscall"
)

// hideWindow prevents the subprocess from showing a console window.
// Without this, netstat.exe flashes a CMD window every time it is called
// from a process built with -H windowsgui.
func hideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
