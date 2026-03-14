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
	"strings"
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

	uri := "file://" + imagePath

	// GNOME / Ubuntu: gsettings sets both light and dark wallpaper URIs.
	// GNOME 42+ requires picture-uri-dark for dark-mode desktops; the key is
	// silently ignored on older releases where it does not exist.
	if err := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run(); err == nil {
		// picture-uri succeeded — we're on GNOME.  Also update the dark variant.
		_ = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri).Run()
		log.Printf("✅ Linux desktop background set successfully using gsettings")
		return nil
	}

	// KDE Plasma
	kdeScript := fmt.Sprintf(`
		var allDesktops = desktops();
		for (i=0;i<allDesktops.length;i++) {
			d = allDesktops[i];
			d.wallpaperPlugin = "org.kde.image";
			d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
			d.writeConfig("Image", "file://%s");
		}`, imagePath)
	if err := exec.Command("qdbus", "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", kdeScript).Run(); err == nil {
		log.Printf("✅ Linux desktop background set successfully using qdbus")
		return nil
	}

	// XFCE
	if err := exec.Command("xfconf-query", "-c", "xfce4-desktop", "-p", "/backdrop/screen0/monitor0/workspace0/last-image", "-s", imagePath).Run(); err == nil {
		log.Printf("✅ Linux desktop background set successfully using xfconf-query")
		return nil
	}

	// Fallback: feh (works with many lightweight window managers)
	if err := exec.Command("feh", "--bg-scale", imagePath).Run(); err == nil {
		log.Printf("✅ Linux desktop background set successfully using feh")
		return nil
	}

	return fmt.Errorf("failed to set Linux background: no supported desktop environment found")
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

// BackupCurrentWallpaper backs up the current desktop wallpaper
// Returns the path where the backup was saved
func BackupCurrentWallpaper(backupDir string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	var currentWallpaperPath string
	var err error

	switch runtime.GOOS {
	case "darwin":
		currentWallpaperPath, err = getMacOSCurrentWallpaper()
	case "linux":
		currentWallpaperPath, err = getLinuxCurrentWallpaper()
	case "windows":
		currentWallpaperPath, err = getWindowsCurrentWallpaper()
	default:
		return "", fmt.Errorf("wallpaper backup is not supported on %s", runtime.GOOS)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get current wallpaper path: %w", err)
	}

	if currentWallpaperPath == "" {
		return "", fmt.Errorf("could not determine current wallpaper path")
	}

	// Don't backup if the current wallpaper is already an IPTW wallpaper
	// This prevents overwriting the true "original" with our generated output
	baseName := filepath.Base(currentWallpaperPath)
	if strings.Contains(baseName, "iptw_wallpaper_") || strings.Contains(baseName, "iptw.png") {
		return "", fmt.Errorf("current wallpaper is already an IPTW wallpaper, skipping backup to preserve original")
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("original_wallpaper_%s%s", timestamp, filepath.Ext(currentWallpaperPath))
	backupPath := filepath.Join(backupDir, backupFilename)

	// Copy the current wallpaper to backup location
	if err := copyFile(currentWallpaperPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to backup wallpaper: %w", err)
	}

	slog.Info("💾 Original wallpaper backed up", "from", currentWallpaperPath, "to", backupPath)
	return backupPath, nil
}

// getMacOSCurrentWallpaper gets the current wallpaper path on macOS
func getMacOSCurrentWallpaper() (string, error) {
	script := `tell application "System Events"
		tell current desktop
			get picture
		end tell
	end tell`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get macOS wallpaper: %w", err)
	}

	wallpaperPath := strings.TrimSpace(string(output))
	// Remove alias format if present
	wallpaperPath = strings.TrimPrefix(wallpaperPath, "alias ")
	// Remove quotes if present
	wallpaperPath = strings.Trim(wallpaperPath, "\"")

	return wallpaperPath, nil
}

// getLinuxCurrentWallpaper gets the current wallpaper path on Linux
func getLinuxCurrentWallpaper() (string, error) {
	// Try different desktop environments

	// GNOME/Ubuntu
	cmd := exec.Command("gsettings", "get", "org.gnome.desktop.background", "picture-uri")
	if output, err := cmd.Output(); err == nil {
		wallpaperURI := strings.TrimSpace(string(output))
		wallpaperURI = strings.Trim(wallpaperURI, "'\"")
		if strings.HasPrefix(wallpaperURI, "file://") {
			return strings.TrimPrefix(wallpaperURI, "file://"), nil
		}
		return wallpaperURI, nil
	}

	// XFCE
	cmd = exec.Command("xfconf-query", "-c", "xfce4-desktop", "-p", "/backdrop/screen0/monitor0/workspace0/last-image")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// KDE - this is more complex, try to get from config
	homeDir, _ := os.UserHomeDir()
	kdePlasmaConfig := filepath.Join(homeDir, ".config", "plasma-org.kde.plasma.desktop-appletsrc")
	if _, err := os.Stat(kdePlasmaConfig); err == nil {
		// This is a simplified approach - KDE config parsing is complex
		slog.Warn("KDE wallpaper detection is limited - backup may not work perfectly")
		return "", fmt.Errorf("KDE wallpaper backup not fully supported")
	}

	return "", fmt.Errorf("could not detect current wallpaper on this Linux desktop environment")
}
