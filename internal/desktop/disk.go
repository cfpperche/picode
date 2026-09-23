package desktop

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DiskFacts is the half of a WSL disk only Windows can see: the VHDX file's
// real size, whether it is sparse (only then does WSL give freed blocks back
// on its own), and how much room is left on the volume holding it.
//
// The Linux half measures what the distro uses; the two numbers together are
// the one a person actually needs, because the difference is space they
// cannot see anywhere in Windows (ADR-0020 keeps the two halves apart, and
// docs/plans/wsl-control.md is what joins them).
type DiskFacts struct {
	Distro string `json:"distro"`
	// VHDXPath comes from WSL's own bookkeeping, never from guessing the
	// package folder: `wsl --manage --move` and the Store-installed layouts
	// put the file in different places.
	VHDXPath string `json:"vhdxPath"`
	// VHDXBytes is the file's own length — the high-water mark of everything
	// WSL has ever written into it. AllocatedBytes is what the volume actually
	// holds; the two differ only while the file is sparse, and that difference
	// is the space Windows cannot give back on its own.
	VHDXBytes      int64 `json:"vhdxBytes"`
	AllocatedBytes int64 `json:"allocatedBytes"`
	Sparse         bool  `json:"sparse"`
	// FreeBytes and VolumeBytes describe the volume the VHDX lives on — for a
	// full C: this is the number that matters, not the distro's own df.
	FreeBytes   int64 `json:"freeBytes"`
	VolumeBytes int64 `json:"volumeBytes"`
	// WSL is the product version, and CanSparse says whether this build can
	// convert the file (`wsl --manage <distro> --set-sparse true`). WSL 2.0
	// and later can; older builds have to be compacted by hand.
	WSL       string `json:"wsl"`
	CanSparse bool   `json:"canSparse"`
	// CanMove / CanExportVHD: this WSL knows `--manage --move` and
	// `--export --format vhd` — checked before anything is stopped.
	CanMove      bool      `json:"canMove"`
	CanExportVHD bool      `json:"canExportVhd"`
	At           time.Time `json:"at"`
}

// DistroFolder is one distro as WSL itself records it.
type DistroFolder struct {
	Name     string
	BasePath string
	VHDX     string
}

// Held is what the VHDX file holds that the distro no longer uses — the space
// a compaction returns to Windows. It is an estimate on purpose: the VHDX
// keeps its own metadata, so the honest label is "about this much", and the
// job that compacts measures the real difference.
func Held(f DiskFacts, distroUsedBytes int64) int64 {
	if distroUsedBytes <= 0 || f.AllocatedBytes <= distroUsedBytes {
		return 0
	}
	return f.AllocatedBytes - distroUsedBytes
}

// DistroDisk reads everything Windows knows about one distro's disk file.
func DistroDisk(r Runner, distro string) (DiskFacts, error) {
	facts := DiskFacts{Distro: distro, At: time.Now().UTC()}

	folders, err := listDistroFolders(r)
	if err != nil {
		return facts, err
	}
	folder, err := pickFolder(folders, distro)
	if err != nil {
		return facts, err
	}
	facts.VHDXPath = winJoin(folder.BasePath, folder.VHDX)

	vol, allocated, sparse, err := footprintOf(r, facts.VHDXPath)
	if err != nil {
		return facts, err
	}
	facts.VHDXBytes, facts.VolumeBytes, facts.FreeBytes = vol.Length, vol.Total, vol.Free
	facts.AllocatedBytes, facts.Sparse = allocated, sparse

	// wsl.exe exits non-zero on `--help` (255, on every machine we have seen)
	// while printing the whole text to stdout, so the exit status is not the
	// answer here — output is. Same guard on `--version`, which has no reason
	// to stay well-behaved across WSL updates.
	if out, err := r.Output(WSLExe, "--version"); err == nil || len(out) > 0 {
		facts.WSL = ParseWSLVersion(out)
	}
	if out, err := r.Output(WSLExe, "--help"); err == nil || len(out) > 0 {
		help := DecodeWindows(out)
		facts.CanSparse = strings.Contains(help, "--set-sparse")
		facts.CanMove = strings.Contains(help, "--move")
		facts.CanExportVHD = strings.Contains(help, "--format")
	}
	return facts, nil
}

// listDistroFolders asks the registry WSL writes: the subkeys under Lxss are
// the installed distros, each with the name a person sees and the folder that
// holds its ext4.vhdx.
func listDistroFolders(r Runner) ([]DistroFolder, error) {
	out, err := r.Output(RegExe, "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Lxss`, "/s")
	if err != nil {
		return nil, fmt.Errorf("read WSL's registry: %w", err)
	}
	folders := ParseLxss(out)
	if len(folders) == 0 {
		return nil, fmt.Errorf("WSL lists no distributions on this machine")
	}
	return folders, nil
}

// pickFolder matches the distro by the name WSL reports, so `--distro Ubuntu`
// and the registry's `DistributionName` are the same string.
func pickFolder(folders []DistroFolder, distro string) (DistroFolder, error) {
	for _, f := range folders {
		if strings.EqualFold(f.Name, distro) {
			return f, nil
		}
	}
	if len(folders) == 1 {
		return folders[0], nil
	}
	return DistroFolder{}, fmt.Errorf("no WSL distribution named %q", distro)
}

// winJoin joins a Windows path the way Windows would, whatever platform this
// binary was compiled on. The desktop package is written and tested from
// Linux on purpose (its logic lives in pure functions), where filepath.Join
// answers with a forward slash and no Windows tool accepts the result.
func winJoin(dir, name string) string {
	return strings.TrimRight(dir, `\/`) + `\` + name
}

// ParseLxss reads `reg query …\Lxss /s`: one block per distro, each starting
// at its HKEY_ line and carrying indented `NAME  REG_SZ  value` lines in no
// particular order.
func ParseLxss(out []byte) []DistroFolder {
	var folders []DistroFolder
	var cur *DistroFolder

	flush := func() {
		if cur != nil && cur.Name != "" && cur.BasePath != "" {
			if cur.VHDX == "" {
				// The value is absent on older WSL builds; ext4.vhdx is what
				// every one of them has always used.
				cur.VHDX = "ext4.vhdx"
			}
			folders = append(folders, *cur)
		}
		cur = nil
	}

	for _, line := range strings.Split(DecodeWindows(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "HKEY_") {
			flush()
			cur = &DistroFolder{}
			continue
		}
		if cur == nil {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 || !strings.HasPrefix(fields[1], "REG_") {
			continue
		}
		// Everything after the type is the value: it may contain spaces, and a
		// REG_EXPAND_SZ path keeps its %VAR% here.
		value := strings.TrimSpace(line[strings.Index(line, fields[1])+len(fields[1]):])
		switch fields[0] {
		case "DistributionName":
			cur.Name = value
		case "BasePath":
			cur.BasePath = value
		case "VhdFileName":
			cur.VHDX = value
		}
	}
	flush()
	return folders
}

// Volume is the three numbers PowerShell answers with.
type Volume struct {
	Length int64 `json:"len"`
	Total  int64 `json:"total"`
	Free   int64 `json:"free"`
}

// volumeOf reads the file and the drive it sits on in one call, as JSON so
// nothing about a locale decides what "." means in a size.
func volumeOf(r Runner, path string) (Volume, error) {
	if strings.Contains(path, "'") {
		// Single quotes are the only way out of a PowerShell literal we do not
		// want to learn to escape; a Windows path never contains one.
		return Volume{}, fmt.Errorf("unexpected quote in %s", path)
	}
	script := `$p='` + path + `'; $f=Get-Item -LiteralPath $p -Force; ` +
		`$d=New-Object System.IO.DriveInfo([System.IO.Path]::GetPathRoot($p)); ` +
		`[pscustomobject]@{len=[int64]$f.Length; total=[int64]$d.TotalSize; free=[int64]$d.AvailableFreeSpace} | ConvertTo-Json -Compress`
	out, err := r.Output(PowerShellExe, "-NoProfile", "-NonInteractive", "-Command", script)
	if err != nil {
		return Volume{}, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseVolumeJSON(out)
}

// ParseVolumeJSON tolerates anything the host printed before the object —
// a PowerShell profile banner, a warning — and starts at the first brace.
func ParseVolumeJSON(out []byte) (Volume, error) {
	text := DecodeWindows(out)
	start := strings.Index(text, "{")
	if start < 0 {
		return Volume{}, fmt.Errorf("no JSON in %q", firstLine(text))
	}
	var v Volume
	if err := json.Unmarshal([]byte(text[start:]), &v); err != nil {
		return Volume{}, fmt.Errorf("volume JSON: %w", err)
	}
	if v.Length <= 0 {
		return Volume{}, fmt.Errorf("volume JSON has no length")
	}
	return v, nil
}

// sparseRange asks fsutil for the file's allocated extents.
//
// The parse reads the last two hex numbers on a line and nothing else: fsutil
// speaks the machine's display language, so "Allocated range", "Offset" and
// "Length" are all translated, while the numbers are not. No extent line at
// all means the file is not sparse — which is also how the command answers,
// in whichever sentence that language uses.
func sparseRange(r Runner, path string) (allocated int64, sparse bool, err error) {
	out, err := r.Output(FsutilExe, "sparse", "queryrange", path)
	if err != nil && len(out) == 0 {
		return 0, false, fmt.Errorf("fsutil queryrange: %w", err)
	}
	allocated, sparse = ParseSparseRanges(out)
	return allocated, sparse, nil
}

// ParseSparseRanges sums the extents and reports whether there were any.
func ParseSparseRanges(out []byte) (int64, bool) {
	var total int64
	var found bool
	for _, line := range strings.Split(DecodeWindows(out), "\n") {
		hexes := hexTokens(line)
		if len(hexes) < 2 {
			continue
		}
		length, ok := parseHex(hexes[len(hexes)-1])
		if !ok {
			continue
		}
		total += length
		found = true
	}
	return total, found
}

// hexTokens returns every 0x-prefixed number on a line, in order.
func hexTokens(line string) []string {
	var out []string
	for i := 0; i < len(line); i++ {
		if line[i] != '0' || i+2 >= len(line) || (line[i+1] != 'x' && line[i+1] != 'X') {
			continue
		}
		j := i + 2
		for j < len(line) && isHexDigit(line[j]) {
			j++
		}
		if j > i+2 {
			out = append(out, line[i:j])
			i = j - 1
		}
	}
	return out
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func parseHex(s string) (int64, bool) {
	n, err := strconv.ParseInt(s[2:], 16, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// ParseWSLVersion reads the first version-shaped token of `wsl --version`,
// skipping the label so a translated build reports the same number.
func ParseWSLVersion(out []byte) string {
	for _, line := range strings.Split(DecodeWindows(out), "\n") {
		for _, tok := range strings.Fields(strings.TrimRight(line, "\r")) {
			if isVersion(tok) {
				return tok
			}
		}
	}
	return ""
}

// isVersion accepts "2.7.0.0" and rejects a word with a dot in it.
func isVersion(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for i := 0; i < len(p); i++ {
			if p[i] < '0' || p[i] > '9' {
				return false
			}
		}
	}
	return true
}
