// Package hostfs measures what occupies the machine PiCode runs on — the
// numbers a person would otherwise collect by hand from `df` and `du` — and
// names, for every consumer it knows, the action that gives that space back.
//
// Two rules shape it. Every fact from outside Go arrives through a Runner, so
// the parsing and the arithmetic are testable on a machine that has neither
// tool (the Windows CI job compiles and runs this package). And nothing here
// changes anything: the reclaim commands are text in a table until a person
// reviews and confirms them (docs/plans/wsl-control.md).
package hostfs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Runner is the process boundary. `du` and `df` are the only things this
// package cannot do in Go without walking every inode itself.
type Runner interface {
	Output(name string, args ...string) ([]byte, error)
}

// Exec is the Runner the CLI and the daemon use: a child process in a clean,
// English locale, with a bounded life so a walk over a cold tree cannot hold
// a request open forever.
type Exec struct {
	// Timeout bounds one measurement. Zero means two minutes, which is
	// generous for the worst case on a slow spinning disk and short enough
	// that a wedged mount does not become a hung page.
	Timeout time.Duration
}

func (e Exec) Output(name string, args ...string) ([]byte, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	// Messages are for the person reading a failure, and the parse never
	// depends on a header, so one predictable locale is right here.
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return out, err
	}
	// `du` exits non-zero when one path in a list is missing and still prints
	// the sizes it did read, which is the useful half.
	return out, err
}

// Kind is what giving a consumer back costs the person.
type Kind string

const (
	// KindSafe costs only time: the next build is slower, nothing is lost.
	KindSafe Kind = "safe"
	// KindRedownload comes back from the network on the next use.
	KindRedownload Kind = "redownload"
	// KindData is the person's own data. Shown, never offered as a command.
	KindData Kind = "data"
)

// Consumer is one measured thing with a name: what it is, where it lives, and
// how the space comes back. Adding a row is the whole change — nothing else
// knows a consumer by anything but this table.
type Consumer struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Paths []string `json:"paths"`
	Kind  Kind     `json:"kind"`
	// How is the command that gives the space back, exactly as a person would
	// type it. Empty for KindData, which has no safe one-liner.
	How string `json:"how,omitempty"`
	// Note is what it costs, in one line, for a person who has never run the
	// command before.
	Note string `json:"note"`
}

// Measured is a consumer with its size on this machine.
type Measured struct {
	Consumer
	Bytes int64 `json:"bytes"`
}

// Entry is one measured path: the top-level breakdown of a home directory.
type Entry struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// FS is one mounted filesystem, in bytes.
type FS struct {
	SizeBytes  int64  `json:"sizeBytes"`
	UsedBytes  int64  `json:"usedBytes"`
	AvailBytes int64  `json:"availBytes"`
	Mount      string `json:"mount"`
}

// Report is one measurement of a machine.
type Report struct {
	FS        FS         `json:"fs"`
	HomeDir   string     `json:"homeDir"`
	Consumers []Measured `json:"consumers"`
	Home      []Entry    `json:"home"`
	Measured  time.Time  `json:"measured"`
	// Other is what UsedBytes does not account for: paths owned by another
	// account (docker's /var/lib/docker is the usual one on a WSL box) and
	// anything outside the home directory. It is the honest remainder, not a
	// category — the point of printing it is that the numbers add up.
	Other int64 `json:"other"`
}

// dfArgs asks for the numbers in bytes for every mount point. The whole list
// is needed, not just the root: "is this path on the filesystem I measured"
// is answered by the longest mount point that contains it, and a second disk
// mounted under /mnt must not be subtracted from the root's total. The parse
// skips any line whose first field is not a number, so a localized header is
// not a failure mode.
var dfArgs = []string{"-B1", "--output=size,used,avail,target"}

// Measure reads the filesystem numbers, the reclaim table and the top-level
// breakdown of home. It changes nothing.
func Measure(r Runner, home string) (Report, error) {
	out, err := r.Output("df", dfArgs...)
	if err != nil {
		return Report{}, fmt.Errorf("df: %w", err)
	}
	rows := ParseDF(out)
	fs, err := Root(rows)
	if err != nil {
		return Report{}, err
	}
	mounts := make([]string, 0, len(rows))
	for _, row := range rows {
		mounts = append(mounts, row.Mount)
	}

	entries, err := os.ReadDir(home)
	if err != nil {
		return Report{}, fmt.Errorf("read %s: %w", home, err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, filepath.Join(home, e.Name()))
	}

	consumers := Consumers(home)
	var consumerPaths []string
	for _, c := range consumers {
		consumerPaths = append(consumerPaths, c.Paths...)
	}

	// Two `du` calls, not one. `du` skips a path it has already visited, so
	// asking for ~/.cache and ~/.cache/go-build in the same command reports the
	// parent with that child left out — a breakdown missing the biggest
	// directory in it. Each list is measured on its own.
	consumerSizes, err := sizesOf(r, consumerPaths)
	if err != nil {
		return Report{}, err
	}
	homeSizes, err := sizesOf(r, names)
	if err != nil {
		return Report{}, err
	}
	sizes := make(map[string]int64, len(consumerSizes)+len(homeSizes))
	for p, n := range consumerSizes {
		sizes[p] = n
	}
	for p, n := range homeSizes {
		sizes[p] = n
	}
	return Build(fs, consumers, sizes, mounts, home, names, time.Now().UTC()), nil
}

// Build is the pure half: sizes in, report out. mounts is every mount point
// the machine reported, so a home directory that lives on another filesystem
// is counted in the breakdown and never subtracted from this one's total.
func Build(fs FS, consumers []Consumer, sizes map[string]int64, mounts []string, home string, homePaths []string, at time.Time) Report {
	rep := Report{FS: fs, HomeDir: home, Measured: at, Home: []Entry{}, Consumers: []Measured{}}

	for _, c := range consumers {
		var total int64
		for _, p := range c.Paths {
			total += sizes[p]
		}
		// A consumer that is not on this machine is not a zero-byte row.
		if total > 0 {
			rep.Consumers = append(rep.Consumers, Measured{Consumer: c, Bytes: total})
		}
	}
	sort.SliceStable(rep.Consumers, func(i, j int) bool { return rep.Consumers[i].Bytes > rep.Consumers[j].Bytes })

	var accounted int64
	for _, p := range homePaths {
		n := sizes[p]
		if n == 0 {
			continue
		}
		rep.Home = append(rep.Home, Entry{Path: p, Bytes: n})
		if mountOf(p, mounts) == fs.Mount {
			accounted += n
		}
	}
	sort.SliceStable(rep.Home, func(i, j int) bool { return rep.Home[i].Bytes > rep.Home[j].Bytes })

	// Only a consumer outside home adds to the accounted total: one inside home
	// is already inside a top-level entry, and counting it twice would make the
	// machine look smaller than the numbers say it is.
	for _, m := range rep.Consumers {
		for _, p := range m.Consumer.Paths {
			if mountOf(p, mounts) == fs.Mount && !within(p, home) {
				accounted += sizes[p]
			}
		}
	}
	if rest := fs.UsedBytes - accounted; rest > 0 {
		rep.Other = rest
	}
	return rep
}

// mountOf answers which filesystem a path is on: the longest mount point that
// contains it, the way the kernel resolves it.
func mountOf(path string, mounts []string) string {
	best := ""
	for _, m := range mounts {
		if within(path, m) && len(m) > len(best) {
			best = m
		}
	}
	return best
}

// within reports whether path is dir or lives inside it. The root is its own
// case: handlng it as "dir + separator" would ask for a leading "//".
func within(path, dir string) bool {
	if path == dir {
		return true
	}
	if dir == "" {
		return false
	}
	if strings.HasSuffix(dir, "/") {
		return strings.HasPrefix(path, dir)
	}
	return strings.HasPrefix(path, dir+"/")
}

// Reclaimable sums the consumers of the given kinds — what a person could
// have back without losing anything they cannot get again.
func Reclaimable(list []Measured, kinds ...Kind) int64 {
	want := map[Kind]bool{}
	for _, k := range kinds {
		want[k] = true
	}
	var total int64
	for _, m := range list {
		if want[m.Kind] {
			total += m.Bytes
		}
	}
	return total
}

// Consumers is the reclaim table. Every path is built from home so the table
// is one literal list on every machine, and every How is a command that
// exists in the tool it names.
func Consumers(home string) []Consumer {
	at := func(rel string) []string { return []string{filepath.Join(home, rel)} }
	rm := func(rel string) string { return "rm -rf " + filepath.Join(home, rel) }

	return []Consumer{
		{
			ID: "go-build", Title: "Go build cache", Paths: at(".cache/go-build"),
			Kind: KindSafe, How: "go clean -cache",
			Note: "The next build is slower; nothing else changes.",
		},
		{
			ID: "go-mod", Title: "Go module cache", Paths: at("go/pkg/mod"),
			Kind: KindRedownload, How: "go clean -modcache",
			Note: "Modules download again on the next build.",
		},
		{
			ID: "npm", Title: "npm cache", Paths: at(".npm"),
			Kind: KindRedownload, How: "npm cache clean --force",
			Note: "Packages download again on the next install.",
		},
		{
			ID: "pnpm", Title: "pnpm store", Paths: at(".cache/pnpm"),
			Kind: KindRedownload, How: "pnpm store prune",
			Note: "Prunes only what no project references.",
		},
		{
			ID: "uv", Title: "uv cache", Paths: at(".cache/uv"),
			Kind: KindRedownload, How: "uv cache prune",
			Note: "Wheels download again on the next install.",
		},
		{
			ID: "playwright", Title: "Playwright browsers", Paths: at(".cache/ms-playwright"),
			Kind: KindRedownload, How: rm(".cache/ms-playwright"),
			Note: "Browser tests re-download the engines they need.",
		},
		{
			ID: "huggingface", Title: "Hugging Face cache", Paths: at(".cache/huggingface"),
			Kind: KindRedownload, How: rm(".cache/huggingface"),
			Note: "Models download again on the next run.",
		},
		{
			ID: "picode-data", Title: "PiCode data", Paths: []string{DataDir(home)},
			Kind: KindData,
			Note: "The database, history and captures. Cleanup belongs to the app and to backups, never to a delete.",
		},
		{
			ID: "pi-sessions", Title: "Pi sessions", Paths: at(".pi"),
			Kind: KindData,
			Note: "Pi's own sessions and credentials. Never delete this by hand.",
		},
	}
}

// DataDir is where PiCode keeps its state, honouring PICODE_DATA the way the
// server does, so a machine that moved it is measured where it actually is.
func DataDir(home string) string {
	if dir := os.Getenv("PICODE_DATA"); dir != "" {
		return dir
	}
	return filepath.Join(home, ".picode")
}

// sizesOf runs one `du` and maps path to bytes. A path that does not exist is
// missing from the map, which is how a consumer that is not installed stays
// out of the report.
func sizesOf(r Runner, paths []string) (map[string]int64, error) {
	if len(paths) == 0 {
		return map[string]int64{}, nil
	}
	args := append([]string{"-sB1", "--"}, paths...)
	out, err := r.Output("du", args...)
	sizes := ParseDU(out)
	if len(sizes) == 0 {
		if err != nil {
			return nil, fmt.Errorf("du: %w", err)
		}
		return nil, fmt.Errorf("du: no sizes in the output")
	}
	return sizes, nil
}

// ParseDF reads `df -B1 --output=size,used,avail,target` into one row per
// mount point. The header is skipped by shape rather than by text: a line
// whose first three fields are not integers is not a filesystem.
func ParseDF(out []byte) []FS {
	var rows []FS
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		size, err1 := strconv.ParseInt(f[0], 10, 64)
		used, err2 := strconv.ParseInt(f[1], 10, 64)
		avail, err3 := strconv.ParseInt(f[2], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		mount := "/"
		if len(f) > 3 {
			mount = strings.Join(f[3:], " ")
		}
		rows = append(rows, FS{SizeBytes: size, UsedBytes: used, AvailBytes: avail, Mount: mount})
	}
	return rows
}

// Root is the filesystem this package measures: the one mounted at /.
func Root(rows []FS) (FS, error) {
	for _, row := range rows {
		if row.Mount == "/" {
			return row, nil
		}
	}
	return FS{}, fmt.Errorf("df: no filesystem is mounted at /")
}

// ParseDU reads `du -sB1`: one "<bytes>\t<path>" line per path it could read.
func ParseDU(out []byte) map[string]int64 {
	sizes := map[string]int64{}
	for _, line := range strings.Split(string(out), "\n") {
		tab := strings.IndexByte(line, '\t')
		if tab <= 0 {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(line[:tab]), 10, 64)
		if err != nil {
			continue
		}
		sizes[line[tab+1:]] = n
	}
	return sizes
}

// Bytes renders a size the way Windows Explorer does: binary units under the
// familiar label. The tray and the File Explorer then agree on one disk,
// which matters more here than the 7% an SI label would claim.
func Bytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	v := float64(n) / unit
	i := 0
	for v >= unit && i < len(units)-1 {
		v /= unit
		i++
	}
	if v < 10 {
		return fmt.Sprintf("%.1f %s", v, units[i])
	}
	return fmt.Sprintf("%.0f %s", v, units[i])
}

// FirstLine keeps an error one line long, the way every other failure in
// PiCode is reported.
func FirstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
