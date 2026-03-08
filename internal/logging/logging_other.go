//go:build !windows

package logging

import "io"

// SetupWindowsFileLogger is a no-op on non-Windows platforms.
func SetupWindowsFileLogger(_ interface{}) io.Closer { return nil }
