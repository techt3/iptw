//go:build !windows

package gui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"fyne.io/systray"
)

// generateTrayIcon creates a 32×32 globe icon as PNG bytes.
// fyne.io/systray on non-Windows platforms expects PNG data.
func generateTrayIcon() []byte {
	const size = 32
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1

	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy > r*r {
				// Outside circle — leave transparent (zero value).
				continue
			}
			lat := dy / r
			lon := math.Atan2(dy, dx) / math.Pi
			if math.Abs(math.Sin(lat*math.Pi*3+lon*math.Pi*2)) > 0.55 {
				// green land
				img.SetNRGBA(x, y, color.NRGBA{R: 0x34, G: 0xa8, B: 0x53, A: 0xff})
			} else {
				// blue ocean
				img.SetNRGBA(x, y, color.NRGBA{R: 0x1a, G: 0x73, B: 0xe8, A: 0xff})
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// generateTemplateTrayIcon creates a 32×32 white globe icon suitable for use as
// a macOS menu-bar template image.  Template images must be monochrome
// (white + transparency); macOS then colours them automatically to match the
// current menu-bar appearance (light/dark mode, tinting, etc.).
//
// On Linux the template bytes are never used — systray_unix falls back to the
// regular coloured icon — so this function only needs to produce valid PNG data.
func generateTemplateTrayIcon() []byte {
	const size = 32
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1

	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist2 := dx*dx + dy*dy
			if dist2 > r*r {
				continue
			}

			// Draw a thin outer ring to outline the globe.
			outerRing := r - 1
			if dist2 >= outerRing*outerRing {
				img.SetNRGBA(x, y, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
				continue
			}

			// Interior: use the same longitude-line pattern as the coloured icon
			// but render everything as solid white.
			lat := dy / r
			lon := math.Atan2(dy, dx) / math.Pi
			if math.Abs(math.Sin(lat*math.Pi*3+lon*math.Pi*2)) > 0.55 {
				// "land" — full white
				img.SetNRGBA(x, y, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
			} else {
				// "ocean" — semi-transparent white so the two regions are
				// distinguishable while still being a valid template image.
				img.SetNRGBA(x, y, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xaa})
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// setTrayIcon sets the system tray icon.
// On macOS (Sonoma / Sequoia and later) only template images are rendered
// correctly in the menu bar; SetTemplateIcon passes isTemplate=true to Cocoa so
// the OS adapts the icon to light/dark mode automatically.
// On Linux, fyne.io/systray forwards the regularIconBytes to SetIcon unchanged.
func setTrayIcon() {
	systray.SetTemplateIcon(generateTemplateTrayIcon(), generateTrayIcon())
}
