package desktop

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// The host view: what the WSL VM costs the machine in memory right now, and
// which WSL it is. Both sides are read, never guessed: the distro's own
// /proc/meminfo (whose MemTotal *is* the VM's memory limit — WSL sizes the VM
// to it) and the Windows process that backs the VM.

// Meminfo is the distro's /proc/meminfo, in bytes.
type Meminfo struct {
	TotalBytes     int64 `json:"totalBytes"`
	AvailableBytes int64 `json:"availableBytes"`
	CachedBytes    int64 `json:"cachedBytes"`
	SwapTotalBytes int64 `json:"swapTotalBytes"`
	SwapFreeBytes  int64 `json:"swapFreeBytes"`
}

// ParseMeminfo reads the fields the view needs. Unknown lines are ignored;
// a text with no MemTotal is an error, not a zero-memory machine.
func ParseMeminfo(text string) (Meminfo, error) {
	var m Meminfo
	fields := map[string]*int64{
		"MemTotal":     &m.TotalBytes,
		"MemAvailable": &m.AvailableBytes,
		"Cached":       &m.CachedBytes,
		"SwapTotal":    &m.SwapTotalBytes,
		"SwapFree":     &m.SwapFreeBytes,
	}
	seen := false
	for _, line := range strings.Split(text, "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		dst, want := fields[strings.TrimSpace(key)]
		if !want {
			continue
		}
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			continue
		}
		n, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		if len(parts) > 1 && strings.EqualFold(parts[1], "kB") {
			n *= 1024
		}
		*dst = n
		if key == "MemTotal" {
			seen = true
		}
	}
	if !seen {
		return Meminfo{}, fmt.Errorf("no MemTotal in /proc/meminfo")
	}
	return m, nil
}

// HostMemory is the Windows side: the machine's RAM and what the VM's
// process holds of it.
type HostMemory struct {
	TotalBytes int64 `json:"totalBytes"`
	FreeBytes  int64 `json:"freeBytes"`
	// VMWorkingSetBytes is the VM process's resident memory — what Task
	// Manager shows for vmmemWSL. Zero when the VM is not running.
	VMWorkingSetBytes int64 `json:"vmWorkingSetBytes"`
}

// hostMemoryScript prints one JSON object. The VM process is vmmemWSL on
// current Windows; plain vmmem (older builds) is only the fallback, since on
// a new Windows it can be another Hyper-V VM. FreePhysicalMemory is in KiB.
const hostMemoryScript = `$p = Get-Process -Name vmmemWSL -ErrorAction SilentlyContinue | Select-Object -First 1
if (-not $p) { $p = Get-Process -Name vmmem -ErrorAction SilentlyContinue | Select-Object -First 1 }
$cs = Get-CimInstance Win32_ComputerSystem
$os = Get-CimInstance Win32_OperatingSystem
[pscustomobject]@{ total = [int64]$cs.TotalPhysicalMemory; freeKiB = [int64]$os.FreePhysicalMemory; vm = [int64]($(if ($p) { $p.WorkingSet64 } else { 0 })) } | ConvertTo-Json -Compress`

// ReadHostMemory asks Windows for its half.
func ReadHostMemory(r Runner) (HostMemory, error) {
	out, err := r.Output(PowerShellExe, "-NoProfile", "-NonInteractive", "-Command", hostMemoryScript)
	if err != nil && len(out) == 0 {
		return HostMemory{}, fmt.Errorf("read Windows memory: %w", err)
	}
	return parseHostMemory(DecodeWindows(out))
}

func parseHostMemory(text string) (HostMemory, error) {
	start := strings.Index(text, "{")
	if start < 0 {
		return HostMemory{}, fmt.Errorf("no JSON in %q", firstLine(text))
	}
	var raw struct {
		Total   int64 `json:"total"`
		FreeKiB int64 `json:"freeKiB"`
		VM      int64 `json:"vm"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text[start:])), &raw); err != nil {
		return HostMemory{}, fmt.Errorf("memory JSON: %w", err)
	}
	if raw.Total <= 0 {
		return HostMemory{}, fmt.Errorf("memory JSON has no total")
	}
	return HostMemory{TotalBytes: raw.Total, FreeBytes: raw.FreeKiB * 1024, VMWorkingSetBytes: raw.VM}, nil
}

// ReadDistroMemory reads /proc/meminfo inside the distro. No picode needed:
// cat is enough, and it works on a distro whose picode is older.
func ReadDistroMemory(r Runner, distro, user string) (Meminfo, error) {
	out, err := r.Output(WSLExe, WSLArgs(distro, user, "cat", "/proc/meminfo")...)
	if err != nil && len(out) == 0 {
		return Meminfo{}, fmt.Errorf("read /proc/meminfo in %s: %w", distro, err)
	}
	return ParseMeminfo(DecodeWindows(out))
}

// WSLVersions is `wsl --version`, by position: the labels are translated on
// a non-English Windows ("Versão do WSL"), the order is not.
type WSLVersions struct {
	WSL     string `json:"wsl"`
	Kernel  string `json:"kernel"`
	WSLg    string `json:"wslg,omitempty"`
	Windows string `json:"windows,omitempty"`
}

// ParseWSLVersions takes the value after the last ": " of each line. Lines
// 1, 2 and 3 are WSL, kernel and WSLg; the last one is Windows.
func ParseWSLVersions(text string) (WSLVersions, error) {
	var vals []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		i := strings.LastIndex(line, ": ")
		if i < 0 {
			continue
		}
		if v := strings.TrimSpace(line[i+2:]); v != "" {
			vals = append(vals, v)
		}
	}
	// An inbox WSL that does not know --version prints its usage text; a
	// first value that is not a version is that, not a version.
	if len(vals) < 2 {
		return WSLVersions{}, fmt.Errorf("wsl --version printed %q", firstLine(text))
	}
	if _, ok := versionParts(vals[0]); !ok {
		return WSLVersions{}, fmt.Errorf("wsl --version printed %q", firstLine(text))
	}
	v := WSLVersions{WSL: vals[0], Kernel: vals[1]}
	if len(vals) > 2 {
		v.WSLg = vals[2]
	}
	if len(vals) > 3 {
		v.Windows = vals[len(vals)-1]
	}
	return v, nil
}

// ReadWSLVersion runs `wsl --version`. An inbox WSL (the old Windows
// component) does not know the flag; that is reported, not guessed.
func ReadWSLVersion(r Runner) (WSLVersions, error) {
	out, err := r.Output(WSLExe, "--version")
	if err != nil && len(out) == 0 {
		return WSLVersions{}, fmt.Errorf("wsl --version: %w", err)
	}
	return ParseWSLVersions(DecodeWindows(out))
}

// NewerVersion reports whether latest is a later dotted version than
// current ("2.7.1" > "2.7.0.0"). A tag's leading "v" is ignored; anything
// unparsable is "not newer", so a bad answer never nags.
func NewerVersion(latest, current string) bool {
	a, okA := versionParts(latest)
	b, okB := versionParts(current)
	if !okA || !okB {
		return false
	}
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func versionParts(s string) ([]int, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return nil, false
	}
	var out []int
	for _, p := range strings.Split(s, ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

// ShutdownArgs stops the whole WSL VM — every distro. This is what makes a
// .wslconfig change take effect; a terminate of one distro does not.
func ShutdownArgs() []string { return []string{"--shutdown"} }

// UpdateArgs updates WSL from Microsoft's own channel. The update restarts
// the WSL service, so it ends every session like a shutdown.
func UpdateArgs() []string { return []string{"--update"} }
