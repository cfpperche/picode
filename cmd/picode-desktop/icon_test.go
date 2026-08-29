package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
)

// faviconPath is the mark the browser shows. The tray icon is a transcription
// of it (see mkicon.go), and nothing but this test connects the two files —
// so a rebrand that repaints the favicon and forgets `go generate` is caught
// here rather than by someone noticing the wrong colour in their taskbar.
const faviconPath = "../../web/public/favicon.svg"

func TestTrayIconUsesTheFaviconPalette(t *testing.T) {
	svg, err := os.ReadFile(faviconPath)
	if err != nil {
		t.Fatal(err)
	}
	want := hexFills(string(svg))
	if len(want) != 2 {
		t.Fatalf("expected two fills in the favicon, got %v", want)
	}

	pixels := largestImage(t, trayIcon)
	seen := map[string]int{}
	for _, c := range pixels {
		seen[c]++
	}
	for _, hex := range want {
		if seen[hex] == 0 {
			t.Errorf("the tray icon has no %s pixels — favicon fills are %v, icon has %v",
				hex, want, topColours(seen, 3))
		}
	}
}

// hexFills pulls the fill colours out of the SVG, lowercased, in order.
func hexFills(svg string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(svg, `fill="#`)[1:] {
		end := strings.IndexByte(part, '"')
		if end < 0 {
			continue
		}
		hex := strings.ToLower(part[:end])
		if len(hex) == 3 { // #fff
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 && !seen[hex] {
			seen[hex] = true
			out = append(out, hex)
		}
	}
	return out
}

// largestImage decodes the biggest BMP payload in the ICO to hex strings,
// skipping anything not fully opaque so antialiased edges do not count.
func largestImage(t *testing.T, ico []byte) []string {
	t.Helper()
	count := int(binary.LittleEndian.Uint16(ico[4:]))

	var best, bestSize int
	for i := 0; i < count; i++ {
		e := 6 + 16*i
		if size := int(binary.LittleEndian.Uint32(ico[e+8:])); size > bestSize {
			best, bestSize = i, size
		}
	}
	e := 6 + 16*best
	w, h := int(ico[e]), int(ico[e+1])
	off := int(binary.LittleEndian.Uint32(ico[e+12:]))
	pix := ico[off+40:] // past the BITMAPINFOHEADER

	out := make([]string, 0, w*h)
	for i := 0; i+3 < w*h*4; i += 4 {
		b, g, r, a := pix[i], pix[i+1], pix[i+2], pix[i+3]
		if a != 0xFF {
			continue
		}
		out = append(out, fmt.Sprintf("%02x%02x%02x", r, g, b))
	}
	return out
}

// topColours copies before ranking: mutating the caller's tally would make a
// second failure message describe the leftovers of the first.
func topColours(seen map[string]int, n int) []string {
	rest := make(map[string]int, len(seen))
	for hex, count := range seen {
		rest[hex] = count
	}

	var out []string
	for i := 0; i < n; i++ {
		var bestHex string
		var bestN int
		for hex, count := range rest {
			if count > bestN {
				bestHex, bestN = hex, count
			}
		}
		if bestHex == "" {
			break
		}
		out = append(out, fmt.Sprintf("%s(%d)", bestHex, bestN))
		delete(rest, bestHex)
	}
	return out
}
