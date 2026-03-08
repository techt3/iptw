//go:build windows

package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// SetupWindowsFileLogger opens (or creates) the log file and registers it as
// the extra writer so that all subsequent SetupLogger calls include file output.
// Returns a closer for the log file that the caller should defer-close.
//
// NOTE: os.Stdout is intentionally excluded from the writer. When built with
// -H windowsgui there is no attached console and os.Stdout.Write returns an
// error. io.MultiWriter stops on the first error, so putting os.Stdout first
// would silently drop every log line to the file.
func SetupWindowsFileLogger(level slog.Level) io.Closer {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	logDir := filepath.Join(appData, "iptw")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil
	}
	logFile, err := os.OpenFile(
		filepath.Join(logDir, "iptw.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644,
	)
	if err != nil {
		return nil
	}
	// Write a plain-text marker immediately — before slog is involved — so the
	// file is never empty even if slog setup itself fails for any reason.
	fmt.Fprintf(logFile, "--- IPTW startup %s ---\n", time.Now().Format("2006-01-02 15:04:05"))

	// Register the file as the sole extra writer (no os.Stdout).
	SetExtraWriter(logFile)
	// Apply immediately with the bootstrap level (before config is loaded).
	SetupLogger(level.String())
	return logFile
}
