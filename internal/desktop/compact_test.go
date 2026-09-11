package desktop

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

// compactRunner records argv the way fakeRunner does and lets each call fail
// independently, because the one property worth pinning about the compact flow
// is what happens when a step in the middle fails.
type compactRunner struct {
	replies [][]byte
	errs    map[int]error
	calls   [][]string
}

func (c *compactRunner) Output(name string, args ...string) ([]byte, error) {
	c.calls = append(c.calls, append([]string{name}, args...))
	i := len(c.calls) - 1
	var out []byte
	if i < len(c.replies) {
		out = c.replies[i]
	}
	return out, c.errs[i]
}

func (c *compactRunner) Run(name string, args ...string) error {
	_, err := c.Output(name, args...)
	return err
}

// The volume read and the sparse read happen twice: once for the facts before,
// once after. The before read says "not sparse, 100 bytes"; the after read says
// "sparse, 20 allocated" — the conversion worked and the numbers moved.
func compactRunnerAfter(before, after int64) *compactRunner {
	return &compactRunner{
		replies: [][]byte{
			[]byte(`{"len":` + strconv.FormatInt(before, 10) + `,"total":1000,"free":900}`),
			[]byte("not sparse\r\n"),
			[]byte(`{"len":` + strconv.FormatInt(after, 10) + `,"total":1000,"free":1000}`),
			[]byte("Allocated range[1]: Offset: 0x0 Length: 0x" + strconv.FormatInt(after, 16)),
		},
		errs: map[int]error{},
	}
}

func callsJoined(c *compactRunner) []string {
	out := []string{}
	for _, call := range c.calls {
		out = append(out, strings.Join(call, " "))
	}
	return out
}

func containsCall(list []string, needle string) bool {
	for _, s := range list {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func TestCompactRunsStopConvertStart(t *testing.T) {
	r := compactRunnerAfter(100, 20)
	res, err := Compact(r, "Ubuntu", "goat", CompactSparse, `C:\wsl\ext4.vhdx`, 100, nil)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	calls := callsJoined(r)
	if !containsCall(calls, "--terminate Ubuntu") {
		t.Errorf("the distro is never stopped: %v", calls)
	}
	if !containsCall(calls, "--manage Ubuntu --set-sparse true") {
		t.Errorf("the sparse conversion never ran: %v", calls)
	}
	if !containsCall(calls, "-d Ubuntu -u goat -- true") {
		t.Errorf("the distro is never started again: %v", calls)
	}
	if res.AfterBytes != 20 || res.Sparse != true {
		t.Errorf("result: %+v", res)
	}
	if got := res.Returned(); got != 80 {
		t.Errorf("Returned = %d, want 80", got)
	}
}

// TestCompactRestartsTheDistroEvenWhenItFails is the safety property of the
// whole flow: a conversion that fails half-way must still hand the machine
// back its distro, because the keepalive and the service both expect it and a
// stopped distro is worse than the one we started with.
func TestCompactRestartsTheDistroEvenWhenItFails(t *testing.T) {
	r := compactRunnerAfter(100, 100)
	r.errs[2] = errors.New("exit status 1") // the conversion itself fails
	_, err := Compact(r, "Ubuntu", "goat", CompactSparse, `C:\wsl\ext4.vhdx`, 100, nil)
	if err == nil {
		t.Fatal("a failed conversion must be reported")
	}
	if !containsCall(callsJoined(r), "-d Ubuntu -u goat -- true") {
		t.Errorf("the distro was left stopped after a failure: %v", callsJoined(r))
	}
}

func TestCompactOptimizeVHDUsesLiteralPath(t *testing.T) {
	r := compactRunnerAfter(100, 100)
	path := `C:\Users\cfpp\AppData\Local\wsl\{c2af16cd}\ext4.vhdx`
	if _, err := Compact(r, "Ubuntu", "goat", CompactOptimizeVHD, path, 100, nil); err != nil {
		t.Fatalf("Compact: %v", err)
	}
	found := false
	for _, call := range callsJoined(r) {
		if strings.Contains(call, "Optimize-VHD -LiteralPath '"+path+"' -Mode Full") {
			found = true
		}
	}
	if !found {
		t.Errorf("Optimize-VHD never ran with the file: %v", callsJoined(r))
	}
}

func TestCompactRefusesAnUnknownMethod(t *testing.T) {
	r := compactRunnerAfter(100, 100)
	if _, err := Compact(r, "Ubuntu", "goat", "shrink", `C:\x.vhdx`, 100, nil); err == nil {
		t.Error("an unknown method must be refused before anything runs")
	}
	if len(r.calls) != 0 {
		t.Errorf("nothing should have run: %v", callsJoined(r))
	}
}

func TestPlanCompactPrefersSparseWhenTheBuildHasIt(t *testing.T) {
	method, _ := PlanCompact(DiskFacts{CanSparse: true})
	if method != CompactSparse {
		t.Errorf("got %q, want sparse", method)
	}
	method, why := PlanCompact(DiskFacts{CanSparse: false})
	if method != CompactOptimizeVHD || !strings.Contains(why, "administrator") {
		t.Errorf("fallback: %q %q", method, why)
	}
}

func TestCompactArgs(t *testing.T) {
	if got := TerminateArgs("Ubuntu"); strings.Join(got, " ") != "--terminate Ubuntu" {
		t.Errorf("TerminateArgs: %v", got)
	}
	if got := SetSparseArgs("Ubuntu"); strings.Join(got, " ") != "--manage Ubuntu --set-sparse true" {
		t.Errorf("SetSparseArgs: %v", got)
	}
	if got := StartDistroArgs("Ubuntu", "goat"); strings.Join(got, " ") != "-d Ubuntu -u goat -- true" {
		t.Errorf("StartDistroArgs: %v", got)
	}
	bad := OptimizeVHDArgs(`C:\it's\ext4.vhdx`)
	if !strings.Contains(strings.Join(bad, " "), "refused") {
		t.Errorf("a path with a quote must be refused, not escaped: %v", bad)
	}
}
