//go:build ignore

// mkicon writes cmd/picode-desktop/icon.ico from the same mark the browser
// shows: web/public/favicon.svg, the blocky Pi in white on #09090b.
//
// The SVG's geometry is reproduced here rather than parsed, because it is nine
// axis-aligned rectangles on an 800-unit grid and a rasteriser would be a
// dependency for arithmetic. If the favicon changes, change these numbers with
// it — the shapes below are a transcription, not an interpretation.
//
// Everything is drawn at 4x and boxed down, so the rounded corners and the
// step edges stay clean at 16px instead of crawling.
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

// The favicon palette, verbatim.
var (
	ground = color.NRGBA{0x09, 0x09, 0x0B, 0xFF}
	mark   = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
)

// grid is the SVG viewBox. The mark is laid out on sevenths of it: every
// coordinate in the path is a multiple of 800/6.817.
const grid = 800.0

// corner is the rx of the background rect.
const corner = 120.0

// rect is one shape in grid units.
type rect struct{ x0, y0, x1, y1 float64 }

// The white shapes, transcribed from favicon.svg. The P is a staircase, so it
// is expressed as the three horizontal bands the path walks down; the counter
// is punched out afterwards, matching the SVG's fill-rule="evenodd".
var (
	bands = []rect{
		{165.29, 165.29, 517.36, 400.00}, // the bowl, full width
		{165.29, 400.00, 400.00, 517.36}, // first step in
		{165.29, 517.36, 282.65, 634.72}, // the stem
		{517.36, 400.00, 634.72, 634.72}, // the separate i
	}
	counter = rect{282.65, 282.65, 400.00, 400.00}
)

func render(size int) *image.NRGBA {
	const ss = 4 // supersample factor
	big := image.NewNRGBA(image.Rect(0, 0, size*ss, size*ss))
	s := float64(size * ss)
	unit := s / grid

	r := corner * unit
	for y := 0; y < size*ss; y++ {
		for x := 0; x < size*ss; x++ {
			if insideRounded(float64(x)+0.5, float64(y)+0.5, s, r) {
				big.SetNRGBA(x, y, ground)
			}
		}
	}
	for _, b := range bands {
		fill(big, b, unit, mark)
	}
	fill(big, counter, unit, ground)

	return downsample(big, size)
}

// insideRounded is a rounded square: inside the inner cross always, and inside
// a corner only within its radius.
func insideRounded(x, y, s, r float64) bool {
	if x < 0 || y < 0 || x > s || y > s {
		return false
	}
	cx := clamp(x, r, s-r)
	cy := clamp(y, r, s-r)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
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

func fill(img *image.NRGBA, r rect, unit float64, c color.NRGBA) {
	draw.Draw(img, image.Rect(
		int(r.x0*unit+0.5), int(r.y0*unit+0.5),
		int(r.x1*unit+0.5), int(r.y1*unit+0.5),
	), &image.Uniform{c}, image.Point{}, draw.Src)
}

// downsample box-averages the supersampled image, alpha included, so the
// rounded corners fade instead of stepping.
func downsample(src *image.NRGBA, size int) *image.NRGBA {
	ss := src.Bounds().Dx() / size
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a int
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					c := src.NRGBAAt(x*ss+dx, y*ss+dy)
					// Weight colour by alpha so transparent pixels outside the
					// rounded corner do not drag the edge towards black.
					r += int(c.R) * int(c.A) / 255
					g += int(c.G) * int(c.A) / 255
					b += int(c.B) * int(c.A) / 255
					a += int(c.A)
				}
			}
			n := ss * ss
			if a == 0 {
				continue
			}
			// Un-premultiply back to straight alpha.
			out.SetNRGBA(x, y, color.NRGBA{
				R: uint8(r * 255 / a),
				G: uint8(g * 255 / a),
				B: uint8(b * 255 / a),
				A: uint8(a / n),
			})
		}
	}
	return out
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
	if err := binary.Write(&buf, binary.LittleEndian, hdr); err != nil {
		log.Fatal(err)
	}

	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(x, y)
			buf.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
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
	if err := binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))}); err != nil {
		log.Fatal(err)
	}

	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		out.Write([]byte{byte(s), byte(s), 0, 0})
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))
		_ = binary.Write(&out, binary.LittleEndian, uint16(32))
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(payloads[i])))
		_ = binary.Write(&out, binary.LittleEndian, uint32(offset))
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
