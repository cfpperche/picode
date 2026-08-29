package provision

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// Scope says who can apply a step's fix. Checks are read-only and always run;
// only fixes are gated, so one dry run describes the whole machine even when
// it is invoked by an unprivileged user.
type Scope string

const (
	ScopeUser Scope = "user"
	ScopeRoot Scope = "root"
)

// Status is the verdict of a check.
type Status string

const (
	StatusOK      Status = "ok"      // nothing to do
	StatusFix     Status = "fix"     // this step can converge it
	StatusBlocked Status = "blocked" // needs something this run cannot do
)

// Action is what actually happened to a step.
type Action string

const (
	ActionNone    Action = "none"
	ActionPlanned Action = "planned"
	ActionFixed   Action = "fixed"
	ActionSkipped Action = "skipped"
	ActionFailed  Action = "failed"
)

// State is a check's verdict plus the sentence a human needs to trust it.
type State struct {
	Status Status `json:"status"`
	Detail string `json:"detail"`
}

func ok(format string, a ...any) State {
	return State{Status: StatusOK, Detail: fmt.Sprintf(format, a...)}
}

func needsFix(format string, a ...any) State {
	return State{Status: StatusFix, Detail: fmt.Sprintf(format, a...)}
}

func blocked(format string, a ...any) State {
	return State{Status: StatusBlocked, Detail: fmt.Sprintf(format, a...)}
}

// Env is everything the steps are allowed to know about the machine.
type Env struct {
	User    string // the account PiCode runs as
	Home    string // that account's home, where the unit and data live
	DataDir string
	Exe     string // the binary to install
	PathEnv string // PATH snapshot for the unit (ADR-0018)
	IsRoot  bool
	InWSL   bool
}

// Detect builds an Env for this process. target overrides the account when
// provisioning on someone's behalf, which is how the Windows side reaches the
// owner's account through `wsl -u root` (ADR-0020).
func Detect(target string) (Env, error) {
	env := Env{IsRoot: os.Geteuid() == 0, InWSL: InWSL()}

	u, err := user.Current()
	if err != nil {
		return env, fmt.Errorf("current user: %w", err)
	}
	env.User, env.Home = u.Username, u.HomeDir

	if target != "" && target != env.User {
		t, err := user.Lookup(target)
		if err != nil {
			return env, fmt.Errorf("user %q: %w", target, err)
		}
		env.User, env.Home = t.Username, t.HomeDir
	}

	env.DataDir = filepath.Join(env.Home, ".picode")
	if d := os.Getenv("PICODE_DATA"); d != "" && target == "" {
		env.DataDir = d
	}
	env.PathEnv = os.Getenv("PATH")
	if env.Exe, err = os.Executable(); err != nil {
		return env, fmt.Errorf("locate this binary: %w", err)
	}
	return env, nil
}

// Step is one converging unit of work: a read-only Check and the Fix that
// makes Check pass. Fix must be safe to run when Check already passes.
type Step struct {
	ID    string
	Title string
	Scope Scope
	Check func(Env) State
	Fix   func(Env) error
}

// Result is one step's outcome, shaped for both the terminal and --json.
type Result struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Scope  Scope  `json:"scope"`
	Before State  `json:"before"`
	Action Action `json:"action"`
	After  *State `json:"after,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Run checks every step in order and fixes what it is allowed to fix. It never
// stops early: a step that cannot run is reported and the rest still get their
// turn, so one pass always describes the whole machine.
func Run(env Env, steps []Step, dryRun bool) []Result {
	out := make([]Result, 0, len(steps))
	for _, s := range steps {
		r := Result{ID: s.ID, Title: s.Title, Scope: s.Scope, Before: s.Check(env)}

		switch {
		case r.Before.Status == StatusOK:
			r.Action = ActionNone
		// Blocked outranks dryRun: a step nothing here can fix is never part
		// of a plan, so reporting it as "would change" would be a lie.
		case r.Before.Status == StatusBlocked:
			r.Action = ActionSkipped
		case dryRun:
			r.Action = ActionPlanned
		case s.Scope == ScopeRoot && !env.IsRoot:
			r.Action = ActionSkipped
			r.Error = "needs root"
		case s.Scope == ScopeUser && env.IsRoot && env.User != "root":
			r.Action = ActionSkipped
			r.Error = "run as " + env.User + ", not root"
		default:
			if err := s.Fix(env); err != nil {
				r.Action, r.Error = ActionFailed, err.Error()
			} else {
				after := s.Check(env)
				r.After = &after
				r.Action = ActionFixed
				if after.Status != StatusOK {
					r.Action, r.Error = ActionFailed, "fix applied but the check still fails"
				}
			}
		}
		out = append(out, r)
	}
	return out
}

// Converged reports whether nothing is left to do.
func Converged(results []Result) bool {
	for _, r := range results {
		if r.Action != ActionNone && r.Action != ActionFixed {
			return false
		}
	}
	return true
}

// InWSL reports whether this is a WSL distro rather than plain Linux.
func InWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}

// run and output are the process boundary; tests replace them.
var run = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

var output = func(name string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &buf
	err := cmd.Run()
	return buf.String(), err
}

// writeBackup copies path to path+suffix before it is modified. A file that
// does not exist yet has nothing to preserve.
func writeBackup(path, suffix string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(path+suffix, b, 0o644)
}

// writeAtomic replaces path in one rename, so a crash mid-write cannot leave a
// half-written system file behind.
func writeAtomic(path, content string, perm os.FileMode) error {
	tmp := path + ".picode.tmp"
	if err := os.WriteFile(tmp, []byte(content), perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
