// Package background provides desktop background management utilities
package background

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// SetDesktopBackground sets an image as the desktop background
// This implementation is macOS-specific using osascript
// Creates a timestamped copy to force wallpaper refresh
func SetDesktopBackground(imagePath string) error {
	// Check if the image file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("image file does not exist: %s", imagePath)
	}

	// Create a timestamped copy to force refresh
	timestampedPath, err := createTimestampedCopy(imagePath)
	if err != nil {
		return fmt.Errorf("failed to create timestamped copy: %w", err)
	}

	// Clean up the timestamped copy after setting the background
	defer func() {
		if removeErr := os.Remove(timestampedPath); removeErr != nil {
			slog.Error("⚠️  Failed to clean up timestamped file %s: %v", timestampedPath, removeErr)
		}
	}()

	// Get absolute path
	absPath, err := filepath.Abs(timestampedPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Different implementations based on OS
	switch runtime.GOOS {
	case "darwin":
		return setMacOSBackground(absPath)
	case "linux":
		return setLinuxBackground(absPath)
	case "windows":
		return setWindowsBackground(absPath)
	default:
		return fmt.Errorf("setting desktop background is not supported on %s", runtime.GOOS)
	}
}

// setMacOSBackground sets the background on macOS using osascript
func setMacOSBackground(imagePath string) error {
	slog.Debug("🖼️  Setting desktop background", "image", imagePath)

	// Use AppleScript to set the desktop background with a small delay
	script := fmt.Sprintf(`tell application "System Events"
		tell every desktop
			set picture to "%s"
		end tell
	end tell
	delay 0.5`, imagePath)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set macOS background: %w (output: %s)", err, string(output))
	}

	slog.Debug("✅ Desktop background set successfully")
	return nil
}

// setLinuxBackground sets the background on Linux (multiple DE support)
func setLinuxBackground(imagePath string) error {
	slog.Debug("🖼️  Setting Linux desktop background:", "imagePath", imagePath)

	// Try different desktop environments
	commands := [][]string{
		// GNOME/Ubuntu
		{"gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://" + imagePath},
		// KDE
		{"qdbus", "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript",
			fmt.Sprintf(`
			var allDesktops = desktops();
			for (i=0;i<allDesktops.length;i++) {
				d = allDesktops[i];
				d.wallpaperPlugin = "org.kde.image";
				d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
				d.writeConfig("Image", "file://%s");
			}`, imagePath)},
		// XFCE
		{"xfconf-query", "-c", "xfce4-desktop", "-p", "/backdrop/screen0/monitor0/workspace0/last-image", "-s", imagePath},
		// Fallback: feh (works with many window managers)
		{"feh", "--bg-scale", imagePath},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err == nil {
			log.Printf("✅ Linux desktop background set successfully using %s", cmd[0])
			return nil
		}
	}

	return fmt.Errorf("failed to set Linux background: no supported desktop environment found")
}

// setWindowsBackground sets the background on Windows
func setWindowsBackground(imagePath string) error {
	slog.Debug("🖼️  Setting Windows desktop background", "image", imagePath)

	// Use PowerShell to set the background with proper escaping
	// Split into multiple parts to avoid complex escaping issues
	typeDefinition := `Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public class Wallpaper {
	[DllImport("user32.dll", CharSet = CharSet.Auto)]
	public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
	public static void SetWallpaper(string path) {
		SystemParametersInfo(20, 0, path, 3);
	}
}
'@`

	// Execute the wallpaper setting command
	setWallpaperCmd := fmt.Sprintf(`[Wallpaper]::SetWallpaper('%s')`, imagePath)

	// Combine both commands
	script := typeDefinition + "; " + setWallpaperCmd

	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set Windows background: %w (output: %s)", err, string(output))
	}

	slog.Debug("✅ Windows desktop background set successfully")
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

// createTimestampedCopy creates a copy of the image with a timestamp in the filename
// This forces macOS to refresh the wallpaper even if the content is the same
func createTimestampedCopy(originalPath string) (string, error) {
	// Get the directory and file extension
	dir := filepath.Dir(originalPath)
	ext := filepath.Ext(originalPath)

	// Clean up old timestamped wallpaper files first
	cleanupOldWallpapers(dir)

	// Create a timestamp-based filename
	timestamp := time.Now().Format("20060102_150405_000")
	filename := fmt.Sprintf("iptw_wallpaper_%s%s", timestamp, ext)
	timestampedPath := filepath.Join(dir, filename)

	// Copy the file to the new timestamped location
	if err := copyFile(originalPath, timestampedPath); err != nil {
		return "", fmt.Errorf("failed to copy file to timestamped location: %w", err)
	}

	slog.Debug("🕒 Created timestamped wallpaper copy:", "filename", filename)
	return timestampedPath, nil
}

// cleanupOldWallpapers removes old timestamped wallpaper files to prevent disk space issues
func cleanupOldWallpapers(dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "iptw_wallpaper_*"))
	if err != nil {
		return
	}

	for _, file := range files {
		if err := os.Remove(file); err == nil {
			slog.Debug("🗑️  Cleaned up old wallpaper:", "file", filepath.Base(file))
		}
	}
}
