package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

type CLIDiagnostic = clilaunch.Diagnostic

// CLITerminals serializes terminal operations, not agent turns. Diagnostics
// are a cache of explicit checks and never launch a native conversation.
type CLITerminals struct {
	locks sync.Map
}

// NewCLITerminals shares lifecycle locks between HTTP routes and background preparation.
func NewCLITerminals() *CLITerminals { return &CLITerminals{} }

func newCLITerminals() *CLITerminals { return NewCLITerminals() }
func terminalLock(deps Deps, id string) func() {
	if deps.CLIs == nil {
		return func() {}
	}
	v, _ := deps.CLIs.locks.LoadOrStore(id, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

func cliConfig(deps Deps, id string) (clilaunch.Config, error) {
	if deps.Store == nil {
		return clilaunch.Config{}, errors.New("CLI settings are unavailable.")
	}
	c, found, err := deps.Store.CLIConfig(id)
	if !found {
		c.Integration = interceptOn(deps.DataDir, id)
	}
	return c, err
}

func cliPath(c clilaunch.Config) string {
	parts := append([]string{}, c.Path...)
	parts = append(parts, filepath.SplitList(os.Getenv("PATH"))...)
	return strings.Join(parts, string(os.PathListSeparator))
}

func isCLIWrapper(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, 256))
	return bytes.Contains(b, []byte("# PiCode intercept"))
}

func resolveCLIExecutable(cli clilaunch.CLI, c clilaunch.Config) (string, error) {
	name := strings.TrimSpace(c.Executable)
	if name == "" {
		name = cli.Command
	}
	valid := func(p string) bool {
		st, err := os.Stat(p)
		return err == nil && !st.IsDir() && st.Mode()&0o111 != 0 && !isCLIWrapper(p)
	}
	if strings.ContainsRune(name, filepath.Separator) {
		if !filepath.IsAbs(name) {
			return "", errors.New("Choose an absolute executable path.")
		}
		if valid(name) {
			return filepath.Clean(name), nil
		}
	} else {
		for _, dir := range filepath.SplitList(cliPath(c)) {
			if !filepath.IsAbs(dir) {
				continue
			}
			p := filepath.Join(dir, name)
			if valid(p) {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("%s was not found. Check its executable or PATH.", cli.Name)
}

type cliView struct {
	clilaunch.CLI
	Config             clilaunch.Config `json:"config"`
	Installed          bool             `json:"installed"`
	Executable         string           `json:"executable,omitempty"`
	Problem            string           `json:"problem,omitempty"`
	IntegrationApplied bool             `json:"integrationApplied"`
	Diagnostic         *CLIDiagnostic   `json:"diagnostic,omitempty"`
	Plan               clilaunch.Plan   `json:"plan"`
	Lifecycle          lifecycleView    `json:"lifecycle"`
	// IntegrationCapable says the surface may offer activity, launch
	// settings and the setup panes; Launchable says New terminal exists.
	// They are separate: Muse Code and Antigravity open a terminal with no
	// adapter behind it (surface "terminal").
	IntegrationCapable bool `json:"integrationCapable"`
	Launchable         bool `json:"launchable"`
	// Sessions advertises what this CLI's session source can do (list,
	// read a transcript, receive a native session, start from a brief) so
	// the web derives handoff targets from the server (ADR-0088).
	Sessions clisession.Capabilities `json:"sessions"`
}

func describeCLI(deps Deps, cli clilaunch.CLI) (cliView, error) {
	c, err := cliConfig(deps, cli.ID)
	if err != nil {
		return cliView{}, err
	}
	v := cliView{CLI: cli, Config: c, IntegrationApplied: cliIntegrationPrepared(deps.DataDir, cli), Sessions: clisession.CapabilitiesOf(cli.ID), IntegrationCapable: cli.Integrable(), Launchable: cli.Launchable()}
	v.Sessions.Agent = cliAgentLanding(cli.ID)
	v.Plan, _ = launchPlan(deps, cli, c, clilaunch.Overrides{}, filepath.Join(deps.DataDir, "cli-launch", "{terminal}", "run-{next}"))
	v.Executable, err = resolveCLIExecutable(cli, c)
	v.Installed = err == nil
	if err != nil {
		v.Problem = err.Error()
	}
	v.Lifecycle = describeLifecycle(cli.ID, v.Installed, v.Executable)
	if v.Installed && c.Integration && !v.IntegrationApplied {
		v.Problem = "Activity reporting files need repair."
	}
	d, err := deps.Store.CLICheck(cli.ID)
	if err != nil {
		return v, err
	}
	if d != nil {
		d.Stale = d.Fingerprint != clilaunch.Fingerprint(c) || d.Identity != executableIdentity(v.Executable)
		v.Diagnostic = d
	}
	return v, nil
}

func syncCLIIntegration(deps Deps, id string, on bool) error {
	if on {
		return installIntercept(deps.DataDir, id)
	}
	// Keep launch artifacts for existing processes. Only remove the entry
	// point used by future manual invocations; immutable launches are separate.
	cli, ok := clilaunch.Find(id)
	if !ok {
		return errors.New("Unknown CLI.")
	}
	if err := os.Remove(wrapperPath(deps.DataDir, cli.Command)); err != nil && !os.IsNotExist(err) {
		return err
	}
	m := loadInterceptEnabled(deps.DataDir)
	delete(m, id)
	return saveInterceptEnabled(deps.DataDir, m)
}

func registerCLIRoutes(mux Registrar, deps Deps) {
	registerCLIProfileRoutes(mux, deps)
	registerCLISessionRoutes(mux, deps)
	registerCLILifecycleRoutes(mux, deps)
	mux.HandleFunc("GET /api/clis", func(w http.ResponseWriter, r *http.Request) {
		rows := []cliView{}
		for _, cli := range clilaunch.Catalog() {
			v, err := describeCLI(deps, cli)
			if err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			rows = append(rows, v)
		}
		writeJSON(w, 200, map[string]any{"clis": rows, "terminalAvailable": deps.Tmux != nil && deps.Tmux.Available()})
	})
	mux.HandleFunc("PUT /api/clis/{cli}", func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, 404, "Unknown CLI.")
			return
		}
		if !cli.Integrable() {
			writeErr(w, 400, fmt.Sprintf("Launch settings are not available for %s yet.", cli.Name))
			return
		}
		var c clilaunch.Config
		if !readCLIJSON(w, r, &c) {
			return
		}
		if err := clilaunch.Validate(c); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		unlock := terminalLock(deps, "cli-config")
		defer unlock()
		if err := deps.Store.SetCLIConfig(cli.ID, c); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		applyErr := syncCLIIntegration(deps, cli.ID, c.Integration)
		v, err := describeCLI(deps, cli)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if applyErr != nil {
			v.Problem = "Settings saved; integration could not be applied: " + applyErr.Error()
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("POST /api/clis/{cli}/check", handleCLICheck(deps))
	mux.HandleFunc("POST /api/clis/{cli}/preview", handleCLIPreview(deps))
	mux.HandleFunc("POST /api/clis/{cli}/repair", func(w http.ResponseWriter, r *http.Request) {
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
		if err := syncCLIIntegration(deps, cli.ID, c.Integration); err != nil {
			writeErr(w, 500, "Could not repair PiCode integration files.")
			return
		}
		if deps.Feed != nil {
			deps.Feed.Ephemeral("cli.repaired", map[string]string{"id": cli.ID})
		}
		v, err := describeCLI(deps, cli)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("POST /api/clis/{cli}/terminals", handleCreateCLITerminal(deps))
	mux.HandleFunc("GET /api/terminals/{id}/launch", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetTerminal(id); err != nil {
			writeStoreErr(w, err)
			return
		}
		v, err := deps.Store.TerminalLaunch(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, 200, v)
	})
	mux.HandleFunc("PUT /api/terminals/{id}/launch", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		unlock := terminalLock(deps, id)
		defer unlock()
		var v struct {
			CLI       string              `json:"cli"`
			Overrides clilaunch.Overrides `json:"overrides"`
		}
		if !readCLIJSON(w, r, &v) {
			return
		}
		if _, ok := clilaunch.Find(v.CLI); !ok {
			writeErr(w, 400, "Unknown CLI.")
			return
		}
		c, err := cliConfig(deps, v.CLI)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if err := clilaunch.Validate(clilaunch.Resolve(c, v.Overrides)); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if err := deps.Store.SetTerminalLaunch(id, v.CLI, v.Overrides); err != nil {
			writeStoreErr(w, err)
			return
		}
		invalidateTerminals(deps)
		writeJSON(w, 200, map[string]any{"saved": true})
	})
	for _, action := range []string{"start", "stop", "restart", "remove"} {
		mux.HandleFunc("POST /api/terminals/{id}/launch/"+action, func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("action", action)
			handleCLITerminalAction(deps)(w, r)
		})
	}
}

func readCLIJSON(w http.ResponseWriter, r *http.Request, into any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(into); err != nil {
		writeErr(w, 400, "Invalid launch settings: "+err.Error())
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeErr(w, 400, "Expected one JSON object.")
		return false
	}
	return true
}

type boundedCLIOutput struct{ bytes.Buffer }

func (b *boundedCLIOutput) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < 4096 {
		take := min(n, 4096-b.Len())
		_, _ = b.Buffer.Write(p[:take])
	}
	return n, nil
}

func cliEnvironment(c clilaunch.Config) []string {
	env := os.Environ()
	keys := []string{}
	for k := range c.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		env = append(env, k+"="+c.Env[k])
	}
	return append(env, "PATH="+cliPath(c))
}

// dropEnv removes every entry naming key, so a value appended afterwards is
// the one the child reads (getenv returns the first match).
func dropEnv(env []string, key string) []string {
	out := env[:0]
	for _, kv := range env {
		if k, _, ok := strings.Cut(kv, "="); ok && k == key {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func handleCLICheck(deps Deps) http.HandlerFunc {
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
		d := CLIDiagnostic{CheckedAt: time.Now().UTC().Format(time.RFC3339), Fingerprint: clilaunch.Fingerprint(c)}
		b, err := computeCLICheck(deps, cli, c, d, r.Context())
		if b.StoreErr != nil {
			writeErr(w, 500, b.StoreErr.Error())
			return
		}
		writeJSON(w, 200, b.Diagnostic)
	}
}

// checkOutcome carries the computed diagnostic and any store write error so
// the HTTP handler and the post-job refresh share one code path.
type checkOutcome struct {
	Diagnostic CLIDiagnostic
	StoreErr   error
}

// computeCLICheck runs the bounded --version probe and prerequisite checks,
// then persists the diagnostic (cli.checked event fires inside the store).
func computeCLICheck(deps Deps, cli clilaunch.CLI, c clilaunch.Config, d CLIDiagnostic, parent context.Context) (checkOutcome, error) {
	out := checkOutcome{Diagnostic: d}
	binary, err := resolveCLIExecutable(cli, c)
	out.Diagnostic.Executable = binary
	out.Diagnostic.Identity = executableIdentity(binary)
	if err == nil {
		ctx, cancel := context.WithTimeout(parent, 4*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "--version")
		cmd.Env = cliEnvironment(c)
		if cli.ID == "muse" {
			// The launcher background-updates unless the check is silent;
			// drop an inherited value so ours is the one the child sees.
			cmd.Env = append(dropEnv(cmd.Env, "MUSE_NO_AUTO_UPDATE"), "MUSE_NO_AUTO_UPDATE=1")
		}
		var output boundedCLIOutput
		cmd.Stdout = &output
		cmd.Stderr = &output
		cmd.WaitDelay = time.Second
		if err = cmd.Run(); err == nil {
			out.Diagnostic.Version = clipCLIVersion(output.String())
			for _, value := range c.Env {
				if value != "" {
					out.Diagnostic.Version = strings.ReplaceAll(out.Diagnostic.Version, value, "••••")
				}
			}
			if out.Diagnostic.Version == "" {
				err = errors.New("The executable returned no version.")
			}
		} else {
			err = errors.New("The executable did not answer --version. Check its path and installation.")
		}
	}
	if err == nil && c.Integration {
		for _, command := range []string{"curl", "python3"} {
			if _, e := resolveCLIExecutable(clilaunch.CLI{Name: command, Command: command}, clilaunch.Config{Path: c.Path}); e != nil {
				err = fmt.Errorf("%s is required for activity reporting.", command)
				break
			}
		}
		out.Diagnostic.Prerequisites = err == nil
	}
	if err != nil {
		out.Diagnostic.Error = err.Error()
	}
	if err := deps.Store.SetCLICheck(cli.ID, out.Diagnostic); err != nil {
		out.StoreErr = err
	}
	return out, nil
}

// clipCLIVersion keeps the setup line to one short sentence. Some CLIs
// (Hermes Agent) dump install paths and SDK versions on extra lines.
func clipCLIVersion(raw string) string {
	s := ""
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			s = line
			break
		}
	}
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}

// cliTerminalRequest is the body of POST /api/clis/{cli}/terminals, also
// composed by the handoff endpoint (ADR-0088).
type cliTerminalRequest struct {
	Name        string              `json:"name"`
	WorkspaceID string              `json:"workspaceId"`
	Cwd         string              `json:"cwd"`
	Overrides   clilaunch.Overrides `json:"overrides"`
}

func handleCreateCLITerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, 404, "Unknown CLI.")
			return
		}
		if !cli.Launchable() {
			writeErr(w, 400, fmt.Sprintf("Launch is not available for %s yet.", cli.Name))
			return
		}
		var v cliTerminalRequest
		if !readCLIJSON(w, r, &v) {
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, 503, "Install tmux to open a terminal.")
			return
		}
		_, view, status, err := createCLITerminal(deps, r, cli, v)
		if err != nil {
			writeErr(w, status, err.Error())
			return
		}
		writeJSON(w, status, view)
	}
}

// checkCLILaunch resolves the CLI's stored settings with overrides and
// verifies they validate and point at an installed executable — the
// pre-flight both terminal creation and a handoff run before creating
// anything. Returns the HTTP status for the failure.
func checkCLILaunch(deps Deps, cli clilaunch.CLI, overrides clilaunch.Overrides) (clilaunch.Config, int, error) {
	if !cli.Launchable() {
		return clilaunch.Config{}, 400, fmt.Errorf("Launch is not available for %s yet.", cli.Name)
	}
	c, err := cliConfig(deps, cli.ID)
	if err != nil {
		return c, 500, err
	}
	c = clilaunch.Resolve(c, overrides)
	if err := clilaunch.Validate(c); err != nil {
		return c, 400, err
	}
	if _, err := resolveCLIExecutable(cli, c); err != nil {
		return c, 400, err
	}
	return c, 0, nil
}

// createCLITerminal creates a terminal with a CLI launch and starts it. On
// a launch failure the terminal is kept for repair and the 201 view
// carries launchError, exactly as before the handoff endpoint shared it.
func createCLITerminal(deps Deps, r *http.Request, cli clilaunch.CLI, v cliTerminalRequest) (store.Terminal, map[string]any, int, error) {
	if _, status, err := checkCLILaunch(deps, cli, v.Overrides); err != nil {
		return store.Terminal{}, nil, status, err
	}
	if strings.TrimSpace(v.Name) == "" {
		v.Name = cli.Name
	}
	unlockWorkspace := terminalLock(deps, "workspace:"+v.WorkspaceID)
	defer unlockWorkspace()
	t, err := deps.Store.CreateTerminalIn(v.WorkspaceID, v.Name, v.Cwd)
	if err != nil {
		return store.Terminal{}, nil, 400, err
	}
	unlock := terminalLock(deps, t.ID)
	defer unlock()
	if err := deps.Store.SetTerminalLaunch(t.ID, cli.ID, v.Overrides); err != nil {
		_ = deps.Store.DeleteTerminal(t.ID)
		return store.Terminal{}, nil, 500, err
	}
	name := tmux.ShellSessionName(t.ID)
	if err := ensureShell(deps, r, name, t.ID, t.Cwd); err != nil {
		// Keep the configured terminal available for repair and retry.
		publishTerminalState(deps, r, t, false)
		return t, map[string]any{"id": t.ID, "launchError": err.Error()}, 201, nil
	}
	// The store announced terminal.created with the bare row (id, name, cwd,
	// workspace, createdAt); the identity a sidebar row, a tab and a face draw
	// — the CLI this terminal launches, its runtime and its launch state —
	// lives outside it. Announce the same live view every other terminal
	// response uses, or a terminal created while a page is open reads as a
	// plain shell until a reload.
	publishTerminalState(deps, r, t, true)
	return t, liveTermView(deps, r, t, name, true), 201, nil
}

func resolvedTerminalLaunch(deps Deps, v *store.TerminalLaunch) (clilaunch.CLI, clilaunch.Config, string, error) {
	cli, ok := clilaunch.Find(v.CLI)
	if !ok {
		return cli, clilaunch.Config{}, "", errors.New("Unknown CLI.")
	}
	c, err := cliConfig(deps, v.CLI)
	if err != nil {
		return cli, c, "", err
	}
	p, c := launchPlan(deps, cli, c, v.Overrides, filepath.Join(deps.DataDir, "cli-launch", v.TerminalID, "run-{next}"))
	if p.Problem != "" {
		return cli, c, "", errors.New(p.Problem)
	}
	return cli, c, p.Executable, nil
}

func handleCLITerminalAction(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, action := r.PathValue("id"), r.PathValue("action")
		if action != "start" && action != "stop" && action != "restart" && action != "remove" {
			writeErr(w, 404, "Unknown terminal action.")
			return
		}
		var v struct {
			Confirm bool `json:"confirm"`
			Resume  bool `json:"resume"`
		}
		if !readCLIJSON(w, r, &v) {
			return
		}
		unlock := terminalLock(deps, id)
		defer unlock()
		t, err := deps.Store.GetTerminal(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, 503, "Terminal control is unavailable.")
			return
		}
		name := tmux.ShellSessionName(id)
		live, err := deps.Tmux.HasSession(r.Context(), name)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if (live && action != "start" || action == "remove") && !v.Confirm {
			writeErr(w, 409, "Confirm the action on this terminal.")
			return
		}
		launch, err := deps.Store.TerminalLaunch(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if action == "start" && v.Resume {
			// One-click recovery of the pinned native conversation (ADR-0084).
			if launch == nil {
				writeErr(w, 400, "This terminal has no CLI launch to resume.")
				return
			}
			if live {
				writeErr(w, 409, "This terminal is already running.")
				return
			}
			ls := launch.LastSession
			if ls == nil || ls.SessionID == "" {
				writeErr(w, 400, "No previous session recorded for this terminal yet.")
				return
			}
			resumeLaunch := *launch
			args := terminalResumeArgs(ls)
			resumeLaunch.Overrides.Args = &args
			prepared, perr := prepareCLITerminal(deps, t.Cwd, &resumeLaunch)
			if perr != nil {
				recordCLILaunchAttempt(deps, id, perr)
				writeErr(w, 400, perr.Error())
				return
			}
			defer prepared.discard()
			if serr := prepared.start(deps, r, name, t.Cwd); serr != nil {
				recordCLILaunchAttempt(deps, id, serr)
				publishTerminalState(deps, r, t, false)
				writeErr(w, 400, serr.Error())
				return
			}
			publishTerminalState(deps, r, t, true)
			writeJSON(w, 200, liveTermView(deps, r, t, name, true))
			return
		}
		var prepared *preparedCLILaunch
		if action == "restart" {
			launch, err := deps.Store.TerminalLaunch(id)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			if launch != nil {
				prepared, err = prepareCLITerminal(deps, t.Cwd, launch)
				if err != nil {
					recordCLILaunchAttempt(deps, id, err)
					writeErr(w, 400, err.Error())
					return
				}
				defer prepared.discard()
			}
			if st, err := os.Stat(t.Cwd); err != nil || !st.IsDir() {
				writeErr(w, 400, "That folder no longer exists.")
				return
			}
		}
		if action != "start" {
			// Last chance to pin the native conversation before the pane goes
			// away (ADR-0084).
			if deps.TermRuntimes != nil && action != "remove" {
				if rt, ok := deps.TermRuntimes.Get(id); ok && rt.CLI != "" {
					pinTerminalLastSession(deps, id, rt)
				}
			}
			// Stop escalation (ADR-0085): the pane root ignores SIGHUP now, so
			// a plain kill-session would leave it running headless. Kill the
			// session first (closes the PTY), then SIGTERM the pane root while
			// we still know its pid. Best effort, never fatal.
			panePID, _ := deps.Tmux.PanePID(r.Context(), name)
			if err := deps.Tmux.KillSession(r.Context(), name); err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			if panePID > 0 {
				if p, perr := os.FindProcess(panePID); perr == nil {
					_ = p.Signal(syscall.SIGTERM)
				}
			}
			if deps.TermStates != nil {
				deps.TermStates.Drop(id)
			}
			if deps.TermRuntimes != nil {
				deps.TermRuntimes.Drop(id)
			}
			live = false
		}
		if action == "remove" {
			if err := cleanCLILaunches(deps.DataDir, id, ""); err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			if err := deps.Store.DeleteTerminal(id); err != nil {
				writeStoreErr(w, err)
				return
			}
			invalidateTerminals(deps)
			w.WriteHeader(204)
			return
		}
		if action == "start" || action == "restart" {
			if prepared != nil {
				if err := prepared.start(deps, r, name, t.Cwd); err != nil {
					recordCLILaunchAttempt(deps, id, err)
					publishTerminalState(deps, r, t, false)
					writeErr(w, 400, err.Error())
					return
				}
			}
			if err := ensureShell(deps, r, name, id, t.Cwd); err != nil {
				publishTerminalState(deps, r, t, false)
				writeErr(w, 400, err.Error())
				return
			}
			live = true
		}
		publishTerminalState(deps, r, t, live)
		writeJSON(w, 200, liveTermView(deps, r, t, name, live))
	}
}

func publishTerminalState(deps Deps, r *http.Request, t store.Terminal, live bool) {
	// Called after the launch action already mutated the pane (started,
	// stopped, restarted): drop the list snapshot so the next GET recomputes.
	invalidateTerminals(deps)
	if deps.Feed != nil {
		deps.Feed.Ephemeral("terminal.changed", liveTermView(deps, r, t, tmux.ShellSessionName(t.ID), live))
	}
	// Pin the native session while the CLI reports in (ADR-0084): a deploy
	// or crash never announces itself, so the pin has to be current BEFORE
	// the terminal dies, not reconstructed after.
	if live && deps.Store != nil && deps.TermRuntimes != nil {
		if rt, ok := deps.TermRuntimes.Get(t.ID); ok && rt.Source == "wrapper" {
			pinTerminalLastSession(deps, t.ID, rt)
		}
	}
}

// pinTerminalLastSession resolves the native CLI session this terminal is
// running and pins it on the launch record (ADR-0084). Sessions last
// written before the run's start never win, so one terminal cannot steal
// another's conversation in a shared folder. Best effort by design: a
// failed pin only costs the resume shortcut, never the terminal.
func pinTerminalLastSession(deps Deps, termID string, runtime TermRuntime) {
	if deps.Store == nil || runtime.CLI == "" || runtime.SessionID != "" || runtime.CLI == "grok" || runtime.CLI == "hermes" {
		return
	}
	// An enrolled mailbox must never change address through a cwd heuristic,
	// including after restart when the ephemeral native report is absent.
	peers, err := deps.Store.ListPeerConnections()
	if err != nil {
		return
	}
	for _, peer := range peers {
		if peer.Kind == "terminal" && peer.OwnerID == termID && peer.Active {
			return
		}
	}
	t, err := deps.Store.GetTerminal(termID)
	if err != nil {
		return
	}
	s, err := clisession.Latest(runtime.CLI, t.Cwd, runtime.StartedAt)
	if err != nil || s == nil {
		return
	}
	_ = deps.Store.SetTerminalLastSession(termID, store.TerminalLastSession{
		CLI:        s.CLI,
		SessionID:  s.ID,
		Path:       s.Path,
		Cwd:        s.Cwd,
		Name:       s.Name,
		UpdatedAt:  s.UpdatedAt,
		Preview:    s.Preview,
		ResumeArgs: s.ResumeArgs,
	})
}

func shellQuote(v string) string { return "'" + strings.ReplaceAll(v, "'", "'\"'\"'") + "'" }

// Only PiCode's private, per-terminal launch artifacts are disposable. Native
// CLI sessions, credentials and packages are never inside this directory.
func cleanCLILaunches(dataDir, id, keep string) error {
	if dataDir == "" || id == "" || id == "." || id == ".." || filepath.Base(id) != id {
		return errors.New("Invalid terminal launch directory.")
	}
	root := filepath.Join(dataDir, "cli-launch", id)
	if keep == "" {
		return os.RemoveAll(root)
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "run-") && e.Name() != keep {
			if err := os.RemoveAll(filepath.Join(root, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func launchCLITerminal(deps Deps, r *http.Request, name, cwd string, v *store.TerminalLaunch) error {
	p, err := prepareCLITerminal(deps, cwd, v)
	if err == nil {
		defer p.discard()
		err = p.start(deps, r, name, cwd)
	}
	if err != nil {
		recordCLILaunchAttempt(deps, v.TerminalID, err)
	}
	return err
}

func recordCLILaunchAttempt(deps Deps, id string, err error) {
	a := clilaunch.Attempt{At: time.Now().UTC().Format(time.RFC3339)}
	if err != nil {
		// Persist a fixed diagnostic, never a native process's output or argv.
		a.Error = "Launch failed. Check the executable, folder and PiCode integration files."
	}
	_ = deps.Store.SetTerminalLaunchAttempt(id, a)
}

type preparedCLILaunch struct {
	dir, script, id string
	environment     []string
	snapshot        clilaunch.Snapshot
	started         bool
}

func (p *preparedCLILaunch) discard() {
	if !p.started {
		_ = os.RemoveAll(p.dir)
	}
}

func prepareCLITerminal(deps Deps, cwd string, v *store.TerminalLaunch) (*preparedCLILaunch, error) {
	if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
		return nil, errors.New("That folder no longer exists.")
	}
	cli, c, binary, err := resolvedTerminalLaunch(deps, v)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(deps.DataDir, "cli-launch", v.TerminalID)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(root, "run-")
	if err != nil {
		return nil, err
	}
	started := false
	defer func() {
		if !started {
			_ = os.RemoveAll(dir)
		}
	}()
	command := binary
	if c.Integration {
		hook, err := ensureHookScript(deps.DataDir)
		if err != nil {
			return nil, err
		}
		switch cli.ID {
		case "pi":
			err = writePiIntercept(dir, hook)
		case "claude-code":
			err = writeClaudeIntercept(dir, hook)
		case "codex":
			err = writeCodexIntercept(dir, hook)
		case "grok":
			err = writeGrokIntercept(dir, hook)
		case "hermes":
			err = writeHermesIntercept(dir, hook)
		case "opencode":
			err = writeOpencodeIntercept(dir, hook)
		}
		if err != nil {
			return nil, err
		}
		command = wrapperPath(dir, cli.Command)
		raw, err := os.ReadFile(command)
		if err != nil {
			return nil, err
		}
		body := strings.Replace(string(raw), wrapperFindReal, "real="+shellQuote(binary)+"\n", 1)
		if err := writeExecutable(command, body); err != nil {
			return nil, err
		}
	}
	peerOptions := communication.LaunchOptions{Env: map[string]string{}}
	peer, err := communication.LoadLaunch(deps.Store, deps.DataDir, "terminal", v.TerminalID)
	if err != nil {
		return nil, err
	}
	// Only the exact native resume recipe receives this conversation's token.
	// Ordinary fresh starts, forks, custom globals/`--` and manual commands receive no credential.
	if peer != nil && peerResumeExact(c.Args, v.LastSession) {
		peerOptions, err = communication.Options(deps.DataDir, *peer)
		if err != nil {
			return nil, err
		}
		if cli.ID == "opencode" {
			peerOptions.Env["PICODE_OPENCODE_SESSION_ID"] = v.LastSession.SessionID
			existing, specified := c.Env["OPENCODE_CONFIG_CONTENT"]
			if !specified {
				existing = os.Getenv("OPENCODE_CONFIG_CONTENT")
			}
			merged, err := communication.MergeOpenCode(existing, peerOptions.Env["OPENCODE_CONFIG_CONTENT"])
			if err != nil {
				return nil, err
			}
			peerOptions.Env["OPENCODE_CONFIG_CONTENT"] = merged
		}
	}
	if c.Integration && cli.ID == "pi" && len(peerOptions.Args) == 0 {
		if p, e := deps.Store.PeerParticipant("terminal", v.TerminalID); e == nil && p.Enabled && p.CLI == cli.ID {
			peerOptions = communication.PiAdapterOptions()
		}
	}
	// Native tool invocations resolve their own current session. This routing
	// root and executable contain no bearer, even before enrollment or resume.
	if c.Integration && communication.NativeMessages(cli.ID) {
		peerOptions, err = communication.NativeOptions(deps.DataDir, cli.ID)
		if err != nil {
			return nil, err
		}
		commands := filepath.Join(dir, "message-bin")
		if err := writeExecutable(filepath.Join(commands, "picode"), "#!/bin/sh\nexec "+shellQuote(peerOptions.Env["PICODE_MESSAGES_BIN"])+" \"$@\"\n"); err != nil {
			return nil, err
		}
		peerOptions.Env["PATH"] = commands + string(os.PathListSeparator) + cliPath(c)
	}

	var body strings.Builder
	body.WriteString("#!/bin/sh\n# PiCode terminal launch. Values below are quoted arguments, never eval.\n")
	// SIGHUP-immune pane root (ADR-0085): interactive bash ignores SIGHUP and
	// survived every daemon restart, while these `sh` roots died with the
	// pane's whole session. Children inherit the ignore, which is the point;
	// explicit Stop escalates to SIGTERM on the pane root, so "stop means
	// stop" survives the trap.
	body.WriteString("trap '' HUP\n")
	// OpenCode (and other TUIs) paint nothing until ready; a one-line
	// banner keeps the pane from looking stuck. The TUI alt-screen
	// replaces it as soon as the CLI draws.
	fmt.Fprintf(&body, "printf '%%s\\n' %s\n", shellQuote("Starting "+cli.Name+"..."))
	keys := []string{}
	for k := range c.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&body, "export %s=%s\n", k, shellQuote(c.Env[k]))
	}
	fmt.Fprintf(&body, "export PATH=%s\n", shellQuote(cliPath(c)))
	if len(peerOptions.Env) > 0 {
		body.WriteString("env ")
		peerKeys := []string{}
		for k := range peerOptions.Env {
			peerKeys = append(peerKeys, k)
		}
		sort.Strings(peerKeys)
		for _, k := range peerKeys {
			body.WriteString(shellQuote(k+"="+peerOptions.Env[k]) + " ")
		}
	}
	body.WriteString(shellQuote(command))
	launchArgs := append(append([]string{}, c.Args...), peerOptions.Args...)
	if cli.ID == "grok" {
		launchArgs = append(append([]string{}, peerOptions.Args...), c.Args...)
	}
	for _, arg := range launchArgs {
		body.WriteByte(' ')
		body.WriteString(shellQuote(arg))
	}
	body.WriteByte('\n')
	body.WriteString("picode_exit=$?\nprintf '\\nProcess exited (%s).\\n' \"$picode_exit\"\n")
	// Returning to the normal shell also restores the manual CLI wrappers.
	rc, err := ensureInterceptBashrc(deps.DataDir)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(&body, "exec %s", shellQuote(defaultShell()))
	if base := filepath.Base(defaultShell()); base == "bash" {
		fmt.Fprintf(&body, " --rcfile %s", shellQuote(rc))
	}
	body.WriteByte('\n')
	script := filepath.Join(dir, "launch.sh")
	if err := os.WriteFile(script, []byte(body.String()), 0o700); err != nil {
		return nil, err
	}
	snapshot := clilaunch.Describe(c, binary, "")
	snapshot.CLI = cli.ID
	snapshot.Identity = executableIdentity(binary)
	if c.Integration {
		plan, _ := launchPlan(deps, cli, c, clilaunch.Overrides{}, dir)
		snapshot.Injection = &plan.Injection
	}
	started = true // ownership transfers to the prepared object
	return &preparedCLILaunch{dir: dir, script: script, id: v.TerminalID, snapshot: snapshot, environment: cliEnvironment(c)}, nil
}

func (p *preparedCLILaunch) start(deps Deps, r *http.Request, name, cwd string) error {
	return p.startSized(deps, r, name, cwd, 0, 0)
}

func (p *preparedCLILaunch) startSized(deps Deps, r *http.Request, name, cwd string, width, height int) error {
	if blocked, e := peerStopPending(deps, p.id); e != nil || blocked {
		return errors.New("The previous process has not finished closing. Try again after it exits.")
	}
	env := append(append([]string{}, p.environment...), tmux.MarkerTermEnv+"="+p.id, tmux.MarkerURLEnv+"="+loopbackURL(deps))
	if b := interceptBinEnv(deps.DataDir); b != "" {
		env = append(env, b)
	}
	if err := deps.Tmux.NewSessionEnvSize(r.Context(), name, cwd, width, height, env, "/bin/sh", p.script); err != nil {
		return err
	}
	p.snapshot.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := deps.Store.SetTerminalLaunchApplied(p.id, p.snapshot); err != nil {
		_ = deps.Tmux.KillSession(r.Context(), name)
		return err
	}
	p.started = true
	// There was no live session on entry. Keep only this immutable generation.
	_ = cleanCLILaunches(deps.DataDir, p.id, filepath.Base(p.dir))
	return nil
}

func applyTerminalLaunch(deps Deps, view map[string]any, id string) {
	if deps.Store == nil {
		return
	}
	v, err := deps.Store.TerminalLaunch(id)
	if err != nil || v == nil {
		return
	}
	view["launchCli"] = v.CLI
	view["launchApplied"] = v.Applied
	view["launchAttempt"] = v.Attempt
	if v.LastSession != nil {
		view["lastSession"] = v.LastSession
	}
	c, err := cliConfig(deps, v.CLI)
	if err == nil && v.Applied != nil {
		effective := clilaunch.Resolve(c, v.Overrides)
		cli, _ := clilaunch.Find(v.CLI)
		binary, _ := resolveCLIExecutable(cli, effective)
		view["launchPending"] = v.Applied.CLI != v.CLI || v.Applied.Fingerprint != clilaunch.Fingerprint(effective) || binary != v.Applied.Executable || (v.Applied.Identity != "" && v.Applied.Identity != executableIdentity(binary))
	}
}

// Pi's catalog uses its own session UI, but a terminal resume needs its file.
func terminalResumeArgs(ls *store.TerminalLastSession) []string {
	if ls.CLI == "pi" && ls.Path != "" {
		return []string{"--session", ls.Path}
	}
	return append([]string{}, ls.ResumeArgs...)
}

// peerResumeExact is the only gate that attaches a conversation credential.
// Leading globals, `--`, extra operands and a different subcommand are a
// different launch, not the enrolled conversation.
func peerResumeExact(args []string, ls *store.TerminalLastSession) bool {
	if ls == nil {
		return false
	}
	recipe := terminalResumeArgs(ls)
	return len(recipe) > 0 && slices.Equal(args, recipe)
}
