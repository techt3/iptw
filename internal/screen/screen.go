// Package screen provides screen size detection utilities
package screen

import (
	"fmt"
	"iptw/internal/logging"

	"github.com/kbinani/screenshot"
)

// ScreenInfo holds information about the screen
type ScreenInfo struct {
	Width  int
	Height int
	Count  int // Number of displays
}

// GetPrimaryScreenSize returns the size of the primary screen
func GetPrimaryScreenSize() (*ScreenInfo, error) {
	// Get the number of displays
	displayCount := screenshot.NumActiveDisplays()
	if displayCount == 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	// Get the bounds of the primary display (display 0)
	bounds := screenshot.GetDisplayBounds(0)

	info := &ScreenInfo{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Count:  displayCount,
	}
	logging.LogScreen(info.Width, info.Height, info.Count)

	return info, nil
}

// GetOptimalMapSize calculates the optimal map size for the screen
// The map will be sized to fit the screen with some padding
func GetOptimalMapSize(screenInfo *ScreenInfo) (width, height int) {
	// Use 80% of screen width to leave space for other windows
	targetWidth := int(float64(screenInfo.Width) * 0.8)

	// Calculate height as half of width (2:1 aspect ratio for world maps)
	targetHeight := targetWidth / 2

	// Ensure the height fits on screen with some padding
	maxHeight := int(float64(screenInfo.Height) * 0.8)
	if targetHeight > maxHeight {
		targetHeight = maxHeight
		targetWidth = targetHeight * 2
	}
	logging.LogMapSize(targetWidth, targetHeight, screenInfo.Width, screenInfo.Height)

	return targetWidth, targetHeight
}

// AutoDetectMapSize automatically detects screen size and returns optimal map dimensions
func AutoDetectMapSize() (width, height int, err error) {
	screenInfo, err := GetPrimaryScreenSize()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to detect screen size: %w", err)
	}

	width, height = GetOptimalMapSize(screenInfo)
	return width, height, nil
}
