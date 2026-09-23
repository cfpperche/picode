package desktop

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cfpperche/picode/internal/hostfs"
)

// The closed list of root commands the Management window may run in the
// distro (ADR-0198). Each is an argv built here — no string from the page
// ever becomes a command — and runs as `wsl.exe -d <distro> -u root --exec …`.

// RootArgs runs argv as the distro's root account through `--exec`, which
// hands argv to the distro as argv — no shell. (WSLArgs' `--` form goes
// through one and eats every `$`: measured 2026-09-23, `--exec sh -c '…$1…'`
// keeps `$1`, the `--` form prints it empty.)
func RootArgs(distro string, argv ...string) []string {
	return append([]string{"-d", distro, "-u", "root", "--exec"}, argv...)
}

// WSLConfPath is the distro's settings file.
const WSLConfPath = "/etc/wsl.conf"

// ReadWSLConfArgs prints the file, or nothing when there is none — decided
// by a test, not by reading a (translated) error message.
func ReadWSLConfArgs(distro string) []string {
	return RootArgs(distro, "sh", "-c", `if [ -e /etc/wsl.conf ]; then cat /etc/wsl.conf; fi`)
}

// writeWSLConfScript decodes its one positional argument into a temp file,
// keeps the old file as .bak, and renames the new one into place — an
// interrupted write never leaves a half file. The content arrives as $1,
// base64: it is data to base64 -d, never shell syntax.
//
// Two copies of what was there: /etc/wsl.conf.picode-orig, written once on
// the first save and never again (the file as it was before PiCode touched
// it), and .bak, the previous save. A symlinked wsl.conf is refused — a
// rename would silently replace the link with a plain file.
const writeWSLConfScript = `set -e
if [ -L /etc/wsl.conf ]; then echo "/etc/wsl.conf is a symbolic link; edit its target by hand" >&2; exit 3; fi
tmp=/etc/wsl.conf.picode-new
printf '%s' "$1" | base64 -d > "$tmp"
chmod 644 "$tmp"
if [ -f /etc/wsl.conf ]; then
  if [ ! -e /etc/wsl.conf.picode-orig ]; then cat /etc/wsl.conf > /etc/wsl.conf.picode-orig; fi
  cat /etc/wsl.conf > /etc/wsl.conf.bak
fi
mv "$tmp" /etc/wsl.conf
sync`

// WriteWSLConfArgs writes content as the new /etc/wsl.conf.
func WriteWSLConfArgs(distro, content string) []string {
	return RootArgs(distro, "sh", "-c", writeWSLConfScript, "picode-wslconf",
		base64.StdEncoding.EncodeToString([]byte(content)))
}

// SystemCache is one root-owned cache: where it lives and the tool's own
// prune, as a fixed argv.
type SystemCache struct {
	ID    string      `json:"id"`
	Title string      `json:"title"`
	Kind  hostfs.Kind `json:"kind"`
	Note  string      `json:"note"`
	Paths []string    `json:"paths"`
	Prune []string    `json:"-"`
}

// SystemCaches is the table. Ids carry the "system:" prefix so they can
// never collide with — or be routed to — the person's own cache table.
var SystemCaches = []SystemCache{
	{
		ID: "system:apt", Title: "apt package cache", Kind: hostfs.KindRedownload,
		Note:  "Downloaded .deb packages; the next install downloads what it needs.",
		Paths: []string{"/var/cache/apt/archives"},
		Prune: []string{"apt-get", "clean"},
	},
	{
		ID: "system:journal", Title: "System logs (journal)", Kind: hostfs.KindSafe,
		// --rotate first: vacuum only deletes archived files, so without it
		// the active log would stay whatever size it is.
		Note:  "Deletes older system logs, keeping about the newest 100 MB.",
		Paths: []string{"/var/log/journal"},
		Prune: []string{"journalctl", "--rotate", "--vacuum-size=100M"},
	},
}

// SystemCacheByID finds a table row. Anything else is refused.
func SystemCacheByID(id string) (SystemCache, bool) {
	for _, c := range SystemCaches {
		if c.ID == id {
			return c, true
		}
	}
	return SystemCache{}, false
}

// MeasuredSystemCache is a row with its size now.
type MeasuredSystemCache struct {
	SystemCache
	Bytes int64 `json:"bytes"`
}

// measureScript sizes each existing path; a path this distro does not have
// (no apt on Fedora) is skipped, so it measures 0 instead of failing the
// whole list. The paths are the table's own literals, passed as $@.
const measureScript = `for p in "$@"; do if [ -e "$p" ]; then du -sB1 -- "$p"; fi; done`

// MeasureSystemCaches runs one `du` pass as root over every table path.
func MeasureSystemCaches(r Runner, distro string) ([]MeasuredSystemCache, error) {
	args := []string{"sh", "-c", measureScript, "picode-measure"}
	for _, c := range SystemCaches {
		args = append(args, c.Paths...)
	}
	out, err := r.Output(WSLExe, RootArgs(distro, args...)...)
	sizes := hostfs.ParseDU([]byte(DecodeWindows(out)))
	if err != nil && len(sizes) == 0 {
		return nil, fmt.Errorf("measure system caches as root: %w", err)
	}
	var res []MeasuredSystemCache
	for _, c := range SystemCaches {
		var n int64
		for _, p := range c.Paths {
			n += sizes[p]
		}
		res = append(res, MeasuredSystemCache{SystemCache: c, Bytes: n})
	}
	return res, nil
}

// PruneSystemCache runs the row's own prune as root.
func PruneSystemCache(r Runner, distro string, c SystemCache) error {
	if err := r.Run(WSLExe, RootArgs(distro, c.Prune...)...); err != nil {
		return fmt.Errorf("%s: %w", strings.Join(c.Prune, " "), err)
	}
	return nil
}
