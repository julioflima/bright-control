//go:build linux || windows

package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"runtime"

	"fyne.io/systray"
)

const iconSize = 64 // rendered big, the tray scales it down

// coverage-based anti-aliasing: sample each pixel 4x4
func render(inside func(x, y float64) color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, iconSize, iconSize))
	const n = 4
	for py := 0; py < iconSize; py++ {
		for px := 0; px < iconSize; px++ {
			var r, g, b, a float64
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					c := inside(float64(px)+(float64(sx)+0.5)/n, float64(py)+(float64(sy)+0.5)/n)
					fa := float64(c.A) / 255
					r += float64(c.R) * fa
					g += float64(c.G) * fa
					b += float64(c.B) * fa
					a += fa
				}
			}
			if a > 0 {
				img.SetNRGBA(px, py, color.NRGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(a / (n * n) * 255)})
			}
		}
	}
	return img
}

func encode(img image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func withAlpha(c color.NRGBA, a float64) color.NRGBA { c.A = uint8(float64(c.A) * a); return c }

// Crescent moon: filled when active, outlined when disabled/paused.
func moonIcon(active bool) []byte {
	fg := trayForeground()
	cx, cy, r := 30.0, 34.0, 24.0
	ox, oy, or := 44.0, 22.0, 20.0
	return encode(render(func(x, y float64) color.NRGBA {
		d1 := math.Hypot(x-cx, y-cy)
		d2 := math.Hypot(x-ox, y-oy)
		if active {
			if d1 <= r && d2 > or {
				return fg
			}
			return color.NRGBA{}
		}
		// outline: ring of the crescent shape, 5 px wide
		const w = 5.0
		inShape := d1 <= r && d2 > or
		inInner := d1 <= r-w && d2 > or+w
		if inShape && !inInner {
			return fg
		}
		return color.NRGBA{}
	}))
}

// Battery drawn like the macOS one: translucent outline + cap, red charge level.
func batteryIcon(percent int) []byte {
	fg := trayForeground()
	red := color.NRGBA{255, 69, 58, 255}
	// body 4..54 x 18..46, cap 56..60 x 26..38
	bx0, by0, bx1, by1, rad := 3.0, 17.0, 55.0, 47.0, 8.0
	frac := math.Max(0.1, math.Min(1, float64(percent)/100))
	fx0, fy0, fy1 := bx0+6, by0+6, by1-6
	fx1 := fx0 + (bx1-6-fx0)*frac
	return encode(render(func(x, y float64) color.NRGBA {
		if x >= fx0 && x <= fx1 && y >= fy0 && y <= fy1 && inRoundRect(x, y, fx0, fy0, fx1, fy1, 3) {
			return red
		}
		outer := inRoundRect(x, y, bx0, by0, bx1, by1, rad)
		inner := inRoundRect(x, y, bx0+3, by0+3, bx1-3, by1-3, rad-3)
		if outer && !inner {
			return withAlpha(fg, 0.6)
		}
		if x >= 57 && x <= 61 && y >= 26 && y <= 38 && inRoundRect(x, y, 55, 26, 61, 38, 3) {
			return withAlpha(fg, 0.6)
		}
		return color.NRGBA{}
	}))
}

func inRoundRect(x, y, x0, y0, x1, y1, r float64) bool {
	if x < x0 || x > x1 || y < y0 || y > y1 {
		return false
	}
	cx := math.Max(x0+r, math.Min(x, x1-r))
	cy := math.Max(y0+r, math.Min(y, y1-r))
	return math.Hypot(x-cx, y-cy) <= r
}

// Windows wants .ico bytes; an ICO may embed a PNG directly (Vista+).
func setTrayIcon(pngBytes []byte) {
	if runtime.GOOS != "windows" {
		systray.SetIcon(pngBytes)
		return
	}
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, []uint16{0, 1, 1}) // reserved, type=icon, count
	buf.Write([]byte{0, 0, 0, 0})                                  // 256x256 (0 = 256), no palette, reserved
	_ = binary.Write(&buf, binary.LittleEndian, []uint16{1, 32})   // planes, bpp
	_ = binary.Write(&buf, binary.LittleEndian, []uint32{uint32(len(pngBytes)), 22})
	buf.Write(pngBytes)
	systray.SetIcon(buf.Bytes())
}
