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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

type CLIDiagnostic = clilaunch.Diagnostic

// CLITerminals serializes terminal operations, not agent turns. Diagnostics
// are a cache of explicit checks and never launch a native conversation.
type CLITerminals struct {
	locks sync.Map
}

func newCLITerminals() *CLITerminals { return &CLITerminals{} }
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
}

func describeCLI(deps Deps, cli clilaunch.CLI) (cliView, error) {
	c, err := cliConfig(deps, cli.ID)
	if err != nil {
		return cliView{}, err
	}
	v := cliView{CLI: cli, Config: c, IntegrationApplied: cliIntegrationPrepared(deps.DataDir, cli)}
	v.Plan, _ = launchPlan(deps, cli, c, clilaunch.Overrides{}, filepath.Join(deps.DataDir, "cli-launch", "{terminal}", "run-{next}"))
	v.Executable, err = resolveCLIExecutable(cli, c)
	v.Installed = err == nil
	if err != nil {
		v.Problem = err.Error()
	}
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
		binary, err := resolveCLIExecutable(cli, c)
		d.Executable = binary
		d.Identity = executableIdentity(binary)
		if err == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, "--version")
			cmd.Env = cliEnvironment(c)
			var output boundedCLIOutput
			cmd.Stdout = &output
			cmd.Stderr = &output
			cmd.WaitDelay = time.Second
			if err = cmd.Run(); err == nil {
				d.Version = strings.TrimSpace(output.String())
				for _, value := range c.Env {
					if value != "" {
						d.Version = strings.ReplaceAll(d.Version, value, "••••")
					}
				}
				if len(d.Version) > 160 {
					d.Version = d.Version[:160]
				}
				if d.Version == "" {
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
			d.Prerequisites = err == nil
		}
		if err != nil {
			d.Error = err.Error()
		}
		if err := deps.Store.SetCLICheck(cli.ID, d); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, d)
	}
}

func handleCreateCLITerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, 404, "Unknown CLI.")
			return
		}
		var v struct {
			Name        string              `json:"name"`
			WorkspaceID string              `json:"workspaceId"`
			Cwd         string              `json:"cwd"`
			Overrides   clilaunch.Overrides `json:"overrides"`
		}
		if !readCLIJSON(w, r, &v) {
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, 503, "Install tmux to open a terminal.")
			return
		}
		c, err := cliConfig(deps, cli.ID)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		c = clilaunch.Resolve(c, v.Overrides)
		if err := clilaunch.Validate(c); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if _, err := resolveCLIExecutable(cli, c); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if strings.TrimSpace(v.Name) == "" {
			v.Name = cli.Name
		}
		unlockWorkspace := terminalLock(deps, "workspace:"+v.WorkspaceID)
		defer unlockWorkspace()
		t, err := deps.Store.CreateTerminalIn(v.WorkspaceID, v.Name, v.Cwd)
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		unlock := terminalLock(deps, t.ID)
		defer unlock()
		if err := deps.Store.SetTerminalLaunch(t.ID, cli.ID, v.Overrides); err != nil {
			_ = deps.Store.DeleteTerminal(t.ID)
			writeErr(w, 500, err.Error())
			return
		}
		name := tmux.ShellSessionName(t.ID)
		if err := ensureShell(deps, r, name, t.ID, t.Cwd); err != nil {
			// Keep the configured terminal available for repair and retry.
			writeJSON(w, 201, map[string]any{"id": t.ID, "launchError": err.Error()})
			return
		}
		writeJSON(w, 201, liveTermView(deps, r, t, name, true))
	}
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
			if err := deps.Tmux.KillSession(r.Context(), name); err != nil {
				writeErr(w, 500, err.Error())
				return
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
	if deps.Feed != nil {
		deps.Feed.Ephemeral("terminal.changed", liveTermView(deps, r, t, tmux.ShellSessionName(t.ID), live))
	}
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
	var body strings.Builder
	body.WriteString("#!/bin/sh\n# PiCode terminal launch. Values below are quoted arguments, never eval.\n")
	keys := []string{}
	for k := range c.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&body, "export %s=%s\n", k, shellQuote(c.Env[k]))
	}
	fmt.Fprintf(&body, "export PATH=%s\n", shellQuote(cliPath(c)))
	body.WriteString(shellQuote(command))
	for _, arg := range c.Args {
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
	env := append(append([]string{}, p.environment...), "PICODE_TERM_ID="+p.id, "PICODE_TERM_URL="+loopbackURL(deps))
	if b := interceptBinEnv(deps.DataDir); b != "" {
		env = append(env, b)
	}
	if err := deps.Tmux.NewSessionEnv(r.Context(), name, cwd, env, "/bin/sh", p.script); err != nil {
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
	c, err := cliConfig(deps, v.CLI)
	if err == nil && v.Applied != nil {
		effective := clilaunch.Resolve(c, v.Overrides)
		cli, _ := clilaunch.Find(v.CLI)
		binary, _ := resolveCLIExecutable(cli, effective)
		view["launchPending"] = v.Applied.CLI != v.CLI || v.Applied.Fingerprint != clilaunch.Fingerprint(effective) || binary != v.Applied.Executable || (v.Applied.Identity != "" && v.Applied.Identity != executableIdentity(binary))
	}
}
