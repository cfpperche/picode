package provision

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// stub builds a one-step plan whose check returns want until it is fixed.
func stub(scope Scope, first State, fixErr error) (Step, *int) {
	calls := 0
	fixed := false
	return Step{
		ID:    "stub",
		Title: "stub",
		Scope: scope,
		Check: func(Env) State {
			if fixed {
				return ok("done")
			}
			return first
		},
		Fix: func(Env) error {
			calls++
			if fixErr != nil {
				return fixErr
			}
			fixed = true
			return nil
		},
	}, &calls
}

// The decision table for Run: conditions in, action out. Every row is a
// combination that changes the outcome, because a provisioner that fixes when
// it should have skipped is how a "safe" installer damages a live machine.
func TestRunDecisionTable(t *testing.T) {
	tests := []struct {
		name       string
		state      State
		scope      Scope
		dryRun     bool
		isRoot     bool
		user       string // the account being provisioned for
		acting     string // the account this run is actually under
		wantAction Action
		wantFixes  int
	}{
		{
			name: "satisfied step is left alone", state: ok("fine"), scope: ScopeUser,
			user: "goat", acting: "goat", wantAction: ActionNone, wantFixes: 0,
		},
		{
			name: "satisfied step is left alone in a dry run too", state: ok("fine"), scope: ScopeUser,
			dryRun: true, user: "goat", acting: "goat", wantAction: ActionNone, wantFixes: 0,
		},
		{
			name: "fixable step is fixed", state: needsFix("needs it"), scope: ScopeUser,
			user: "goat", acting: "goat", wantAction: ActionFixed, wantFixes: 1,
		},
		{
			name: "dry run plans instead of fixing", state: needsFix("needs it"), scope: ScopeUser,
			dryRun: true, user: "goat", acting: "goat", wantAction: ActionPlanned, wantFixes: 0,
		},
		{
			name: "blocked step is skipped, never fixed", state: blocked("cannot"), scope: ScopeUser,
			user: "goat", acting: "goat", wantAction: ActionSkipped, wantFixes: 0,
		},
		{
			// Blocked outranks dryRun: a plan must not promise what no run can do.
			name: "blocked step is skipped in a dry run, not planned", state: blocked("cannot"), scope: ScopeUser,
			dryRun: true, user: "goat", acting: "goat", wantAction: ActionSkipped, wantFixes: 0,
		},
		{
			name: "root step without root is skipped", state: needsFix("needs it"), scope: ScopeRoot,
			isRoot: false, user: "goat", acting: "goat", wantAction: ActionSkipped, wantFixes: 0,
		},
		{
			name: "root step with root is fixed", state: needsFix("needs it"), scope: ScopeRoot,
			isRoot: true, user: "goat", acting: "goat", wantAction: ActionFixed, wantFixes: 1,
		},
		{
			// `wsl -u root picode provision --user goat` must not install into
			// /root, so user-scope work waits for a run as that user.
			name: "user step as root on someone else's behalf is skipped", state: needsFix("needs it"), scope: ScopeUser,
			isRoot: true, user: "goat", acting: "root", wantAction: ActionSkipped, wantFixes: 0,
		},
		{
			name: "user step as root for root itself is fixed", state: needsFix("needs it"), scope: ScopeUser,
			isRoot: true, user: "root", acting: "root", wantAction: ActionFixed, wantFixes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, calls := stub(tt.scope, tt.state, nil)
			env := Env{User: tt.user, Acting: tt.acting, IsRoot: tt.isRoot}

			got := Run(env, []Step{step}, tt.dryRun)
			if len(got) != 1 {
				t.Fatalf("got %d results, want 1", len(got))
			}
			if got[0].Action != tt.wantAction {
				t.Errorf("action = %q, want %q (error: %q)", got[0].Action, tt.wantAction, got[0].Error)
			}
			if *calls != tt.wantFixes {
				t.Errorf("fix ran %d time(s), want %d", *calls, tt.wantFixes)
			}
		})
	}
}

func TestRunReportsAFixThatDidNotTake(t *testing.T) {
	step := Step{
		ID: "stub", Title: "stub", Scope: ScopeUser,
		Check: func(Env) State { return needsFix("still broken") },
		Fix:   func(Env) error { return nil }, // claims success, changes nothing
	}
	got := Run(Env{User: "goat", Acting: "goat"}, []Step{step}, false)
	if got[0].Action != ActionFailed {
		t.Errorf("action = %q, want %q", got[0].Action, ActionFailed)
	}
	if got[0].Error == "" {
		t.Error("a fix that did not take must say so")
	}
}

func TestRunReportsAFixError(t *testing.T) {
	step, _ := stub(ScopeUser, needsFix("needs it"), errors.New("boom"))
	got := Run(Env{User: "goat", Acting: "goat"}, []Step{step}, false)
	if got[0].Action != ActionFailed || got[0].Error != "boom" {
		t.Errorf("action = %q, error = %q; want failed / boom", got[0].Action, got[0].Error)
	}
}

// One failing step must not hide the rest: a single pass has to describe the
// whole machine, or the operator fixes one thing per run.
func TestRunDoesNotStopAtTheFirstProblem(t *testing.T) {
	bad, _ := stub(ScopeUser, needsFix("needs it"), errors.New("boom"))
	good, goodCalls := stub(ScopeUser, needsFix("needs it"), nil)

	got := Run(Env{User: "goat", Acting: "goat"}, []Step{bad, good}, false)
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2", len(got))
	}
	if got[1].Action != ActionFixed || *goodCalls != 1 {
		t.Errorf("second step did not run: action %q, %d fix call(s)", got[1].Action, *goodCalls)
	}
}

func TestConverged(t *testing.T) {
	tests := []struct {
		name    string
		actions []Action
		want    bool
	}{
		{"nothing to do", []Action{ActionNone, ActionNone}, true},
		{"everything fixed", []Action{ActionNone, ActionFixed}, true},
		{"a plan is not convergence", []Action{ActionNone, ActionPlanned}, false},
		{"a skip is not convergence", []Action{ActionNone, ActionSkipped}, false},
		{"a failure is not convergence", []Action{ActionFixed, ActionFailed}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := make([]Result, len(tt.actions))
			for i, a := range tt.actions {
				results[i] = Result{Action: a}
			}
			if got := Converged(results); got != tt.want {
				t.Errorf("Converged = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteBackupAndAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wsl.conf")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeBackup(path, BackupSuffix); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, "replaced\n", 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil || string(got) != "replaced\n" {
		t.Errorf("file = %q (%v), want %q", got, err, "replaced\n")
	}
	bak, err := os.ReadFile(path + BackupSuffix)
	if err != nil || string(bak) != "original\n" {
		t.Errorf("backup = %q (%v), want %q", bak, err, "original\n")
	}
	if _, err := os.Stat(path + ".picode.tmp"); !os.IsNotExist(err) {
		t.Error("the temp file outlived the write")
	}
}

// Backing up a file that does not exist yet is not an error — there is simply
// nothing to preserve.
func TestWriteBackupIgnoresAMissingFile(t *testing.T) {
	if err := writeBackup(filepath.Join(t.TempDir(), "absent"), BackupSuffix); err != nil {
		t.Errorf("writeBackup on a missing file: %v", err)
	}
}

// A user-scope step is skipped whenever this run is not the target account —
// root is the case that matters, but any mismatch has the same problem: the
// unit, the data dir and the certificate all belong to somebody else.
func TestOnBehalf(t *testing.T) {
	tests := []struct {
		name string
		env  Env
		want bool
	}{
		{"same account", Env{User: "goat", Acting: "goat"}, false},
		{"root provisioning for the owner", Env{User: "goat", Acting: "root"}, true},
		{"root provisioning for root", Env{User: "root", Acting: "root"}, false},
		{"unknown acting account is not a mismatch", Env{User: "goat"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.env.OnBehalf(); got != tt.want {
				t.Errorf("OnBehalf = %v, want %v", got, tt.want)
			}
		})
	}
}
