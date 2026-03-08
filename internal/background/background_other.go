//go:build !windows

package background

import (
	"fmt"
	"os"
)

func setWindowsBackground(_ string) error {
	return fmt.Errorf("setWindowsBackground called on non-Windows platform")
}

func getWindowsCurrentWallpaper() (string, error) {
	return "", fmt.Errorf("getWindowsCurrentWallpaper called on non-Windows platform")
}

// RestoreWallpaper restores the wallpaper from a backup file using SetDesktopBackground.
func RestoreWallpaper(backupPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup wallpaper file does not exist: %s", backupPath)
	}
	if err := SetDesktopBackground(backupPath); err != nil {
		return fmt.Errorf("failed to restore wallpaper: %w", err)
	}
	return nil
}
