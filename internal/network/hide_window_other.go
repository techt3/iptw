//go:build !windows

package network

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
