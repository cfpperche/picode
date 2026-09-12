package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/hostfs"
)

// pipe gives a refusal test a private stdout without touching os/exec.
func pipe() (*os.File, *os.File) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	return r, w
}

func readLines(t *testing.T, r io.Reader) []string {
	t.Helper()
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, r); err != nil {
		t.Fatalf("read: %v", err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// fakeRunner records the How commands a clean run issued and reports fixed
// before/after sizes, so the flow is tested without du or rm touching the
// machine.
type fakeRunner struct {
	applied []string
	err     error
}

func (f *fakeRunner) apply(_ string, chosen []hostfs.Consumer, asJSON bool) []cleanResult {
	results := make([]cleanResult, 0, len(chosen))
	for _, c := range chosen {
		f.applied = append(f.applied, c.ID)
		if asJSON {
			// The real runner announces each step; the wire contract (one
			// object per line, outcome last) is what the shell parses.
			fmt.Println(progressLine("cleaning " + c.Title + "..."))
		}
		res := cleanResult{ID: c.ID, Before: 100, After: 40, Freed: 60}
		if f.err != nil {
			res.Error = f.err.Error()
			res.Freed = 0
		}
		results = append(results, res)
	}
	_ = asJSON
	return results
}

// refused is the JSON shape every refusal shares: exit zero, the reason in
// the field.
func refused(t *testing.T, fn func() error) map[string]string {
	t.Helper()
	r, w := pipe()
	old := os.Stdout
	os.Stdout = w
	err := fn()
	os.Stdout = old
	w.Close()
	if err != nil {
		t.Fatalf("refusals exit zero in json mode, got error: %v", err)
	}
	var v map[string]string
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatalf("refusal is not JSON: %v", err)
	}
	if _, ok := v["refused"]; !ok {
		t.Fatalf("no refused field: %v", v)
	}
	return v
}

func TestCleanApplyRefusalMatrix(t *testing.T) {
	home := t.TempDir()
	fake := &fakeRunner{}

	if got := refused(t, func() error {
		return cleanApply(home, []string{"go-build"}, false, true, fake)
	})["refused"]; !strings.Contains(got, "--yes") {
		t.Errorf("no --yes: got %q", got)
	}

	if got := refused(t, func() error {
		return cleanApply(home, nil, true, true, fake)
	})["refused"]; !strings.Contains(got, "no cache ids") {
		t.Errorf("empty ids: got %q", got)
	}

	if got := refused(t, func() error {
		return cleanApply(home, []string{"not-a-cache"}, true, true, fake)
	})["refused"]; !strings.Contains(got, "unknown cache id") {
		t.Errorf("unknown id: got %q", got)
	}

	// The data rows are refused EVEN when asked for by exact name — the
	// cleanup of Pi sessions and the database is never a delete.
	for _, id := range []string{"picode-data", "pi-sessions"} {
		got := refused(t, func() error {
			return cleanApply(home, []string{id}, true, true, fake)
		})["refused"]
		if !strings.Contains(got, "not a cache") {
			t.Errorf("%s: got %q", id, got)
		}
	}

	if len(fake.applied) != 0 {
		t.Errorf("refused runs must apply nothing, applied %v", fake.applied)
	}
}

func TestCleanApplyHappyPathAndProgress(t *testing.T) {
	home := t.TempDir()
	fake := &fakeRunner{}

	r, w := pipe()
	old := os.Stdout
	os.Stdout = w
	err := cleanApply(home, []string{"go-build", "npm"}, true, true, fake)
	os.Stdout = old
	w.Close()
	if err != nil {
		t.Fatalf("cleanApply: %v", err)
	}

	lines := readLines(t, r)
	if len(lines) != 3 {
		t.Fatalf("want 2 progress lines + 1 outcome, got %d: %v", len(lines), lines)
	}
	var outcome struct {
		Results    []cleanResult `json:"results"`
		TotalFreed int64         `json:"totalFreed"`
	}
	if err := json.Unmarshal([]byte(lines[2]), &outcome); err != nil {
		t.Fatalf("outcome line: %v", err)
	}
	if len(outcome.Results) != 2 || outcome.TotalFreed != 120 {
		t.Fatalf("outcome: %+v", outcome)
	}
	if outcome.Results[0].Freed != 60 || outcome.Results[1].Freed != 60 {
		t.Fatalf("freed per cache: %+v", outcome.Results)
	}
	if strings.Join(fake.applied, ",") != "go-build,npm" {
		t.Errorf("applied %v", fake.applied)
	}
}

func TestRunHowGuardsRmPaths(t *testing.T) {
	c := hostfs.Consumer{
		ID: "x", Title: "X", Kind: hostfs.KindRedownload,
		Paths: []string{"/home/u/.cache/x"},
		How:   "rm -rf /home/u/.cache/x",
	}
	if !strings.HasPrefix(c.How, "rm -rf ") {
		t.Fatal("guard test wants the rm form")
	}
	// A row whose rm path drifts from its own paths must refuse, never delete.
	drifted := c
	drifted.How = "rm -rf /home/u/.cache/other"
	if err := runHow(drifted); err == nil {
		t.Fatal("drifted rm path must error")
	}
}

func TestSplitIDs(t *testing.T) {
	got := splitIDs(" go-build ,,npm,")
	want := "go-build,npm"
	if strings.Join(got, ",") != want {
		t.Errorf("got %q want %q", got, want)
	}
}
