package desktop

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cfpperche/picode/internal/provision"
)

// Runner is the process boundary. Everything that leaves this program goes
// through it, so the whole driver is testable without a Windows machine.
type Runner interface {
	Output(name string, args ...string) ([]byte, error)
	Run(name string, args ...string) error
}

// WSLExe is the launcher. It is resolved by PATH on Windows; from inside a
// distro (where interop leaves Windows off PATH) the caller passes the full
// /mnt/c path instead.
var WSLExe = "wsl.exe"

// WSLArgs builds `wsl.exe -d <distro> [-u <user>] -- <command...>`. An empty
// user means the distro's default account.
func WSLArgs(distro, user string, command ...string) []string {
	args := []string{"-d", distro}
	if user != "" {
		args = append(args, "-u", user)
	}
	args = append(args, "--")
	return append(args, command...)
}

// ListDistros asks WSL what is installed.
func ListDistros(r Runner) ([]Distro, error) {
	out, err := r.Output(WSLExe, "--list", "--verbose")
	if err != nil {
		// WSL reports "no installed distributions" through a non-zero exit,
		// so the output is worth more than the status.
		if text := strings.TrimSpace(DecodeWindows(out)); text != "" {
			return nil, fmt.Errorf("wsl --list: %s", firstLine(text))
		}
		return nil, fmt.Errorf("wsl --list: %w", err)
	}
	return ParseDistros(out)
}

// Provision drives `picode provision` inside the distro. It runs twice on
// purpose (ADR-0020): once as root for the steps that write outside the home
// directory, once as the owner for the unit, the certificate and the data
// dir — installing those as root would put PiCode in /root.
//
// The root pass goes first because enabling systemd and lingering is what
// makes the user pass meaningful.
func Provision(r Runner, distro, user string, dryRun bool) ([]provision.Report, error) {
	passes := []struct{ as string }{{"root"}, {user}}

	var reports []provision.Report
	for _, p := range passes {
		rep, err := provisionPass(r, distro, p.as, user, dryRun)
		if err != nil {
			return reports, err
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

func provisionPass(r Runner, distro, as, target string, dryRun bool) (provision.Report, error) {
	command := []string{"picode", "provision", "--json", "--user", target}
	if dryRun {
		command = append(command, "--dry-run")
	}

	out, err := r.Output(WSLExe, WSLArgs(distro, as, command...)...)
	// picode provision exits non-zero when it could not converge, and still
	// prints a full report — so the JSON is parsed before the exit status is
	// allowed to mean anything.
	rep, jsonErr := parseReport(out)
	if jsonErr != nil {
		if err != nil {
			return rep, fmt.Errorf("provision as %s: %w (%s)", as, err, firstLine(DecodeWindows(out)))
		}
		return rep, fmt.Errorf("provision as %s: %w", as, jsonErr)
	}
	return rep, nil
}

// parseReport reads the JSON envelope, tolerating anything the distro's shell
// printed before it (profile banners, sudo notices).
func parseReport(out []byte) (provision.Report, error) {
	text := DecodeWindows(out)
	start := strings.Index(text, "{")
	if start < 0 {
		return provision.Report{}, fmt.Errorf("no JSON in the provision output")
	}
	var rep provision.Report
	if err := json.Unmarshal([]byte(text[start:]), &rep); err != nil {
		return provision.Report{}, fmt.Errorf("provision JSON: %w", err)
	}
	return rep, nil
}

// Merge folds the two passes into one view of the machine. A step that any
// pass reports as done is done; the step order of the first pass is kept,
// because that is the dependency order internal/provision defines.
func Merge(reports []provision.Report) []provision.Result {
	best := map[string]provision.Result{}
	var order []string

	for _, rep := range reports {
		for _, step := range rep.Steps {
			prev, seen := best[step.ID]
			if !seen {
				order = append(order, step.ID)
				best[step.ID] = step
				continue
			}
			if rank(step.Action) > rank(prev.Action) {
				best[step.ID] = step
			}
		}
	}

	out := make([]provision.Result, 0, len(order))
	for _, id := range order {
		out = append(out, best[id])
	}
	return out
}

// rank orders outcomes by how settled they are, so merging two passes keeps
// the one that actually resolved the step. A pass that skipped a step for
// lack of privilege must never override the pass that fixed it.
func rank(a provision.Action) int {
	switch a {
	case provision.ActionNone:
		return 4
	case provision.ActionFixed:
		return 3
	case provision.ActionFailed:
		return 2
	case provision.ActionPlanned:
		return 1
	default: // skipped
		return 0
	}
}

// Converged reports whether the merged view leaves nothing to do.
func Converged(merged []provision.Result) bool {
	for _, s := range merged {
		if s.Action != provision.ActionNone && s.Action != provision.ActionFixed {
			return false
		}
	}
	return len(merged) > 0
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
