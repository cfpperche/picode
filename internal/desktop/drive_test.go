package desktop

import (
	"errors"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/provision"
)

// fakeRunner answers a canned reply per command, and records what it was asked
// to run so the argv itself can be asserted.
type fakeRunner struct {
	replies [][]byte
	errs    []error
	calls   [][]string
}

func (f *fakeRunner) Output(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	i := len(f.calls) - 1
	var out []byte
	if i < len(f.replies) {
		out = f.replies[i]
	}
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return out, err
}

func (f *fakeRunner) Run(name string, args ...string) error {
	_, err := f.Output(name, args...)
	return err
}

func reportJSON(root bool, steps string) []byte {
	scope := "goat"
	if root {
		scope = "root"
	}
	return []byte(`{"user":"` + scope + `","wsl":true,"root":` +
		map[bool]string{true: "true", false: "false"}[root] +
		`,"dryRun":false,"converged":false,"steps":[` + steps + `]}`)
}

func step(id, scope, action string) string {
	return `{"id":"` + id + `","title":"` + id + `","scope":"` + scope +
		`","before":{"status":"fix","detail":"d"},"action":"` + action + `"}`
}

func TestWSLArgs(t *testing.T) {
	got := WSLArgs("Ubuntu", "root", "picode", "provision")
	want := []string{"-d", "Ubuntu", "-u", "root", "--", "picode", "provision"}
	assertArgs(t, got, want)

	got = WSLArgs("Ubuntu", "", "/bin/sleep", "infinity")
	want = []string{"-d", "Ubuntu", "--", "/bin/sleep", "infinity"}
	assertArgs(t, got, want)
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("args = %v, want %v", got, want)
	}
}

// The two passes are the whole point of the split: root first, because
// enabling systemd and linger is what makes the user pass meaningful.
func TestProvisionRunsRootThenUser(t *testing.T) {
	r := &fakeRunner{replies: [][]byte{
		utf16le("/home/goat/.local/bin/picode\n"),
		reportJSON(true, step("linger", "root", "fixed")),
		reportJSON(false, step("service", "user", "fixed")),
	}}

	reports, err := Provision(r, "Ubuntu", "goat", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Fatalf("got %d reports, want 2", len(reports))
	}
	if len(r.calls) != 3 {
		t.Fatalf("got %d calls, want 3 (one lookup, two passes)", len(r.calls))
	}

	// Both passes must use the absolute path: root's PATH does not carry
	// ~/.local/bin, so the bare name resolves only for the owner.
	for i, c := range r.calls[1:] {
		if !strings.Contains(strings.Join(c, " "), "/home/goat/.local/bin/picode") {
			t.Errorf("pass %d does not use the resolved path: %v", i, c)
		}
	}

	first, second := strings.Join(r.calls[1], " "), strings.Join(r.calls[2], " ")
	if !strings.Contains(first, "-u root") {
		t.Errorf("first pass = %q, want it to run as root", first)
	}
	if !strings.Contains(second, "-u goat") {
		t.Errorf("second pass = %q, want it to run as goat", second)
	}
	// Both passes name the target account, or the root pass would enable
	// lingering for root instead of the owner.
	for i, c := range r.calls[1:] {
		if !strings.Contains(strings.Join(c, " "), "--user goat") {
			t.Errorf("pass %d does not target the owner: %v", i, c)
		}
	}
}

func TestProvisionPassesDryRunThrough(t *testing.T) {
	r := &fakeRunner{replies: [][]byte{
		utf16le("/home/goat/.local/bin/picode\n"),
		reportJSON(true, step("linger", "root", "planned")),
		reportJSON(false, step("service", "user", "planned")),
	}}
	if _, err := Provision(r, "Ubuntu", "goat", true); err != nil {
		t.Fatal(err)
	}
	for i, c := range r.calls[1:] {
		if !strings.Contains(strings.Join(c, " "), "--dry-run") {
			t.Errorf("pass %d lost --dry-run: %v", i, c)
		}
	}
}

// provision exits non-zero when it cannot converge but still prints a full
// report. Treating that exit status as fatal would throw away the reason.
func TestProvisionKeepsTheReportOnANonZeroExit(t *testing.T) {
	r := &fakeRunner{
		replies: [][]byte{
			utf16le("/usr/local/bin/picode\n"),
			reportJSON(true, step("linger", "root", "failed")),
			reportJSON(false, step("service", "user", "fixed")),
		},
		errs: []error{nil, errors.New("exit status 1"), nil},
	}
	reports, err := Provision(r, "Ubuntu", "goat", false)
	if err != nil {
		t.Fatalf("a report that parsed should not be an error: %v", err)
	}
	if len(reports) != 2 || reports[0].Steps[0].Action != provision.ActionFailed {
		t.Errorf("lost the failing step: %+v", reports)
	}
}

func TestProvisionReportsUnparseableOutput(t *testing.T) {
	r := &fakeRunner{
		replies: [][]byte{utf16le("/usr/local/bin/picode\n"), []byte("wsl: distribution not found")},
		errs:    []error{nil, errors.New("exit status 1")},
	}
	if _, err := Provision(r, "Nope", "goat", false); err == nil {
		t.Error("garbage output was accepted")
	}
}

// A shell profile that prints a banner must not break the contract.
func TestParseReportSkipsShellNoise(t *testing.T) {
	noisy := append([]byte("Welcome to Ubuntu\nsudo: unable to resolve host\n"),
		reportJSON(false, step("cert", "user", "none"))...)
	rep, err := parseReport(noisy)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Steps) != 1 || rep.Steps[0].ID != "cert" {
		t.Errorf("got %+v", rep.Steps)
	}
}

// Merging is where a wrong rule would lie to the user: the root pass skips
// every user step for lack of privilege, and the user pass skips every root
// step. Whichever pass actually resolved a step has to win.
func TestMergeKeepsTheResolvingPass(t *testing.T) {
	rootPass := provision.Report{Steps: []provision.Result{
		{ID: "linger", Action: provision.ActionFixed},
		{ID: "service", Action: provision.ActionSkipped, Error: "run as goat, not root"},
	}}
	userPass := provision.Report{Steps: []provision.Result{
		{ID: "linger", Action: provision.ActionSkipped, Error: "needs root"},
		{ID: "service", Action: provision.ActionFixed},
	}}

	merged := Merge([]provision.Report{rootPass, userPass})
	if len(merged) != 2 {
		t.Fatalf("got %d steps, want 2", len(merged))
	}
	for _, s := range merged {
		if s.Action != provision.ActionFixed {
			t.Errorf("step %q = %q, want fixed", s.ID, s.Action)
		}
	}
	if !Converged(merged) {
		t.Error("Converged = false when both steps were fixed")
	}
}

func TestMergeKeepsDependencyOrderAndReportsFailures(t *testing.T) {
	rootPass := provision.Report{Steps: []provision.Result{
		{ID: "wsl-conf", Action: provision.ActionNone},
		{ID: "linger", Action: provision.ActionFailed},
	}}
	userPass := provision.Report{Steps: []provision.Result{
		{ID: "wsl-conf", Action: provision.ActionSkipped},
		{ID: "linger", Action: provision.ActionSkipped},
		{ID: "health", Action: provision.ActionNone},
	}}

	merged := Merge([]provision.Report{rootPass, userPass})
	want := []string{"wsl-conf", "linger", "health"}
	for i, id := range want {
		if merged[i].ID != id {
			t.Errorf("step %d = %q, want %q", i, merged[i].ID, id)
		}
	}
	if merged[1].Action != provision.ActionFailed {
		t.Errorf("a failure was masked by a skip: %q", merged[1].Action)
	}
	if Converged(merged) {
		t.Error("Converged = true with a failed step")
	}
}

func TestConvergedIsFalseWhenNothingRan(t *testing.T) {
	if Converged(nil) {
		t.Error("an empty run must not count as converged")
	}
}

func TestCATrusted(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"0\r\n", false},
		{"1\r\n", true},
		{"2", true},
		{"", false},
		{"   \r\n", false},
	}
	for _, tt := range tests {
		if got := CATrusted(utf16le(tt.in)); got != tt.want {
			t.Errorf("CATrusted(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestServerURL(t *testing.T) {
	r := &fakeRunner{replies: [][]byte{
		utf16le(`{"url":"https://localhost:8445","port":8445}`),
	}}
	got, err := ServerURL(r, "Ubuntu", "goat")
	if err != nil || got != "https://localhost:8445" {
		t.Errorf("ServerURL = %q (%v), want https://localhost:8445", got, err)
	}

	bad := &fakeRunner{replies: [][]byte{utf16le("cat: no such file")}}
	if _, err := ServerURL(bad, "Ubuntu", "goat"); err == nil {
		t.Error("a missing server.json was accepted")
	}
}

// The tray must not run elevated: an administrator tray cannot talk to
// Explorer's notification area, and the browser it opens would inherit
// administrator rights.
func TestTaskCreateArgsRegisterAnUnelevatedLogonTask(t *testing.T) {
	got := strings.Join(TaskCreateArgs(`C:\PiCode\picode-desktop.exe`), " ")
	for _, want := range []string{"/create", "/tn " + TaskName, "/sc onlogon", "/rl limited", "/f"} {
		if !strings.Contains(got, want) {
			t.Errorf("args %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "/rl highest") {
		t.Error("the tray task must not be elevated")
	}
}

// The lookup has to run as the owner through a login shell. Running it as root,
// or without -l, finds nothing — which is exactly the bug that shipped: the
// root pass received a bare "picode" and died with "command not found".
func TestPicodePathUsesTheOwnersLoginShell(t *testing.T) {
	r := &fakeRunner{replies: [][]byte{utf16le("/home/goat/.local/bin/picode\n")}}

	got, err := PicodePath(r, "Ubuntu", "goat")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/home/goat/.local/bin/picode" {
		t.Errorf("PicodePath = %q", got)
	}

	call := strings.Join(r.calls[0], " ")
	if !strings.Contains(call, "-u goat") {
		t.Errorf("the lookup ran as the wrong account: %q", call)
	}
	if !strings.Contains(call, "sh -lc") {
		t.Errorf("the lookup used no login shell, so ~/.local/bin is not on PATH: %q", call)
	}
}

func TestPicodePathReportsAMissingBinaryPlainly(t *testing.T) {
	r := &fakeRunner{replies: [][]byte{utf16le("\n")}}
	if _, err := PicodePath(r, "Ubuntu", "goat"); err == nil {
		t.Fatal("an empty lookup was accepted")
	} else if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error = %q, want it to say picode is not installed", err)
	}
}
