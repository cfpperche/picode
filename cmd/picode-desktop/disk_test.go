package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/hostfs"
)

// diskStub answers the calls collectDisk makes, in order, and records the argv
// so the command shapes can be asserted too.
type diskStub struct {
	replies [][]byte
	errs    []error
	calls   [][]string
}

func (s *diskStub) Output(name string, args ...string) ([]byte, error) {
	s.calls = append(s.calls, append([]string{name}, args...))
	i := len(s.calls) - 1
	var out []byte
	if i < len(s.replies) {
		out = s.replies[i]
	}
	var err error
	if i < len(s.errs) {
		err = s.errs[i]
	}
	return out, err
}

func (s *diskStub) Run(name string, args ...string) error {
	_, err := s.Output(name, args...)
	return err
}

func wide(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(units)*2)
	for _, u := range units {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}

const diskLxss = `
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Lxss\{c2af16cd-4a9b-4d07-94bd-1c82c914c39e}
    DistributionName    REG_SZ    Ubuntu
    BasePath    REG_SZ    C:\Users\cfpp\AppData\Local\wsl\{c2af16cd-4a9b-4d07-94bd-1c82c914c39e}
    VhdFileName    REG_SZ    ext4.vhdx
`

// reportJSON is what `picode disk --json` prints inside the distro.
const reportJSON = `{
  "fs": {"sizeBytes": 1081101176832, "usedBytes": 134980124672, "availBytes": 891128696832, "mount": "/"},
  "homeDir": "/home/goat",
  "consumers": [
    {"id": "go-build", "title": "Go build cache", "paths": ["/home/goat/.cache/go-build"], "kind": "safe",
     "how": "go clean -cache", "note": "The next build is slower; nothing else changes.", "bytes": 32317198336}
  ],
  "home": [{"path": "/home/goat/.cache", "bytes": 38700000000}],
  "measured": "2026-09-11T19:00:00Z",
  "other": 1000
}`

// TestCollectDiskJoinsBothHalves is the whole point of the command: the number
// a person needs is the difference between what Windows holds and what the
// distro uses, and neither half can produce it alone.
func TestCollectDiskJoinsBothHalves(t *testing.T) {
	r := &diskStub{replies: [][]byte{
		wide(diskLxss),
		[]byte(`{"len":233711861760,"total":510964789248,"free":29251764224}`),
		wide("The specified file is NOT sparse\r\n"),
		wide("WSL version: 2.7.0.0\r\n"),
		wide("--set-sparse\n"),
		[]byte("/home/goat/.local/bin/picode\n"),
		[]byte(reportJSON),
	}}
	rep := collectDisk(app{runner: r, distro: "Ubuntu", user: "goat"}, nil)

	if rep.WindowsError != "" || rep.Windows == nil {
		t.Fatalf("Windows half: %v", rep.WindowsError)
	}
	if rep.DistroError != "" || rep.Distro == nil {
		t.Fatalf("distro half: %v", rep.DistroError)
	}
	if want := int64(233711861760 - 134980124672); rep.Held != want {
		t.Errorf("Held = %d, want %d", rep.Held, want)
	}
	if rep.At.Location() != time.UTC {
		t.Errorf("the stamp is local: %v", rep.At)
	}

	// The distro half is asked of the binary inside the distro, by absolute
	// path, with the account the caller resolved.
	last := r.calls[len(r.calls)-1]
	want := []string{desktop.WSLExe, "-d", "Ubuntu", "-u", "goat", "--", "/home/goat/.local/bin/picode", "disk", "--json"}
	if strings.Join(last, " ") != strings.Join(want, " ") {
		t.Errorf("distro call:\n got %v\nwant %v", last, want)
	}
}

// TestCollectDiskKeepsEachFailureWithItsHalf: a report that cannot read one
// side must still show the other, and say which one failed.
func TestCollectDiskKeepsEachFailureWithItsHalf(t *testing.T) {
	r := &diskStub{replies: [][]byte{
		wide(diskLxss),
		[]byte(`{"len":1,"total":2,"free":1}`),
		wide("not sparse\r\n"),
		wide("WSL version: 2.7.0.0\r\n"),
		wide("--set-sparse\n"),
		[]byte("/home/goat/.local/bin/picode\n"),
		[]byte("usage: picode ...\n"),
	}}
	rep := collectDisk(app{runner: r, distro: "Ubuntu", user: "goat"}, nil)
	if rep.Windows == nil {
		t.Fatalf("the Windows half is independent: %v", rep.WindowsError)
	}
	if rep.DistroError == "" {
		t.Fatal("a distro half that printed no JSON is a failure, not a zero-byte disk")
	}
	if rep.Held != 0 {
		t.Errorf("Held needs both halves: got %d", rep.Held)
	}
}

func TestDistroUsedReadsOneNumber(t *testing.T) {
	//  The real shape: a header the parser ignores and one row.
	out := wide("    1B-blocks         Used        Avail Target\n" +
		"1081101176832 134980124672 891128696832 /\n")
	r := &diskStub{replies: [][]byte{out}}
	used, err := distroUsed(r, "Ubuntu", "goat")
	if err != nil {
		t.Fatalf("distroUsed: %v", err)
	}
	if used != 134980124672 {
		t.Errorf("used = %d", used)
	}
}

// TestDistroReportIgnoresANoiseBanner covers a login shell that greets before
// the JSON: the parse starts at the first brace, not at the first byte.
func TestDistroReportIgnoresANoiseBanner(t *testing.T) {
	r := &diskStub{replies: [][]byte{
		[]byte("/home/goat/.local/bin/picode\n"),
		wide("Welcome to Ubuntu 26.04 LTS\n" + reportJSON + "\n"),
	}}
	rep, err := distroReport(r, "Ubuntu", "goat")
	if err != nil {
		t.Fatalf("distroReport: %v", err)
	}
	if rep.FS.UsedBytes != 134980124672 || rep.HomeDir != "/home/goat" {
		t.Errorf("report: %+v", rep)
	}
}

// TestDiskLineIsTheSentenceAPersonReads pins the line's wording: the numbers, in
// the order that answers "is my disk full", and no line at all invented when
// nothing could be read.
func TestDiskLineIsTheSentenceAPersonReads(t *testing.T) {
	healthy := desktop.DiskFacts{
		VHDXPath:       `C:\wsl\Ubuntu\ext4.vhdx`,
		AllocatedBytes: 233711861760,
		FreeBytes:      29251764224,
	}
	giB := func(n int64) int64 { return n << 30 }

	cases := []struct {
		name    string
		facts   *desktop.DiskFacts
		used    int64
		err     error
		want    string
		wantWrn bool
	}{
		{
			name:    "a healthy machine says so quietly",
			facts:   &healthy,
			used:    233711861760,
			want:    "WSL 218 GB · C: 27 GB free",
			wantWrn: false,
		},
		{
			name:    "a low volume is a warning",
			facts:   &desktop.DiskFacts{AllocatedBytes: 233711861760, FreeBytes: 5 << 30},
			used:    233711861760,
			want:    "WSL 218 GB · C: 5.0 GB free — low",
			wantWrn: true,
		},
		{
			name:    "held space is worth naming",
			facts:   &healthy,
			used:    giB(126),
			want:    "WSL 218 GB · ≈92 GB held by Windows · C: 27 GB free",
			wantWrn: false,
		},
		{
			name: "an unread disk is not a zero",
			err:  fmt.Errorf("wsl is not installed"),
			want: "Disk: not read",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, warn := diskLine(c.facts, c.used, c.err)
			if got != c.want {
				t.Errorf("title:\n got %q\nwant %q", got, c.want)
			}
			if warn != c.wantWrn {
				t.Errorf("warn = %v, want %v", warn, c.wantWrn)
			}
		})
	}
}

func TestDriveOf(t *testing.T) {
	cases := map[string]string{
		`C:\Users\cfpp\AppData\Local\wsl\{guid}\ext4.vhdx`: "C",
		`E:\WSL\Debian\ext4.vhdx`:                          "E",
		``:                                                 "C",
		`relative\path.vhdx`:                               "C",
	}
	for in, want := range cases {
		if got := driveOf(in); got != want {
			t.Errorf("driveOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestHeldNeverOverstates is the arithmetic behind the one number that has to
// be safe to act on: an unmeasured distro, or a sparse file already given
// back, must never look like reclaimable space.
func TestHeldNeverOverstates(t *testing.T) {
	f := desktop.DiskFacts{AllocatedBytes: 100}
	if got := desktop.Held(f, 0); got != 0 {
		t.Errorf("unknown usage: %d", got)
	}
	if got := desktop.Held(f, 200); got != 0 {
		t.Errorf("usage above the allocation: %d", got)
	}
	if got := desktop.Held(f, 40); got != 60 {
		t.Errorf("got %d, want 60", got)
	}
}

// TestReclaimTotalsIgnoreData keeps the promise the report makes in words:
// Pi's sessions and PiCode's own data are shown, never counted as reclaimable.
func TestReclaimTotalsIgnoreData(t *testing.T) {
	report := hostfs.Report{Consumers: []hostfs.Measured{
		{Consumer: hostfs.Consumer{Kind: hostfs.KindSafe}, Bytes: 30},
		{Consumer: hostfs.Consumer{Kind: hostfs.KindData}, Bytes: 700},
	}}
	if got := hostfs.Reclaimable(report.Consumers, hostfs.KindSafe, hostfs.KindRedownload); got != 30 {
		t.Errorf("reclaimable = %d, want 30", got)
	}
}

// TestCollectDiskStreamsEachHalf is the Management window's scan: each half
// announces itself, then lands with its data (or its error) before the next
// one starts, so the first card fills while the slow walk still runs.
func TestCollectDiskStreamsEachHalf(t *testing.T) {
	r := &diskStub{replies: [][]byte{
		wide(diskLxss),
		[]byte(`{"len":1,"total":2,"free":1}`),
		wide("not sparse\r\n"),
		wide("WSL version: 2.7.0.0\r\n"),
		wide("--set-sparse\n"),
		[]byte("/home/goat/.local/bin/picode\n"),
		[]byte("usage: picode ...\n"),
	}}
	var steps []scanStep
	collectDisk(app{runner: r, distro: "Ubuntu", user: "goat"}, func(s scanStep) { steps = append(steps, s) })

	var got []string
	for _, s := range steps {
		got = append(got, s.Stage+":"+s.State)
		if s.Progress == "" {
			t.Errorf("%s:%s has no progress line — the shell would take it for the outcome", s.Stage, s.State)
		}
	}
	want := "windows:running windows:done distro:running distro:failed"
	if strings.Join(got, " ") != want {
		t.Fatalf("steps = %v, want %s", got, want)
	}
	if steps[1].Windows == nil {
		t.Error("the windows:done step carries no facts — the card cannot fill early")
	}
	if steps[3].Error == "" {
		t.Error("the distro:failed step carries no error")
	}
}

// TestNoBareExecCommand guards the headless promise: this program links as a
// GUI app, so any console child spawned outside newCmd opens a visible
// terminal window over the app (the Management scan did, through clean.go).
func TestNoBareExecCommand(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || strings.HasPrefix(f, "newcmd_") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, "exec.Command(") || strings.Contains(line, "exec.CommandContext(") {
				t.Errorf("%s:%d spawns without newCmd — the child gets a console window: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}
