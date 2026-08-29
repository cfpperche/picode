//go:build ignore

// mkicon writes cmd/picode-desktop/icon.ico: a white pi on the PiCode indigo,
// at the sizes Windows asks a tray for. BMP payloads (not PNG) so every
// Windows version loads it without a codec path.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
)

var (
	accent = color.NRGBA{0x4B, 0x45, 0xC6, 0xFF}
	ink    = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
)

func render(s int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, s, s))

	// Rounded square ground. The radius is generous so the glyph reads as a
	// badge rather than a screenshot at 16px.
	r := float64(s) * 0.22
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			if inRounded(float64(x)+0.5, float64(y)+0.5, float64(s), r) {
				img.SetNRGBA(x, y, accent)
			}
		}
	}

	// Pi: one top bar and two legs. Thicknesses are rounded up so the glyph
	// never disappears at 16px.
	f := func(v float64) int { return int(v*float64(s) + 0.5) }
	t := f(0.11)
	if t < 2 {
		t = 2
	}
	top, bottom := f(0.28), f(0.76)
	left, right := f(0.17), f(0.83)
	// The legs sit just inside the bar. At 16px a wider inset makes them meet
	// in the middle and the glyph reads as a T, so the gap is what matters,
	// not the elegance of the inset.
	inset := f(0.07)

	fill(img, left, top, right, top+t)                 // bar
	fill(img, left+inset, top, left+inset+t, bottom)   // left leg
	fill(img, right-inset-t, top, right-inset, bottom) // right leg
	return img
}

func inRounded(x, y, s, r float64) bool {
	cx, cy := clamp(x, r, s-r), clamp(y, r, s-r)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r || (x >= r && x <= s-r) || (y >= r && y <= s-r)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func fill(img *image.NRGBA, x0, y0, x1, y1 int) {
	draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{ink}, image.Point{}, draw.Src)
}

// bmpPayload is an ICO image: a BITMAPINFOHEADER whose height is doubled (the
// XOR pixels plus an AND mask), 32-bit BGRA bottom-up, then a 1bpp mask.
func bmpPayload(img *image.NRGBA) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var buf bytes.Buffer

	hdr := struct {
		Size                   uint32
		Width, Height          int32
		Planes, BitCount       uint16
		Compression, SizeImage uint32
		XPPM, YPPM             int32
		ClrUsed, ClrImportant  uint32
	}{Size: 40, Width: int32(w), Height: int32(h * 2), Planes: 1, BitCount: 32}
	binary.Write(&buf, binary.LittleEndian, hdr)

	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(x, y)
			buf.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	// AND mask: fully opaque, rows padded to 4 bytes.
	rowBytes := ((w + 31) / 32) * 4
	buf.Write(make([]byte, rowBytes*h))
	return buf.Bytes()
}

func main() {
	sizes := []int{16, 20, 24, 32, 48, 64}
	payloads := make([][]byte, len(sizes))
	for i, s := range sizes {
		payloads[i] = bmpPayload(render(s))
	}

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})

	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		dim := byte(s)
		if s >= 256 {
			dim = 0
		}
		out.Write([]byte{dim, dim, 0, 0})
		binary.Write(&out, binary.LittleEndian, uint16(1))
		binary.Write(&out, binary.LittleEndian, uint16(32))
		binary.Write(&out, binary.LittleEndian, uint32(len(payloads[i])))
		binary.Write(&out, binary.LittleEndian, uint32(offset))
		offset += len(payloads[i])
	}
	for _, p := range payloads {
		out.Write(p)
	}

	if err := os.WriteFile(os.Args[1], out.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (%d bytes, %d sizes)", os.Args[1], out.Len(), len(sizes))
}
