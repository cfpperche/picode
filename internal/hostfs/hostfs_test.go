package hostfs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestParseDFIgnoresTheHeaderAndKeepsEveryMount pins the reason the parse does
// not read the first line: the header is whatever locale the machine speaks,
// and the numbers are not. A Portuguese df is the same three integers. Every
// mount point is kept because the report has to tell a second disk (or /mnt/c)
// from the root filesystem when it adds up what it accounted for.
func TestParseDFIgnoresTheHeaderAndKeepsEveryMount(t *testing.T) {
	out := "    1B-blocks         Used        Avail Target\n" +
		"1081101176832 134152687616 891956133888 /\n" +
		"510964789248 481713025024 29251764224 /mnt/c\n" +
		"1024 512 512 /mnt/my disk\n"

	rows := ParseDF([]byte(out))
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (%+v)", len(rows), rows)
	}
	root, err := Root(rows)
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	want := FS{SizeBytes: 1081101176832, UsedBytes: 134152687616, AvailBytes: 891956133888, Mount: "/"}
	if root != want {
		t.Errorf("got %+v, want %+v", root, want)
	}
	if rows[2].Mount != "/mnt/my disk" {
		t.Errorf("a mount point with a space: got %q", rows[2].Mount)
	}

	// A localized header is not a row either.
	pt := ParseDF([]byte("   Blocos-1B       Usado  Disponível Montado em\n100 60 40 /\n"))
	if len(pt) != 1 || pt[0].UsedBytes != 60 {
		t.Errorf("portuguese header: %+v", pt)
	}
	if _, err := Root(ParseDF([]byte("df: /: No such file or directory\n"))); err == nil {
		t.Error("a df with no root row must fail, not report a zero filesystem")
	}
}

func TestParseDUSkipsWhatItCannotRead(t *testing.T) {
	out := "32317198336\t/home/goat/.cache/go-build\n6210400256\t/home/goat/.npm\n" +
		"du: cannot access '/home/goat/nope': No such file or directory\n"
	got := ParseDU([]byte(out))
	if got["/home/goat/.cache/go-build"] != 32317198336 {
		t.Errorf("go-build: got %d", got["/home/goat/.cache/go-build"])
	}
	if got["/home/goat/.npm"] != 6210400256 {
		t.Errorf("npm: got %d", got["/home/goat/.npm"])
	}
	if len(got) != 2 {
		t.Errorf("expected exactly the two readable paths, got %v", got)
	}
}

// TestBuildKeepsTheTotalsHonest is the arithmetic that decides whether the
// numbers a person reads can be added up: a consumer inside home must not be
// subtracted twice, and a consumer outside home must be not be counted twice
// either — it is the one case where the report would understate the machine.
func TestBuildKeepsTheTotalsHonest(t *testing.T) {
	home := "/home/goat"
	inside := filepath.Join(home, ".cache/go-build")
	outside := "/var/lib/docker"

	fs := FS{SizeBytes: 1000, UsedBytes: 900, AvailBytes: 100, Mount: "/"}
	sizes := map[string]int64{
		inside:                        300,
		outside:                       200,
		filepath.Join(home, ".cache"): 400,
		filepath.Join(home, "work"):   100,
	}
	consumers := []Consumer{
		{ID: "go-build", Title: "Go build cache", Paths: []string{inside}, Kind: KindSafe},
		{ID: "docker", Title: "Docker", Paths: []string{outside}, Kind: KindData},
		{ID: "absent", Title: "Not installed", Paths: []string{"/nope"}, Kind: KindSafe},
	}
	rep := Build(fs, consumers, sizes, []string{"/"}, home, []string{filepath.Join(home, ".cache"), filepath.Join(home, "work")}, time.Unix(0, 0))

	// The absent consumer is not a zero-byte row.
	if len(rep.Consumers) != 2 {
		t.Fatalf("consumers: got %d rows, want 2 (%+v)", len(rep.Consumers), rep.Consumers)
	}
	if rep.Consumers[0].ID != "go-build" {
		t.Errorf("largest first: got %s", rep.Consumers[0].ID)
	}
	// Home: the biggest entry first.
	if rep.Home[0].Path != filepath.Join(home, ".cache") {
		t.Errorf("home order: got %+v", rep.Home)
	}
	// used 900 − (.cache 400 + work 100 on this mount) − docker 200 = 200.
	if rep.Other != 200 {
		t.Errorf("Other: got %d, want 200", rep.Other)
	}
}

// TestBuildIgnoresHomeOnAnotherFilesystem covers a home directory mounted from
// somewhere else (a second disk, or /mnt/c): its entries are still worth
// showing, but they must not be subtracted from the filesystem they are not
// on — that would make the machine look smaller than the numbers say.
func TestBuildIgnoresHomeOnAnotherFilesystem(t *testing.T) {
	fs := FS{SizeBytes: 1000, UsedBytes: 900, AvailBytes: 100, Mount: "/"}
	sizes := map[string]int64{"/mnt/e/home/work": 800}
	rep := Build(fs, nil, sizes, []string{"/", "/mnt/e"}, "/mnt/e/home", []string{"/mnt/e/home/work"}, time.Unix(0, 0))
	if len(rep.Home) != 1 || rep.Home[0].Bytes != 800 {
		t.Errorf("the breakdown still shows it: %+v", rep.Home)
	}
	if rep.Other != 900 {
		t.Errorf("Other: got %d, want the whole filesystem", rep.Other)
	}
}

func TestBuildLeavesOtherAtZeroWhenMeasurementsOvershoot(t *testing.T) {
	// A file created and deleted between the df and the du: the remainder
	// must stay a floor at zero rather than a negative "other".
	fs := FS{SizeBytes: 100, UsedBytes: 10, AvailBytes: 90, Mount: "/"}
	sizes := map[string]int64{"/home/x/a": 500}
	rep := Build(fs, nil, sizes, []string{"/"}, "/home/x", []string{"/home/x/a"}, time.Unix(0, 0))
	if rep.Other != 0 {
		t.Errorf("Other: got %d, want 0", rep.Other)
	}
}

func TestReclaimableCountsOnlyTheKindsAsked(t *testing.T) {
	list := []Measured{
		{Consumer: Consumer{Kind: KindSafe}, Bytes: 10},
		{Consumer: Consumer{Kind: KindRedownload}, Bytes: 20},
		{Consumer: Consumer{Kind: KindData}, Bytes: 40},
	}
	if got := Reclaimable(list, KindSafe); got != 10 {
		t.Errorf("safe: got %d", got)
	}
	if got := Reclaimable(list, KindSafe, KindRedownload); got != 30 {
		t.Errorf("safe+redownload: got %d", got)
	}
	if got := Reclaimable(list); got != 0 {
		t.Errorf("no kinds means nothing: got %d", got)
	}
}

func TestBytesMatchesExplorerLabels(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{999, "999 B"},
		{1024, "1.0 KB"},
		{217660000000, "203 GB"}, // 202.7 GiB: what Explorer calls 203 GB
		{27 * 1024 * 1024 * 1024, "27 GB"},
	}
	for _, c := range cases {
		if got := Bytes(c.in); got != c.want {
			t.Errorf("Bytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// fakeRunner answers the two tools with fixed numbers, so Measure can be
// exercised end to end without spending a minute walking a real tree.
type fakeRunner struct {
	sizes map[string]int64
	df    string
	calls []string
}

func (f *fakeRunner) Output(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	switch name {
	case "df":
		return []byte(f.df), nil
	case "du":
		var b strings.Builder
		for _, a := range args {
			if !strings.HasPrefix(a, "/") {
				continue
			}
			if n, ok := f.sizes[a]; ok {
				fmt.Fprintf(&b, "%d\t%s\n", n, a)
			}
		}
		return []byte(b.String()), nil
	}
	return nil, fmt.Errorf("unexpected tool %q", name)
}

func TestMeasureReadsTheMachineInTwoCalls(t *testing.T) {
	t.Setenv("PICODE_DATA", "")
	home := t.TempDir()
	for _, dir := range []string{".cache/go-build", "work"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	r := &fakeRunner{
		df: "    1B-blocks         Used        Avail Target\n1000 900 100 /\n",
		sizes: map[string]int64{
			filepath.Join(home, ".cache"):          400,
			filepath.Join(home, "work"):            100,
			filepath.Join(home, ".cache/go-build"): 300,
			filepath.Join(home, ".picode"):         50,
		},
	}
	rep, err := Measure(r, home)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if rep.FS.UsedBytes != 900 || rep.FS.Mount != "/" {
		t.Errorf("filesystem: %+v", rep.FS)
	}
	// Only the two real directories were measured; .cache/go-build and
	// .picode are consumers, not top-level entries of this home's listing.
	if len(rep.Home) != 2 {
		t.Errorf("home entries: %+v", rep.Home)
	}
	var goBuild int64
	for _, c := range rep.Consumers {
		if c.ID == "go-build" {
			goBuild = c.Bytes
		}
	}
	if goBuild != 300 {
		t.Errorf("go-build: got %d, want 300", goBuild)
	}
	// 900 − (.cache 400 + work 100, both on /) and nothing for the consumers:
	// go-build is inside .cache, and .picode is inside home, so both are
	// already inside an entry of the breakdown.
	if rep.Other != 400 {
		t.Errorf("Other: got %d, want 400", rep.Other)
	}
	if len(r.calls) != 3 {
		t.Errorf("expected one df and two du calls, got %v", r.calls)
	}
}

// TestSizesOfTellsAMissingMachineFromAMissingPath: `du` exits non-zero the
// moment one path in a list is gone, which is the ordinary state of a machine
// that never installed two thirds of the table. Only a `du` that cannot run is
// an error — otherwise a fresh box would report a broken tool instead of an
// empty one.
func TestSizesOfTellsAMissingMachineFromAMissingPath(t *testing.T) {
	missing := &errRunner{err: &exec.ExitError{}}
	sizes, err := sizesOf(missing, []string{"/home/x/.npm"})
	if err != nil {
		t.Fatalf("a failed du that walked nothing is an empty report: %v", err)
	}
	if len(sizes) != 0 {
		t.Errorf("got %v", sizes)
	}

	broken := &errRunner{err: exec.ErrNotFound}
	if _, err := sizesOf(broken, []string{"/home/x/.npm"}); err == nil {
		t.Error("du that cannot be run at all must fail loudly")
	}
}

type errRunner struct{ err error }

func (e *errRunner) Output(string, ...string) ([]byte, error) { return nil, e.err }

func TestDataDirHonoursTheEnvironment(t *testing.T) {
	t.Setenv("PICODE_DATA", "/srv/picode")
	if got := DataDir("/home/x"); got != "/srv/picode" {
		t.Errorf("got %q", got)
	}
	t.Setenv("PICODE_DATA", "")
	if got := DataDir("/home/x"); got != "/home/x/.picode" {
		t.Errorf("got %q", got)
	}
}

// TestConsumersAreRunnableCommands guards the honesty of the table: a `How`
// that is not empty has to be a command someone can copy, and a data consumer
// must not pretend to have one.
func TestConsumersAreRunnableCommands(t *testing.T) {
	for _, c := range Consumers("/home/x") {
		if c.ID == "" || c.Title == "" || c.Note == "" || len(c.Paths) == 0 {
			t.Errorf("consumer %+v is missing a field a person reads", c)
		}
		switch c.Kind {
		case KindSafe, KindRedownload:
			if c.How == "" {
				t.Errorf("consumer %s is reclaimable but names no command", c.ID)
			}
		case KindData:
			if c.How != "" {
				t.Errorf("consumer %s is data and must not offer a one-liner: %q", c.ID, c.How)
			}
		default:
			t.Errorf("consumer %s has kind %q", c.ID, c.Kind)
		}
	}
}

// TestLocateConsumers is the decision table for where a cache is measured:
//
//	tool answer             | → path
//	absolute path           | the tool's answer (pnpm's store under ~/.local/share)
//	error / empty / relative| the table's default
//	rm-pruned or data row   | never asked; the default is what the prune removes
func TestLocateConsumers(t *testing.T) {
	home := "/home/u"
	answers := map[string]string{
		"pnpm store path":      "/home/u/.local/share/pnpm/store/v11\n",
		"go env GOCACHE":       "",
		"npm config get cache": "relative/npm",
		"uv cache dir":         "/home/u/.cache/uv",
		"go env GOMODCACHE":    "ERR",
	}
	asked := map[string]bool{}
	got := LocateConsumers(Consumers(home), func(argv []string) (string, error) {
		k := strings.Join(argv, " ")
		asked[k] = true
		if answers[k] == "ERR" {
			return "", errors.New("not installed")
		}
		return answers[k], nil
	}, home)
	paths := map[string]string{}
	for _, c := range got {
		paths[c.ID] = strings.Join(c.Paths, ",")
	}
	want := map[string]string{
		"pnpm":        "/home/u/.local/share/pnpm/store/v11",
		"go-build":    "/home/u/.cache/go-build",
		"npm":         "/home/u/.npm",
		"uv":          "/home/u/.cache/uv",
		"go-mod":      "/home/u/go/pkg/mod",
		"playwright":  "/home/u/.cache/ms-playwright",
		"picode-data": DataDir(home),
	}
	for id, w := range want {
		if paths[id] != w {
			t.Errorf("%s: %q, want %q", id, paths[id], w)
		}
	}
	if len(asked) != 5 {
		t.Errorf("asked %v; only the tool-owned caches may be asked", asked)
	}
	// The input table is not changed in place.
	if Consumers(home)[3].Paths[0] != "/home/u/.cache/pnpm" {
		t.Error("the default table was mutated")
	}
}

// TestLocateConsumersRefusesDangerousAnswers: an answer outside home, on the
// Windows drive, home itself, or nesting with another cache keeps the default.
func TestLocateConsumersRefusesDangerousAnswers(t *testing.T) {
	home := "/home/u"
	for _, bad := range []string{"/", "/mnt/c/Users/u/go/pkg/mod", "/opt/cache", "/home/u", "/home/u/.cache", "/home/u/.npm/sub"} {
		got := LocateConsumers(Consumers(home), func(argv []string) (string, error) {
			if argv[0] == "uv" {
				return bad, nil
			}
			return "", errors.New("no")
		}, home)
		for _, c := range got {
			if c.ID == "uv" && c.Paths[0] != "/home/u/.cache/uv" {
				t.Errorf("answer %q was used: %v", bad, c.Paths)
			}
		}
	}
}
