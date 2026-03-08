//go:build windows

package gui

import (
	"bytes"
	"encoding/binary"
	"math"

	"fyne.io/systray"
)

// setTrayIcon sets the system tray icon.
func setTrayIcon() {
	systray.SetIcon(generateTrayIcon())
}

// generateTrayIcon creates a 32×32 globe icon in ICO format.
//
// fyne.io/systray on Windows passes the bytes to CreateIconFromResourceEx,
// which expects an ICO-format DIB (BITMAPINFOHEADER + BGRA XOR-mask +
// AND-mask), NOT a PNG file. Passing PNG results in an invisible icon.
func generateTrayIcon() []byte {
	const size = 32
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 1

	// XOR mask: 32×32 BGRA pixels, stored bottom-up (BMP convention).
	pixels := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			// BMP rows are bottom-up.
			bmpRow := size - 1 - y
			idx := (bmpRow*size + x) * 4
			if dx*dx+dy*dy > r*r {
				// Outside circle — fully transparent (alpha = 0).
				continue
			}
			lat := dy / r
			lon := math.Atan2(dy, dx) / math.Pi
			if math.Abs(math.Sin(lat*math.Pi*3+lon*math.Pi*2)) > 0.55 {
				// green land — BGRA
				pixels[idx+0] = 0x53
				pixels[idx+1] = 0xa8
				pixels[idx+2] = 0x34
				pixels[idx+3] = 0xff
			} else {
				// blue ocean — BGRA
				pixels[idx+0] = 0xe8
				pixels[idx+1] = 0x73
				pixels[idx+2] = 0x1a
				pixels[idx+3] = 0xff
			}
		}
	}

	// AND mask: all zeros → let the 32bpp alpha channel control transparency.
	andMask := make([]byte, size*(size/8)) // 32 * 4 bytes per row = 128 bytes

	// BITMAPINFOHEADER (40 bytes).
	// biHeight = size*2 because the ICO DIB combines the XOR and AND masks.
	var bih [40]byte
	binary.LittleEndian.PutUint32(bih[0:], 40)             // biSize
	binary.LittleEndian.PutUint32(bih[4:], uint32(size))   // biWidth
	binary.LittleEndian.PutUint32(bih[8:], uint32(size*2)) // biHeight (XOR+AND)
	binary.LittleEndian.PutUint16(bih[12:], 1)             // biPlanes
	binary.LittleEndian.PutUint16(bih[14:], 32)            // biBitCount
	// remaining fields zero: BI_RGB, no compression

	imageBytesLen := uint32(len(bih) + len(pixels) + len(andMask))

	// ICONDIRENTRY (16 bytes).
	var entry [16]byte
	entry[0] = size // bWidth
	entry[1] = size // bHeight
	// bColorCount, bReserved = 0
	binary.LittleEndian.PutUint16(entry[4:], 1)             // wPlanes
	binary.LittleEndian.PutUint16(entry[6:], 32)            // wBitCount
	binary.LittleEndian.PutUint32(entry[8:], imageBytesLen) // dwBytesInRes
	binary.LittleEndian.PutUint32(entry[12:], 22)           // dwImageOffset = 6+16

	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0}) // ICO header: reserved=0, type=1, count=1
	ico.Write(entry[:])
	ico.Write(bih[:])
	ico.Write(pixels)
	ico.Write(andMask)
	return ico.Bytes()
}
