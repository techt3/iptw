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

// setTrayIcon sets the system tray icon.
func setTrayIcon() {
	systray.SetIcon(generateTrayIcon())
}
