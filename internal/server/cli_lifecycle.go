// CLI lifecycle routes (ADR-0087): update checks and durable update,
// reinstall and uninstall jobs that run the vendors' own commands.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clijob"
	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clilifecycle"
	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
)

// lifecyclePlanView rides on cliView: what the surface may offer for this
// CLI's detected install method.
type lifecycleView struct {
	Method         string                     `json:"method"`
	CanInstall     bool                       `json:"canInstall"`
	CanUpdate      bool                       `json:"canUpdate"`
	CanFix         bool                       `json:"canReinstall"`
	CanCheckUpdate bool                       `json:"canCheckUpdate"`
	Uninstall      clilifecycle.UninstallKind `json:"uninstall"`
	Docs           string                     `json:"docs,omitempty"`
}

// describeLifecycle computes what the surface may offer. A missing CLI gets
// an install plan (npm-backed CLIs) or a guided-install docs link (ADR-0093).
func describeLifecycle(cliID string, installed bool, executable string) lifecycleView {
	if !installed {
		if plan, ok := clilifecycle.ForMissing(cliID); ok {
			return lifecycleView{Method: "npm", CanInstall: true, Uninstall: plan.Uninstall}
		}
		return lifecycleView{Method: string(clilifecycle.MethodUnknown), Docs: clilifecycle.InstallDocs(cliID)}
	}
	method := clilifecycle.MethodUnknown
	if executable != "" {
		method = clilifecycle.DetectMethod(executable)
	}
	plan, ok := clilifecycle.For(cliID, method)
	return lifecycleView{
		Method:         string(plan.Method),
		CanUpdate:      ok && plan.CanUpdate(),
		CanFix:         ok && plan.CanReinstall(),
		CanCheckUpdate: ok && plan.LatestFrom != "",
		Uninstall:      plan.Uninstall,
		Docs:           plan.Docs,
	}
}

// npmBinary resolves npm next to the CLI's own PATH so nvm-style installs
// find their matching npm.
func npmBinary(c clilaunch.Config) (string, error) {
	for _, dir := range filepath.SplitList(cliPath(c)) {
		candidate := filepath.Join(dir, "npm")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", errors.New("npm was not found on the CLI's PATH.")
}

// registerCLILifecycleRoutes adds the lifecycle endpoints to /api/clis.
func registerCLILifecycleRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/clis/{cli}/update-check", handleCLIUpdateCheck(deps))
	mux.HandleFunc("POST /api/clis/{cli}/lifecycle", handleCLILifecycle(deps))
	mux.HandleFunc("GET /api/cli-jobs", func(w http.ResponseWriter, r *http.Request) {
		if deps.CLIJobs == nil {
			writeErr(w, 503, "CLI lifecycle is unavailable.")
			return
		}
		jobs, err := deps.CLIJobs.List()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		// A package job carries the command it ran, so a refusal that only a
		// person in a terminal can answer comes with the line to run (ADR-0167).
		views := make([]cliJobView, 0, len(jobs))
		for _, j := range jobs {
			views = append(views, cliJobViewOf(j))
		}
		writeJSON(w, 200, map[string]any{"jobs": views})
	})
}

// resolveLifecycleExec turns a plan into the concrete command for the job
// runner. Under the npm method every action runs npm from the CLI's PATH;
// vendor methods run the CLI's resolved executable. Install always targets
// a missing CLI and always runs npm.
func resolveLifecycleExec(deps Deps, cli clilaunch.CLI, action clilifecycle.Action) (clijob.Exec, error) {
	unlock := terminalLock(deps, "cli-config")
	defer unlock()
	c, err := cliConfig(deps, cli.ID)
	if err != nil {
		return clijob.Exec{}, err
	}
	var plan clilifecycle.Plan
	var managed bool
	if action == clilifecycle.ActionInstall {
		if _, err := resolveCLIExecutable(cli, c); err == nil {
			return clijob.Exec{}, fmt.Errorf("%s is already installed; reinstall updates it in place.", cli.Name)
		}
		plan, managed = clilifecycle.ForMissing(cli.ID)
		if !managed {
			return clijob.Exec{}, fmt.Errorf("%s has no managed install; follow its documentation.", cli.Name)
		}
	} else {
		plan, managed = clilifecycle.For(cli.ID, detectMethod(deps, cli, c))
		if !managed {
			return clijob.Exec{}, fmt.Errorf("%s has no managed lifecycle for this install method. See its documentation.", cli.Name)
		}
	}
	args, err := plan.Args(action)
	if err != nil {
		return clijob.Exec{}, err
	}
	useNpm := action == clilifecycle.ActionInstall
	if !useNpm {
		switch action {
		case clilifecycle.ActionUpdate:
			useNpm = plan.UpdateViaNpm
		case clilifecycle.ActionReinstall:
			// npm reinstall args are npm-style; vendor reinstall args are not.
			useNpm = plan.Method == clilifecycle.MethodNpm
		case clilifecycle.ActionUninstall:
			useNpm = plan.Uninstall == clilifecycle.UninstallNpm
		}
	}
	if useNpm {
		bin, err := npmBinary(c)
		if err != nil {
			return clijob.Exec{}, err
		}
		return clijob.Exec{Exe: bin, Args: args, Env: cliEnvironment(c)}, nil
	}
	executable, err := resolveCLIExecutable(cli, c)
	if err != nil {
		return clijob.Exec{}, err
	}
	env := cliEnvironment(c)
	if (action == clilifecycle.ActionUpdate || action == clilifecycle.ActionReinstall) && len(plan.UpdateEnv) > 0 {
		// Overlay the vendor entrypoint (Muse's launcher flag): drop first
		// so the appended value is the one the child reads.
		for _, kv := range plan.UpdateEnv {
			if k, _, ok := strings.Cut(kv, "="); ok {
				env = dropEnv(env, k)
			}
		}
		env = append(env, plan.UpdateEnv...)
	}
	return clijob.Exec{Exe: executable, Args: args, Env: env}, nil
}

func detectMethod(deps Deps, cli clilaunch.CLI, c clilaunch.Config) clilifecycle.Method {
	executable, err := resolveCLIExecutable(cli, c)
	if err != nil {
		return clilifecycle.MethodUnknown
	}
	return clilifecycle.DetectMethod(executable)
}

// handleCLIUpdateCheck refreshes Latest/UpdateAvailable for one CLI and
// stores it with the diagnostic (event cli.checked keeps the UI current).
func handleCLIUpdateCheck(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, 404, "Unknown CLI.")
			return
		}
		unlock := terminalLock(deps, "cli-config")
		defer unlock()
		c, err := cliConfig(deps, cli.ID)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		d := clilaunch.Diagnostic{CheckedAt: time.Now().UTC().Format(time.RFC3339)}
		if old, err := deps.Store.CLICheck(cli.ID); err == nil && old != nil {
			d = *old
		}
		executable, execErr := resolveCLIExecutable(cli, c)
		if execErr != nil {
			writeErr(w, 409, execErr.Error())
			return
		}
		d.Executable = executable
		d.Identity = executableIdentity(executable)
		d.InstallMethod = string(clilifecycle.DetectMethod(executable))
		plan, managed := clilifecycle.For(cli.ID, clilifecycle.Method(d.InstallMethod))
		d.UpdateCheckedAt = time.Now().UTC().Format(time.RFC3339)
		d.UpdateError, d.Latest, d.UpdateAvailable, d.UpdateSource = "", "", false, ""
		if !managed {
			d.UpdateError = "No managed lifecycle for this install method."
		} else if d.Version == "" {
			d.UpdateError = "Run Check setup first so the installed version is known."
		} else if err := runUpdateCheck(r.Context(), cli.ID, plan, c, executable, &d); err != nil {
			d.UpdateError = err.Error()
		}
		if err := deps.Store.SetCLICheck(cli.ID, d); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, d)
	}
}

// cliNpmLatest is a var so tests can fake the registry.
var cliNpmLatest = pipkg.NpmLatest

// channelGet is a var so tests can fake a vendor version channel.
var channelGet = func(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "picode")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("muse channel returned HTTP %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 1<<20))
}

func channelNewer(latest, current string) bool {
	if latest == "" || current == "" || latest == current {
		return false
	}
	if pipkg.Newer(latest, current) {
		return true
	}
	return clilifecycle.ExtractSemver(latest) == clilifecycle.ExtractSemver(current)
}

// libcIsMusl answers the one platform question the Antigravity manifest asks:
// Alpine-style systems take the _musl artifact. The installer also probes
// `ldd`; the libc file covers the systems that ship it.
func libcIsMusl(goarch string) bool {
	arch := goarch
	if goarch == "arm64" {
		arch = "aarch64"
	}
	for _, p := range []string{"/lib/libc.musl-" + arch + ".so.1", "/lib/ld-musl-" + arch + ".so.1"} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func runUpdateCheck(ctx context.Context, cliID string, plan clilifecycle.Plan, c clilaunch.Config, executable string, d *clilaunch.Diagnostic) error {
	if plan.LatestFrom == "npm" {
		latest, err := cliNpmLatest(ctx, plan.NpmPackage)
		if err != nil {
			return fmt.Errorf("Could not reach the npm registry.")
		}
		d.UpdateSource = "npm"
		d.Latest = latest
		d.UpdateAvailable = pipkg.Newer(latest, clilifecycle.ExtractSemver(d.Version))
		return nil
	}
	if plan.LatestFrom == "channel" {
		checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		url, ok := clilifecycle.ChannelURLFor(cliID, runtime.GOOS, runtime.GOARCH, libcIsMusl(runtime.GOARCH))
		if !ok {
			return fmt.Errorf("No version channel is known for %s.", cliID)
		}
		raw, err := channelGet(checkCtx, url)
		if err != nil {
			name := cliID
			if c, ok := clilaunch.Find(cliID); ok {
				name = c.Name
			}
			return fmt.Errorf("Could not reach the %s version channel.", name)
		}
		ch, err := clilifecycle.ParseChannelVersion(raw)
		if err != nil {
			return err
		}
		d.UpdateSource = "channel"
		d.Latest = ch.Version
		switch cliID {
		case "muse":
			d.UpdateAvailable = channelNewer(ch.Version, clilifecycle.ExtractMuseVersion(d.Version))
		default:
			d.UpdateAvailable = channelNewer(ch.Version, clilifecycle.ExtractSemver(d.Version))
		}
		return nil
	}
	catalog, ok := clilaunch.Find(cliID)
	if !ok {
		return fmt.Errorf("Unknown CLI.")
	}
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(checkCtx, executable, plan.CheckArgs...)
	cmd.Env = cliEnvironment(c)
	var output boundedCLIOutput
	cmd.Stdout = &output
	cmd.Stderr = &output
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("The %s update check did not answer.", catalog.Name)
	}
	switch cliID {
	case "grok":
		c2, err := clilifecycle.ParseGrokCheck(output.Bytes())
		if err != nil {
			return err
		}
		d.UpdateSource = "vendor"
		d.Latest = c2.LatestVersion
		d.UpdateAvailable = c2.UpdateAvailable
		return nil
	case "omp":
		latest, err := clilifecycle.ParseOmpCheck(output.String())
		if err != nil {
			return err
		}
		d.UpdateSource = "vendor"
		d.Latest = latest
		d.UpdateAvailable = latest != "" && pipkg.Newer(latest, clilifecycle.ExtractSemver(d.Version))
		return nil
	case "hermes":
		available, err := clilifecycle.ParseHermesCheck(output.String())
		if err != nil {
			return err
		}
		d.UpdateSource = "vendor"
		d.Latest = ""
		d.UpdateAvailable = available
		return nil
	default:
		return fmt.Errorf("No vendor update check for %s.", cliID)
	}
}

// handleCLILifecycle accepts an update/reinstall/uninstall job request.
func handleCLILifecycle(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, 404, "Unknown CLI.")
			return
		}
		if !cli.Launchable() {
			writeErr(w, 400, fmt.Sprintf("Lifecycle actions are not available for %s yet.", cli.Name))
			return
		}
		if deps.CLIJobs == nil {
			writeErr(w, 503, "CLI lifecycle is unavailable.")
			return
		}
		var v struct {
			Action           string `json:"action"`
			RequestKey       string `json:"requestKey"`
			ConfirmTerminals bool   `json:"confirmTerminals"`
			ConfirmName      string `json:"confirmName"`
		}
		if !readCLIJSON(w, r, &v) {
			return
		}
		if v.Action != "install" && v.Action != "update" && v.Action != "reinstall" && v.Action != "uninstall" {
			writeErr(w, 400, "Unknown lifecycle action.")
			return
		}
		if v.Action == "uninstall" && strings.TrimSpace(v.ConfirmName) != cli.Name {
			writeErr(w, 400, fmt.Sprintf("Type %q to confirm uninstalling %s.", cli.Name, cli.Name))
			return
		}
		j, err := deps.CLIJobs.Start(cli.ID, v.Action, v.RequestKey, "", v.ConfirmTerminals)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrCLILifecycleConflict), errors.Is(err, clijob.ErrTerminalsRunning):
				writeErr(w, 409, err.Error())
			default:
				writeErr(w, 400, err.Error())
			}
			return
		}
		writeJSON(w, 202, j)
	}
}

// resolveLifecycleByID adapts string IDs from the job service wiring.
func resolveLifecycleByID(deps Deps, cliID, action, payload string) (clijob.Exec, error) {
	if isPackageAction(action) {
		return resolvePackageJob(cliID, action, payload)
	}
	cli, ok := clilaunch.Find(cliID)
	if !ok {
		return clijob.Exec{}, fmt.Errorf("Unknown CLI.")
	}
	return resolveLifecycleExec(deps, cli, clilifecycle.Action(action))
}

// liveTerminalsFor counts running terminals launched with one CLI
// (ADR-0062 runtimes are the authoritative presence source).
func liveTerminalsFor(deps Deps, cliID string) int {
	if deps.TermRuntimes == nil {
		return 0
	}
	n := 0
	for _, rt := range deps.TermRuntimes.Snapshot() {
		if rt.CLI == cliID {
			n++
		}
	}
	return n
}

// refreshCLICheckAfterJob re-runs the bounded setup check so version and
// identity reflect the new binary. Update facts reset — the state changed
// and the next update check recomputes them.
func refreshCLICheckAfterJob(deps Deps, cliID string) {
	cli, ok := clilaunch.Find(cliID)
	if !ok || deps.Store == nil {
		return
	}
	unlock := terminalLock(deps, "cli-config")
	defer unlock()
	c, err := cliConfig(deps, cli.ID)
	if err != nil {
		return
	}
	d := CLIDiagnostic{CheckedAt: time.Now().UTC().Format(time.RFC3339), Fingerprint: clilaunch.Fingerprint(c)}
	if _, err := deps.Store.CLICheck(cli.ID); err == nil {
		d.InstallMethod = string(detectMethod(deps, cli, c))
	}
	out, err := computeCLICheck(deps, cli, c, d, context.Background())
	_ = out
	_ = err
}
