package desktop

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Moving the distro's disk file to another drive, and backing it up. Both
// copy the whole file, both need the distro stopped, and both can only land
// on a fixed local drive with room for it — checked here before anything
// stops, so a person is told "not enough space on C:" instead of watching a
// copy fail an hour in.

// WinVolume is one ready drive, as Windows reports it.
type WinVolume struct {
	Name      string `json:"name"` // "E:\"
	Letter    string `json:"letter"`
	Fixed     bool   `json:"fixed"`
	Format    string `json:"format"`
	FreeBytes int64  `json:"freeBytes"`
	SizeBytes int64  `json:"sizeBytes"`
}

const volumesScript = `[System.IO.DriveInfo]::GetDrives() | Where-Object { $_.IsReady } | ForEach-Object { [pscustomobject]@{ name = $_.Name; type = [int]$_.DriveType; format = $_.DriveFormat; free = [int64]$_.AvailableFreeSpace; size = [int64]$_.TotalSize } } | ConvertTo-Json -Compress`

// ListVolumes asks Windows for its ready drives.
func ListVolumes(r Runner) ([]WinVolume, error) {
	out, err := r.Output(PowerShellExe, "-NoProfile", "-NonInteractive", "-Command", volumesScript)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("list drives: %w", err)
	}
	return parseVolumes(DecodeWindows(out))
}

func parseVolumes(text string) ([]WinVolume, error) {
	text = strings.TrimSpace(text)
	start := strings.IndexAny(text, "[{")
	if start < 0 {
		return nil, fmt.Errorf("no JSON in %q", firstLine(text))
	}
	text = text[start:]
	// ConvertTo-Json prints a lone object, not a one-element list.
	if text[0] == '{' {
		text = "[" + text + "]"
	}
	var raw []struct {
		Name   string `json:"name"`
		Type   int    `json:"type"`
		Format string `json:"format"`
		Free   int64  `json:"free"`
		Size   int64  `json:"size"`
	}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("drives JSON: %w", err)
	}
	var out []WinVolume
	for _, v := range raw {
		if len(v.Name) < 2 || v.Name[1] != ':' {
			continue
		}
		out = append(out, WinVolume{
			Name: v.Name, Letter: strings.ToUpper(v.Name[:1]),
			Fixed: v.Type == 3, Format: v.Format, FreeBytes: v.Free, SizeBytes: v.Size,
		})
	}
	return out, nil
}

// placeMargin is kept free after the copy: a drive filled to the last byte
// is a Windows that cannot update or page.
const placeMargin = 5 << 30

// folderRe is one plain folder segment: letters, digits, space, _ . -
var folderRe = regexp.MustCompile(`^[A-Za-z0-9 _.-]{1,64}$`)

// reservedName is a Windows device name, which no folder may be called.
var reservedName = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(\..*)?$`)

// validSegment rejects what Win32 would rewrite or refuse: device names,
// dot-only segments, and a trailing dot or space.
func validSegment(s string) bool {
	return folderRe.MatchString(s) && !reservedName.MatchString(s) &&
		strings.Trim(s, ".") != "" && !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, " ")
}

// PlaceCheck is the answer to "can the disk file go to <letter>:\<folder>".
type PlaceCheck struct {
	Path     string `json:"path"`
	Need     int64  `json:"need"`
	Free     int64  `json:"free"`
	OK       bool   `json:"ok"`
	Reason   string `json:"reason,omitempty"`
	SameDisk bool   `json:"sameDisk,omitempty"`
}

// CheckPlace decides whether the file fits at letter:\folder (folder may be
// several plain segments joined with backslashes). need is the file's full
// length — a copy is not promised to stay sparse — plus the margin.
func CheckPlace(vols []WinVolume, facts DiskFacts, letter, folder string) PlaceCheck {
	letter = strings.ToUpper(strings.TrimSpace(letter))
	pc := PlaceCheck{Need: facts.VHDXBytes + placeMargin}
	var vol *WinVolume
	for i := range vols {
		if vols[i].Letter == letter {
			vol = &vols[i]
		}
	}
	if vol == nil {
		pc.Reason = "there is no drive " + letter + ": on this PC"
		return pc
	}
	pc.Free = vol.FreeBytes
	var segs []string
	for _, s := range strings.Split(strings.Trim(folder, `\/`), `\`) {
		s = strings.TrimSpace(s)
		if s == "" || !validSegment(s) {
			pc.Reason = "use a folder name of letters, digits, spaces and _ . - only"
			return pc
		}
		segs = append(segs, s)
	}
	if len(segs) == 0 {
		pc.Reason = "name a folder on " + letter + ":"
		return pc
	}
	pc.Path = letter + `:\` + strings.Join(segs, `\`)
	// Moving within the same drive is a rename: it needs no room.
	if strings.EqualFold(driveOfPath(facts.VHDXPath), letter) {
		pc.Need = 0
	}
	switch {
	case !vol.Fixed:
		pc.Reason = letter + ": is not a fixed local drive (WSL needs one)"
	case !strings.EqualFold(vol.Format, "NTFS"):
		pc.Reason = letter + ": is " + vol.Format + "; WSL disk files need NTFS"
	case strings.EqualFold(driveOfPath(facts.VHDXPath), letter) && sameFolder(facts.VHDXPath, pc.Path):
		pc.Reason = "the disk file is already there"
		pc.SameDisk = true
	case vol.FreeBytes < pc.Need:
		pc.Reason = fmt.Sprintf("%s: has %s free; the disk file needs %s, plus %s left free",
			letter, gib(vol.FreeBytes), gib(facts.VHDXBytes), gib(placeMargin))
	default:
		pc.OK = true
	}
	return pc
}

func driveOfPath(p string) string {
	if len(p) >= 2 && p[1] == ':' {
		return strings.ToUpper(p[:1])
	}
	return ""
}

func sameFolder(vhdx, folder string) bool {
	i := strings.LastIndex(vhdx, `\`)
	return i > 0 && strings.EqualFold(strings.TrimRight(vhdx[:i], `\`), strings.TrimRight(folder, `\`))
}

// gib rounds like the window's own byte formatter (one decimal under 10),
// so a reason and the label beside it never disagree.
func gib(n int64) string {
	v := float64(n) / (1 << 30)
	if v < 10 {
		return fmt.Sprintf("%.1f GB", v)
	}
	return fmt.Sprintf("%.0f GB", v)
}

// KeepaliveTask is the scheduled task that holds the distro open
// (desktop-shell/src/keepalive.rs registers it, ADR-0154). It restarts on
// failure three times a minute apart — so after a terminate it boots the
// distro again within a minute, which a copy that needs the distro stopped
// for an hour cannot survive. The long flows disable it for the copy.
const KeepaliveTask = "PiCodeDistro"

// TaskArgs changes or runs the keepalive task: "/disable", "/enable", or
// "/run".
func TaskArgs(verb string) []string {
	if verb == "/run" {
		return []string{"/run", "/tn", KeepaliveTask}
	}
	return []string{"/change", "/tn", KeepaliveTask, verb}
}

// MoveArgs moves the distro's disk file into folder.
func MoveArgs(distro, folder string) []string {
	return []string{"--manage", distro, "--move", folder}
}

// ExportArgs writes the whole disk as one .vhdx file.
func ExportArgs(distro, file string) []string {
	return []string{"--export", distro, file, "--format", "vhd"}
}
