// Package desktop is the Windows half of PiCode Desktop (ADR-0020): it owns
// the WSL boundary and the Windows integration points, and drives `picode
// provision` inside the distro for everything else. Nothing here knows about
// systemd, and nothing in internal/provision knows about Windows.
//
// The logic that can be wrong lives in pure functions so it is testable from
// Linux, where this code is written and where CI runs; only the syscalls are
// behind Windows build tags.
package desktop

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Distro is one entry of `wsl.exe --list --verbose`.
type Distro struct {
	Name    string
	State   string
	Version int
	Default bool
}

// Running reports whether the distro is up. WSL reports "Running" only while
// something holds it open, which is exactly what the keepalive child is for.
func (d Distro) Running() bool { return strings.EqualFold(d.State, "Running") }

// DecodeWindows turns the output of a Windows console program into a Go
// string. wsl.exe answers in UTF-16LE **without a BOM**, so decoding has to be
// decided by looking at the bytes: reading that output as UTF-8 is the classic
// way to end up parsing "U\x00b\x00u\x00n\x00t\x00u\x00".
func DecodeWindows(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		return decodeUTF16LE(b[2:])
	}
	if looksUTF16LE(b) {
		return decodeUTF16LE(b)
	}
	return string(b)
}

// looksUTF16LE samples the high byte of each code unit. Console output is
// effectively ASCII, so in UTF-16LE almost every odd byte is zero — while
// well-formed UTF-8 never contains an interior NUL at all.
func looksUTF16LE(b []byte) bool {
	if len(b) < 2 || len(b)%2 != 0 {
		return false
	}
	sample := len(b)
	if sample > 128 {
		sample = 128
	}
	var zeros, total int
	for i := 1; i < sample; i += 2 {
		total++
		if b[i] == 0 {
			zeros++
		}
	}
	return total > 0 && zeros*10 >= total*7
}

func decodeUTF16LE(b []byte) string {
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		units = append(units, uint16(b[i])|uint16(b[i+1])<<8)
	}
	return string(utf16.Decode(units))
}

// ParseDistros reads `wsl.exe --list --verbose`. Fields are read from the
// right — the version is the last column and the state the one before it — so
// a distro whose name contains a space still parses, which splitting on
// whitespace from the left would not.
func ParseDistros(raw []byte) ([]Distro, error) {
	text := strings.ReplaceAll(DecodeWindows(raw), "\r\n", "\n")
	var out []Distro

	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		isDefault := strings.HasPrefix(strings.TrimSpace(line), "*")
		rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))

		fields := strings.Fields(rest)
		if len(fields) < 3 {
			continue
		}
		// The header row has the same shape as a distro row; it is only
		// recognisable by its content.
		if strings.EqualFold(fields[0], "NAME") {
			continue
		}
		version, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			continue
		}
		name := strings.TrimSpace(strings.Join(fields[:len(fields)-2], " "))
		if name == "" {
			continue
		}
		out = append(out, Distro{
			Name:    name,
			State:   fields[len(fields)-2],
			Version: version,
			Default: isDefault,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no WSL distributions found — install one with `wsl --install`")
	}
	return out, nil
}

// Pick chooses the distro to provision. A name the user configured wins; then
// the one WSL marks as default; then, when there is exactly one, that one.
// Anything else is ambiguous and the caller has to ask.
func Pick(distros []Distro, preferred string) (Distro, error) {
	if preferred != "" {
		for _, d := range distros {
			if strings.EqualFold(d.Name, preferred) {
				if d.Version != 2 {
					return d, fmt.Errorf("distro %q is WSL %d — PiCode needs WSL 2", d.Name, d.Version)
				}
				return d, nil
			}
		}
		return Distro{}, fmt.Errorf("distro %q is not installed", preferred)
	}

	var usable []Distro
	for _, d := range distros {
		// docker-desktop's own distros are infrastructure, never a place to
		// install PiCode.
		if strings.HasPrefix(strings.ToLower(d.Name), "docker-desktop") {
			continue
		}
		if d.Version == 2 {
			usable = append(usable, d)
		}
	}
	switch {
	case len(usable) == 0:
		return Distro{}, fmt.Errorf("no WSL 2 distribution available")
	case len(usable) == 1:
		return usable[0], nil
	}
	for _, d := range usable {
		if d.Default {
			return d, nil
		}
	}
	names := make([]string, len(usable))
	for i, d := range usable {
		names[i] = d.Name
	}
	return Distro{}, fmt.Errorf("several WSL 2 distributions (%s) — choose one with --distro",
		strings.Join(names, ", "))
}
