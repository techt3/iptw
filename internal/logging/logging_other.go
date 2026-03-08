//go:build !windows

package logging

import (
	"io"
	"log/slog"
)

// SetupWindowsFileLogger is a no-op on non-Windows platforms.
func SetupWindowsFileLogger(_ slog.Level) io.Closer { return nil }
