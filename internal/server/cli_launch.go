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

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/credentials"
	"github.com/cfpperche/picode/internal/pimission"
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

// interceptCLIPath is the PATH a launched CLI pane runs with: the intercept
// bin dir first when it holds wrappers, then the CLI's own path. CLI panes
// are /bin/sh launch scripts, not interactive shells — the rcfile prepend
// never runs for them — so without this a CLI's opener lookup resolves to
// the system binary and /login opens a browser inside WSL (measured
// 2026-09-22: omp login reached WSL chromium after ADR-0180's BROWSER
// landed, because omp resolves wslview/xdg-open by PATH, not $BROWSER).
func interceptCLIPath(dataDir string, c clilaunch.Config) string {
	bin := interceptBinDir(dataDir)
	if ents, err := os.ReadDir(bin); err != nil || len(ents) == 0 {
		return cliPath(c)
	}
	return bin + string(os.PathListSeparator) + cliPath(c)
}

func isCLIWrapper(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	// Every wrapper PiCode writes opens `#!/bin/sh` + a `# PiCode …`
	// comment naming the integration; vendor binaries never do. Match the
	// comment, not one flavor's wording — the omp/grok/hermes/muse shims
	// say "integration"/"plugin", and when only "# PiCode intercept"
	// matched, those CLIs resolved to their shim, DetectMethod saw an
	// unclassifiable script, and the lifecycle menu vanished (2026-09-24).
	b, _ := io.ReadAll(io.LimitReader(f, 256))
	rest, ok := bytes.CutPrefix(b, []byte("#!/bin/sh\n"))
	return ok && bytes.HasPrefix(rest, []byte("# PiCode "))
}

// resolveRetryDelay rides out the window a vendor self-update leaves its
// launcher or symlink missing mid-swap (2026-09-18: four rows flipped to
// not-installed for one request while their CLIs updated themselves, then
// healed on the next). Package var so the selftest can shorten it.
var resolveRetryDelay = 150 * time.Millisecond

func resolveInstalledCLI(cli clilaunch.CLI, c clilaunch.Config) (string, error) {
	p, err := resolveCLIExecutable(cli, c)
	if err == nil {
		return p, nil
	}
	time.Sleep(resolveRetryDelay)
	return resolveCLIExecutable(cli, c)
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
	// They stay separate flags: a future surface ("terminal") opens a
	// terminal with no adapter behind it.
	IntegrationCapable bool `json:"integrationCapable"`
	Launchable         bool `json:"launchable"`
	// ToolsCapable says PiCode can inject its own tool servers into this
	// CLI's launches (ADR-0154); the others take them at workspace scope.
	ToolsCapable bool `json:"toolsCapable"`
	// Sessions advertises what this CLI's session source can do (list,
	// read a transcript, receive a native session, start from a brief) so
	// the web derives handoff targets from the server (ADR-0088).
	Sessions clisession.Capabilities `json:"sessions"`
	// HasIntegrationMechanism says the Activity reporting toggle may be
	// offered: a plan summary alone (Muse Code today) is not something to
	// enable. The web hides the toggle and the PUT guards refuse it.
	HasIntegrationMechanism bool `json:"hasIntegrationMechanism"`
}

func describeCLI(deps Deps, cli clilaunch.CLI) (cliView, error) {
	c, err := cliConfig(deps, cli.ID)
	if err != nil {
		return cliView{}, err
	}
	v := cliView{CLI: cli, Config: c, IntegrationApplied: cliIntegrationPrepared(deps.DataDir, cli), Sessions: clisession.CapabilitiesOf(cli.ID), IntegrationCapable: cli.Integrable(), Launchable: cli.Launchable(), HasIntegrationMechanism: hasIntegrationMechanism(cli.ID), ToolsCapable: hasToolLaunchMechanism(cli.ID)}
	v.Sessions.Agent = cliAgentLanding(cli.ID)
	v.Plan, _ = launchPlan(deps, cli, c, clilaunch.Overrides{}, filepath.Join(deps.DataDir, "cli-launch", "{terminal}", "run-{next}"))
	v.Executable, err = resolveInstalledCLI(cli, c)
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
	if id == "agy" {
		// The reporter lives in the user's settings.json, not our data
		// dir: off must stop the reports, not just the map entry.
		removeAgyTitleReporter(deps.DataDir)
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
		if c.Integration && !hasIntegrationMechanism(cli.ID) {
			writeErr(w, 400, fmt.Sprintf("Activity reporting is not available for %s in this build.", cli.Name))
			return
		}
		if err := toolLaunchRefusal(cli, c); err != nil {
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
		wc, ok := clilaunch.Find(v.CLI)
		if !ok {
			writeErr(w, 400, "Unknown CLI.")
			return
		}
		c, err := cliConfig(deps, v.CLI)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		merged := clilaunch.Resolve(c, v.Overrides)
		// ADR-0184: a launch belongs to an agent. A shell becomes one through
		// Make agent, never by gaining a launch here; a sign-in's launch is
		// the credential flow's alone.
		t, err := deps.Store.GetTerminal(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if t.Kind != "" {
			writeErr(w, 409, "This terminal belongs to a sign-in.")
			return
		}
		a, e := deps.Store.AgentByTerminal(id)
		if e != nil {
			writeErr(w, 409, "Only an agent's terminal has launch settings. Make it an agent first.")
			return
		}
		{
			if v.CLI != a.CLI {
				writeErr(w, 400, "Choose a new agent to use a different CLI.")
				return
			}
			if a.IsPi() {
				if err := validatePiAgentArgs(merged.Args); err != nil {
					writeErr(w, 400, err.Error())
					return
				}
			}
		}
		if err := clilaunch.Validate(merged); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if merged.Integration && !hasIntegrationMechanism(v.CLI) {
			writeErr(w, 400, fmt.Sprintf("Activity reporting is not available for %s in this build.", wc.Name))
			return
		}
		if err := toolLaunchRefusal(wc, merged); err != nil {
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

// cliTerminalRequest is what a server-side launch asks for: the handoff
// endpoint (ADR-0088) composes it for createCLIAgent.
type cliTerminalRequest struct {
	Name        string              `json:"name"`
	WorkspaceID string              `json:"workspaceId"`
	Cwd         string              `json:"cwd"`
	Overrides   clilaunch.Overrides `json:"overrides"`
	// session, server-side only, runs once the agent exists and before its
	// first start, and names the conversation file the agent owns (a Pi
	// fork made in the new agent's own session folder). An error removes
	// the agent.
	session func(store.Agent) (string, error)
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

// createCLIAgent is the server-side launch door (ADR-0184): the launch
// pre-flight, then an agent — in the workspace, or free — whose
// terminal carries the launch, started at once. A server-side launch (the
// cross-CLI handoff) uses it so the CLI it opens is in the fleet like any
// other; only the credential sign-in keeps a bare terminal.
func createCLIAgent(deps Deps, r *http.Request, cli clilaunch.CLI, v cliTerminalRequest) (store.Agent, map[string]any, int, error) {
	if _, status, err := checkCLILaunch(deps, cli, v.Overrides); err != nil {
		return store.Agent{}, nil, status, err
	}
	if strings.TrimSpace(v.Name) == "" {
		v.Name = cli.Name
	}
	if err := launchFolderExists(v.Cwd); err != nil {
		return store.Agent{}, nil, http.StatusBadRequest, err
	}
	wsID, cwd, work := store.FreeWorkspaceID, "", ""
	if v.WorkspaceID != "" && v.WorkspaceID != store.FreeWorkspaceID {
		wk, err := deps.Store.GetWorkspace(v.WorkspaceID)
		if err != nil {
			return store.Agent{}, nil, storeStatus(err), err
		}
		wsID, cwd = wk.ID, wk.Path
	}
	if strings.TrimSpace(v.Cwd) != "" || wsID == store.FreeWorkspaceID {
		dir, err := resolveAgentWorkDir(deps, v.Cwd, v.Name)
		if err != nil {
			return store.Agent{}, nil, http.StatusBadRequest, err
		}
		cwd, work = dir, dir
	}
	// A Pi agent owns its session: `--session` is reserved on its launch
	// (validatePiAgentArgs) and comes from SessionPath instead.
	ov, session := v.Overrides, ""
	if cli.ID == store.CLIPi && ov.Args != nil {
		args, path := takeFlagValue(*ov.Args, "--session")
		ov.Args, session = &args, path
	}
	if cli.ID == store.CLIPi && ov.Args != nil {
		if err := validatePiAgentArgs(*ov.Args); err != nil {
			return store.Agent{}, nil, http.StatusBadRequest, err
		}
	}
	unlockWorkspace := terminalLock(deps, "workspace:"+wsID)
	defer unlockWorkspace()
	agent, status, err := newLaunchAgent(deps, wsID, cwd, cli.ID, v.Name, work, &ov)
	if err != nil {
		return agent, nil, status, err
	}
	if v.session != nil {
		path, err := v.session(agent)
		if err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			return store.Agent{}, nil, http.StatusBadGateway, err
		}
		session = path
	}
	if session != "" {
		updated, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &session})
		if err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			return store.Agent{}, nil, http.StatusInternalServerError, err
		}
		agent = updated
	}
	t, err := deps.Store.GetTerminal(*agent.TerminalID)
	if err != nil {
		_ = deps.Store.DeleteAgent(agent.ID)
		return store.Agent{}, nil, http.StatusInternalServerError, err
	}
	unlock := terminalLock(deps, t.ID)
	defer unlock()
	name := tmux.ShellSessionName(t.ID)
	created, err := ensureShell(deps, r, name, t.ID, t.Cwd)
	if err != nil {
		publishTerminalState(deps, r, t, false)
		return agent, map[string]any{"id": t.ID, "agentId": agent.ID, "launchError": err.Error()}, http.StatusCreated, nil
	}
	publishTerminalState(deps, r, t, true)
	view := termViewForCreation(deps, r, t, name, created)
	view["agentId"] = agent.ID
	return agent, view, http.StatusCreated, nil
}

// takeFlagValue removes `flag value` (or `flag=value`) from args and
// returns the rest with the value.
func takeFlagValue(args []string, flag string) ([]string, string) {
	out, value := make([]string, 0, len(args)), ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == flag && i+1 < len(args):
			value = args[i+1]
			i++
		case strings.HasPrefix(args[i], flag+"="):
			value = strings.TrimPrefix(args[i], flag+"=")
		default:
			out = append(out, args[i])
		}
	}
	return out, value
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
	if cli.ID == "pi" && c.Executable == "" && deps.Store != nil {
		if a, e := deps.Store.AgentByTerminal(v.TerminalID); e == nil && a.IsPi() {
			c.Executable = deps.AgentCmd
		}
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
		unlockAgent := boundAgentLock(deps, id)
		defer unlockAgent()
		if a, e := deps.Store.AgentByTerminal(id); e == nil && deps.Replies != nil {
			release := deps.Replies.Controls.BeginMutation(a.ID)
			defer release()
		}
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
			prepared, perr := prepareCLITerminal(deps, t.Cwd, agentAwareResumeLaunch(deps, launch))
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
			// Pin first so the next generation can reopen the conversation
			// (ADR-0158). Prepare still runs before kill: a bad resume recipe
			// must not destroy the live pane.
			pinCLITerminalLastSession(deps, id, t)
			launch, err = deps.Store.TerminalLaunch(id)
			if err != nil {
				writeStoreErr(w, err)
				return
			}
			if launch != nil {
				prepared, err = prepareCLITerminal(deps, t.Cwd, agentAwareResumeLaunch(deps, launch))
				if err != nil {
					recordCLILaunchAttempt(deps, id, err)
					writeErr(w, 400, err.Error())
					return
				}
				defer prepared.discard()
			}
			if st, err := os.Stat(agentTerminalCwd(deps, id, t.Cwd)); err != nil || !st.IsDir() {
				writeErr(w, 400, "That folder no longer exists.")
				return
			}
		}
		if action != "start" {
			if a, e := deps.Store.AgentByTerminal(id); e == nil && a.IsPi() {
				if err := deps.stopAgentInteractive(r.Context(), a.ID); err != nil {
					writeErr(w, 500, err.Error())
					return
				}
				if action == "remove" && deps.Runtime != nil {
					deps.Runtime.Stop(a.ID)
				}
			}
			// Last chance to pin the native conversation before the pane goes
			// away (ADR-0084).
			if action != "remove" {
				pinCLITerminalLastSession(deps, id, t)
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
			// Removing an agent's terminal ends the agent (ADR-0194).
			if err := deps.endTerminalAgent(id, exitRequest{}); err != nil {
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
			if _, err := ensureShell(deps, r, name, id, t.Cwd); err != nil {
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

// pinCLITerminalLastSession is the last-chance pin before stop/restart
// (ADR-0084). Wrapper runtimes pin from the live run; wrapper-less CLIs
// (muse, agy) never register one, so the latest session written in this
// folder after the terminal's creation is used instead. Same Latest()
// heuristic (and same grok/hermes exclusion inside). A missed pin only
// costs the resume shortcut and Continue in…, never the terminal.
func pinCLITerminalLastSession(deps Deps, id string, t store.Terminal) {
	pinned := false
	if deps.TermRuntimes != nil {
		if rt, ok := deps.TermRuntimes.Get(id); ok && rt.CLI != "" {
			pinTerminalLastSession(deps, id, rt)
			pinned = true
		}
	}
	if pinned || deps.Store == nil {
		return
	}
	launch, err := deps.Store.TerminalLaunch(id)
	if err != nil || launch == nil || launch.CLI == "" {
		return
	}
	var since time.Time
	if at, err := time.Parse(time.RFC3339Nano, t.CreatedAt); err == nil {
		since = at
	}
	pinTerminalLastSession(deps, id, TermRuntime{CLI: launch.CLI, StartedAt: since})
}

// pinTerminalLastSession resolves the native CLI session this terminal is
// running and pins it on the launch record (ADR-0084). Sessions last
// written before the run's start never win, so one terminal cannot steal
// another's conversation in a shared folder. Best effort by design: a
// failed pin only costs the resume shortcut, never the terminal.
func pinTerminalLastSession(deps Deps, termID string, runtime TermRuntime) {
	if deps.Store == nil || runtime.CLI == "" || runtime.CLI == "grok" || runtime.CLI == "hermes" {
		return
	}
	if runtime.SessionID != "" {
		if runtime.CLI == "codex" {
			if terminal, err := deps.Store.GetTerminal(termID); err == nil {
				if session, err := clisession.CodexByID(terminal.Cwd, runtime.SessionID); err == nil && session != nil {
					_ = deps.Store.SetTerminalLastSession(termID, store.TerminalLastSession{
						CLI: session.CLI, SessionID: session.ID, Path: session.Path,
						Cwd: session.Cwd, Name: session.Name, UpdatedAt: session.UpdatedAt,
						Preview: session.Preview, ResumeArgs: session.ResumeArgs,
					})
				}
			}
		}
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
	// Muse updates its SQLite index asynchronously after the terminal exits.
	// Give that indexer a short, bounded window before treating the session as
	// absent; otherwise a completed Muse conversation loses Continue in… until
	// the next terminal refresh.
	if runtime.CLI == "muse" && s == nil && err == nil {
		deadline := time.NewTimer(2 * time.Second)
		tick := time.NewTicker(100 * time.Millisecond)
		defer deadline.Stop()
		defer tick.Stop()
		for s == nil {
			select {
			case <-deadline.C:
				return
			case <-tick.C:
				s, err = clisession.Latest(runtime.CLI, t.Cwd, runtime.StartedAt)
				if err != nil {
					return
				}
			}
		}
	}
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

// ompCredentialEnv turns the vault's newest non-paused credential per
// provider into the env names omp's own declaration declares — the one
// channel a guest whose store PiCode cannot write has (ADR-0178). A provider
// the user already envs in their launch config is skipped, never clobbered.
func ompCredentialEnv(file *credentials.File, have map[string]string) [][2]string {
	if file == nil {
		return nil
	}
	spec, ok := clicreds.For("omp")
	if !ok {
		return nil
	}
	var out [][2]string
	for _, declared := range spec.Providers {
		slot, ok := file.Providers[declared.Provider]
		if !ok {
			continue
		}
		// Last written row wins: the vault appends, and the newest credential
		// is the one a fresh sign-in just minted. Paused rows are skipped.
		for i := len(slot.Accounts) - 1; i >= 0; i-- {
			row := slot.Accounts[i]
			if row.Paused || row.Type != "oauth" && row.Type != catalog.LoginAPIKey {
				continue
			}
			var cred struct {
				Key    string `json:"key"`
				Access string `json:"access"`
			}
			if json.Unmarshal(row.Cred, &cred) != nil {
				continue
			}
			for kind, envName := range declared.Env {
				value := cred.Key
				if kind == clicreds.KindOAuth {
					value = cred.Access
				}
				if value == "" || have[envName] != "" {
					continue
				}
				out = append(out, [2]string{envName, value})
			}
			break
		}
	}
	return out
}

// launchIdentityEnv is the caller identity a CLI launch carries (ADR-0160
// Fatia E): PICODE_TERM_ID always, plus PICODE_AGENT_ID when the terminal is
// bound to an agent. picode mcp resolves principal with agent-wins
// (grant.FromIDs), so grants given to the CLI agent in Settings ▸
// Computer / Browser match without per-terminal configuration — the same
// spelling Pi agents already get from Agent.SpawnEnv.
func launchIdentityEnv(deps Deps, termID string) []string {
	env := []string{tmux.MarkerTermEnv + "=" + termID}
	if deps.Store != nil {
		if a, err := deps.Store.AgentByTerminal(termID); err == nil && a.ID != "" {
			env = append(env, tmux.MarkerAgentEnv+"="+a.ID)
		}
	}
	return env
}

// ompAgentSessionDir is durable across terminal generations, but lives in
// PiCode's data directory so it is owned by the workspace agent rather than
// by Omp's shared default home. The directory contains transcripts only;
// Omp authentication and configuration remain in the user's normal profile.
func ompAgentSessionDir(dataDir, agentID string) string {
	return filepath.Join(dataDir, "omp-sessions", agentID)
}

// agentOmpScopeFlags is the agent's own scope as Omp's launch takes it: every
// entry of the agent's list as `-e`, and — when "only this agent's packages" is
// on — the two flags this CLI has for skipping what it would auto-load. Omp has
// no --no-prompt-templates/--no-themes (measured 2026-09-21), so its isolation
// is those two alone, never a flag the CLI would refuse. Pi's list rides the
// same store field with pi's own flags (`store.Agent.CLIFlags`); this is what
// makes the packages pane's agent scope real for Omp (ADR-0176 slice 4).
func agentOmpScopeFlags(a store.Agent) []string {
	var args []string
	if a.PackagesIsolated {
		args = append(args, "--no-extensions")
		// --no-skills also drops skills.customDirectories (measured on
		// 18.2.11): an isolated agent with skills of its own gets an overlay
		// that turns the folders off instead (agentOmpSkillFlags).
		if len(a.SkillDirs()) == 0 {
			args = append(args, "--no-skills")
		}
	}
	for _, src := range a.Packages {
		if src = strings.TrimSpace(src); src != "" {
			args = append(args, "-e", src)
		}
	}
	return args
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

// ompArgsPickSession reports whether launch arguments already name the
// conversation or its folder.
func ompArgsPickSession(args []string) bool {
	for _, a := range args {
		key, _, _ := strings.Cut(a, "=")
		if key == "--resume" || key == "--session-dir" {
			return true
		}
	}
	return false
}

func prepareCLITerminal(deps Deps, cwd string, v *store.TerminalLaunch) (*preparedCLILaunch, error) {
	cwd = agentTerminalCwd(deps, v.TerminalID, cwd)
	if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
		return nil, errors.New("That folder no longer exists.")
	}
	cli, c, binary, err := resolvedTerminalLaunch(deps, v)
	if err != nil {
		return nil, err
	}
	if peerResumeExact(c.Args, v.LastSession) {
		if err := validatePinnedResume(v.LastSession, cwd); err != nil {
			return nil, err
		}
	}
	var piAgent *store.Agent
	agentFingerprint := ""
	// An agent's own scope (packages, isolation, skills) is launch
	// configuration too: a change to it is a restart the pane announces, for
	// every CLI that takes one (ADR-0196 slice 4). Read before any injection.
	var agent *store.Agent
	if a, e := deps.Store.AgentByTerminal(v.TerminalID); e == nil {
		agent = &a
		agentFingerprint = agentLaunchFingerprint(c, a)
	}
	if cli.ID == "pi" {
		if a, e := deps.Store.AgentByTerminal(v.TerminalID); e == nil && a.IsPi() {
			if err := validatePiAgentArgs(c.Args); err != nil {
				return nil, err
			}
			if _, err := pimission.Ensure(deps.DataDir); err != nil {
				return nil, fmt.Errorf("prepare native mission extension: %w", err)
			}
			piAgent = &a
			flags := deps.piSpawnFlags(a, !c.Integration)
			c.Args = append(c.Args, flags...)
			for _, entry := range a.SpawnEnv() {
				k, value, _ := strings.Cut(entry, "=")
				c.Env[k] = value
			}
			c.Env["PICODE_DATA"] = deps.DataDir
		}
	}
	// Omp exposes a native session-storage boundary. Keep each workspace
	// agent's `/resume` picker inside its own durable directory, while leaving
	// Omp's shared auth/configuration untouched. Free-standing Omp terminals
	// retain the vendor default because they have no agent owner to scope.
	// An agent launched on a conversation that lives elsewhere — a resumed
	// session or a handoff (ADR-0184), whose file is in Omp's default
	// folder — keeps that folder: its explicit arguments win.
	if cli.ID == "omp" {
		if a, e := deps.Store.AgentByTerminal(v.TerminalID); e == nil && a.CLI == "omp" && !ompArgsPickSession(c.Args) {
			if err := os.MkdirAll(ompAgentSessionDir(deps.DataDir, a.ID), 0o700); err != nil {
				return nil, err
			}
			c.Args = append(c.Args, "--session-dir", ompAgentSessionDir(deps.DataDir, a.ID))
			// The integration guard (cli_plan.go) refuses the same conflict for
			// the activity extension; the agent's own entries are injected
			// after that guard runs, so they are checked here — omp refuses a
			// run outright when --trusted-extension meets any -e, and a launch
			// that cannot run is not worth starting.
			skillFlags, err := agentOmpSkillFlags(deps.DataDir, a)
			if err != nil {
				return nil, err
			}
			c.Args = append(c.Args, skillFlags...)
			if flags := agentOmpScopeFlags(a); len(flags) > 0 {
				if slices.Contains(flags, "-e") && ompTrustedExtensionConflict(c) {
					return nil, errors.New("Your --trusted-extension argument conflicts with this agent's own extensions. Drop the flag, or remove the entries in the agent's package scope.")
				}
				c.Args = append(c.Args, flags...)
			}
		}
		// Vault credentials travel by env (ADR-0178): the env names are the
		// ones omp's own declaration declares, and env is the lowest channel
		// in omp's resolution chain — a native /login outranks an injected
		// token, and a user-configured env entry outranks us both.
		if file, err := credentials.Default().Load(); err == nil {
			for _, kv := range ompCredentialEnv(&file, c.Env) {
				c.Env[kv[0]] = kv[1]
			}
		}
	}
	// Claude Code's Console key travels by env, and only when the person
	// chose it with Use (ADR-0187): the key outranks the subscription file.
	if cli.ID == "claude-code" {
		if c.Env == nil {
			c.Env = map[string]string{}
		}
		for _, kv := range claudeCredentialEnv(deps, c.Env) {
			c.Env[kv[0]] = kv[1]
		}
	}
	// Grok's chosen key travels by env (ADR-0192).
	if cli.ID == "grok" {
		if c.Env == nil {
			c.Env = map[string]string{}
		}
		for _, kv := range grokCredentialEnv(deps, c.Env) {
			c.Env[kv[0]] = kv[1]
		}
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
	// Claude Code takes an agent's own skills as a session-only plugin
	// folder (measured on 2.1.281: no manifest needed, skills invoked as
	// /picode-agent:<name>).
	if cli.ID == "claude-code" && agent != nil && agent.CLI == "claude-code" {
		plugin, err := writeClaudeAgentPlugin(dir, *agent)
		if err != nil {
			return nil, err
		}
		if plugin != "" {
			c.Args = append(c.Args, "--plugin-dir", plugin)
		}
	}
	command := binary
	// Wrapper-less reporters (agy: settings.json title command) need no
	// PATH shadow; the switch below only writes intercept wrappers.
	needsWrapper := true
	if c.Integration && !hasIntegrationMechanism(cli.ID) {
		// Backstop for every path that skips the PUT guards (a profile
		// carrying integration:true): fail here with the reason, never with
		// a missing wrapper file.
		return nil, fmt.Errorf("Activity reporting is not available for %s in this build.", cli.Name)
	}
	if c.Integration && cli.ID == "omp" && ompTrustedExtensionConflict(c) {
		// Measured on 18.2.4: omp refuses the whole run when a
		// --trusted-extension argument meets PiCode's injected -e.
		return nil, fmt.Errorf("Your --trusted-extension argument conflicts with PiCode's Omp activity extension. Turn Activity reporting off for Omp, or drop the flag.")
	}
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
		case "muse":
			err = writeMuseIntercept(dir, hook)
		case "omp":
			err = writeOmpIntercept(dir, hook)
		case "agy":
			err = writeAgyIntercept(dir, hook)
		}
		if err != nil {
			return nil, err
		}
		if needsWrapper {
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
	if piAgent != nil {
		peerOptions, err = communication.AgentOptions(deps.Store, deps.DataDir, piAgent.ID)
		if err != nil {
			return nil, err
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

	// PiCode tools (ADR-0154): the families in the launch settings ride the
	// same options as the peer-communication server.
	if families, err := toolFamilies(c); err != nil {
		return nil, err
	} else if len(families) > 0 {
		if !hasToolLaunchMechanism(cli.ID) {
			return nil, fmt.Errorf("PiCode tools at launch are not available for %s yet — add the PiCode connector in its Connectors pane instead.", cli.Name)
		}
		existing, specified := c.Env["OPENCODE_CONFIG_CONTENT"]
		if !specified {
			existing = os.Getenv("OPENCODE_CONFIG_CONTENT")
		}
		peerOptions, err = toolLaunchOptions(cli.ID, families, dir, peerOptions, existing, toolIdentityEnv(deps, v.TerminalID))
		if err != nil {
			return nil, err
		}
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
	fmt.Fprintf(&body, "export PATH=%s\n", shellQuote(interceptCLIPath(deps.DataDir, c)))
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
	if t, e := deps.Store.GetTerminal(v.TerminalID); e == nil && t.Kind == store.TerminalKindSignin {
		// A sign-in ends with its CLI (ADR-0184): no shell is left behind
		// in a terminal nobody sees; the reaper then finds no session.
		body.WriteString("exit \"$picode_exit\"\n")
	} else {
		// Returning to the normal shell also restores the manual CLI wrappers.
		rc, err := ensureInterceptBashrc(deps.DataDir)
		if err != nil {
			return nil, err
		}
		shell := defaultShell()
		fmt.Fprintf(&body, "exec %s", shellQuote(shell))
		if shellTakesRcfile(shell) {
			fmt.Fprintf(&body, " --rcfile %s", shellQuote(rc))
		}
		body.WriteByte('\n')
	}
	script := filepath.Join(dir, "launch.sh")
	if err := os.WriteFile(script, []byte(body.String()), 0o700); err != nil {
		return nil, err
	}
	snapshot := clilaunch.Describe(c, binary, "")
	if agentFingerprint != "" {
		snapshot.Fingerprint = agentFingerprint
	}
	snapshot.CLI = cli.ID
	snapshot.Identity = executableIdentity(binary)
	// Which skills this start loads, for Outcomes (ADR-0196 slice 6).
	if agent != nil {
		snapshot.Skills = launchSkills(deps, *agent)
	}
	if c.Integration {
		plan, _ := launchPlan(deps, cli, c, clilaunch.Overrides{}, dir)
		snapshot.Injection = &plan.Injection
	}
	// The pane environment carries the same PATH as the script: tmux -e
	// sets it from the first pane on, and dropEnv keeps a single PATH key
	// (tmux's -e is last-wins, but one key beats hoping).
	paneEnv := dropEnv(cliEnvironment(c), "PATH")
	paneEnv = append(paneEnv, "PATH="+interceptCLIPath(deps.DataDir, c))
	started = true // ownership transfers to the prepared object
	return &preparedCLILaunch{dir: dir, script: script, id: v.TerminalID, snapshot: snapshot, environment: paneEnv}, nil
}

func (p *preparedCLILaunch) start(deps Deps, r *http.Request, name, cwd string) error {
	return p.startSized(deps, r, name, cwd, 0, 0)
}

func (p *preparedCLILaunch) startSized(deps Deps, r *http.Request, name, cwd string, width, height int) error {
	cwd = agentTerminalCwd(deps, p.id, cwd)
	if a, e := deps.Store.AgentByTerminal(p.id); e == nil && a.IsPi() && deps.Tmux != nil {
		if blocked, e := peerStopPending(deps, a.ID); e != nil || blocked {
			return errAgentTUIInFlight
		}
		if has, e := deps.Tmux.HasSession(r.Context(), tmux.SessionName(a.ID)); e != nil || has {
			return errors.New("This agent still has an open terminal. Close it through the agent before starting another.")
		}
	}
	if a, e := deps.Store.AgentByTerminal(p.id); e == nil && a.IsPi() && deps.Runtime != nil && deps.Runtime.Active(a.ID) {
		return errors.New("This agent is running in chat. Open its terminal through the agent to switch modes.")
	}
	if blocked, e := peerStopPending(deps, p.id); e != nil || blocked {
		return errors.New("The previous process has not finished closing. Try again after it exits.")
	}
	env := append(append([]string{}, p.environment...), launchIdentityEnv(deps, p.id)...)
	env = append(env, tmux.MarkerURLEnv+"="+loopbackURL(deps))
	if b := interceptBinEnv(deps.DataDir); b != "" {
		env = append(env, b)
	}
	env = append(env, openURLEnv(deps.DataDir)...) // ADR-0180: login pages open where the user is looking
	if err := deps.Tmux.NewSessionEnvSize(r.Context(), name, cwd, width, height, env, "/bin/sh", p.script); err != nil {
		return err
	}
	p.snapshot.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := deps.Store.SetTerminalLaunchApplied(p.id, p.snapshot); err != nil {
		if p.snapshot.CLI == "pi" {
			// The failing store cannot be trusted to resolve ownership now.
			// This prepared launch already owns the exact pane and receipt key.
			if stopErr := stopInteractivePane(r.Context(), deps, name, p.id); stopErr != nil {
				p.started = true // retain files referenced by the unconfirmed writer
				return errors.Join(err, stopErr)
			}
		} else {
			_ = deps.Tmux.KillSession(r.Context(), name)
		}
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
	_, effective, binary, err := resolvedTerminalLaunch(deps, v)
	if err == nil && v.Applied != nil {
		fingerprint := clilaunch.Fingerprint(effective)
		if a, e := deps.Store.AgentByTerminal(id); e == nil {
			fingerprint = agentLaunchFingerprint(effective, a)
		}
		view["launchPending"] = v.Applied.CLI != v.CLI || v.Applied.Fingerprint != fingerprint || binary != v.Applied.Executable || (v.Applied.Identity != "" && v.Applied.Identity != executableIdentity(binary))
	}
}

// Pi's catalog uses its own session UI, but a terminal resume needs its file.
func terminalResumeArgs(ls *store.TerminalLastSession) []string {
	if ls == nil {
		return nil
	}
	if ls.CLI == "pi" && ls.Path != "" {
		return []string{"--session", ls.Path}
	}
	if ls.CLI == "omp" && ls.Path != "" {
		// Omp accepts an exact file path. Its agent sessions live outside
		// the default home, so resuming by ID alone searches the wrong root.
		return []string{"--resume", ls.Path}
	}
	return append([]string{}, ls.ResumeArgs...)
}

// Restart prepares before ending the current pane. Check the two native
// stores whose observed pins can be unusable despite having a resume ID:
// Codex announces IDs before saving rollouts, and Omp agent files live in a
// private directory outside its default ID lookup.
func validatePinnedResume(ls *store.TerminalLastSession, cwd string) error {
	if ls == nil {
		return nil
	}
	switch ls.CLI {
	case "codex":
		session, err := clisession.CodexByID(cwd, ls.SessionID)
		if err != nil || session == nil {
			return errors.New("The saved Codex conversation is not available for resume. Choose a saved session or start a new conversation.")
		}
	case "omp":
		info, err := os.Stat(ls.Path)
		if ls.Path == "" || err != nil || !info.Mode().IsRegular() {
			return errors.New("The saved Omp conversation is not available for resume. Choose a saved session or start a new conversation.")
		}
	}
	return nil
}

// launchWithPinnedSession returns a one-shot copy of v whose argument
// override is the pinned session's resume recipe (ADR-0084, ADR-0158).
// The store record is not written. Other overrides (env, path, executable,
// tools) stay. When there is nothing to resume, v is returned unchanged.
func launchWithPinnedSession(v *store.TerminalLaunch) *store.TerminalLaunch {
	if v == nil || v.LastSession == nil || v.LastSession.SessionID == "" {
		return v
	}
	args := terminalResumeArgs(v.LastSession)
	if len(args) == 0 {
		return v
	}
	cp := *v
	cp.Overrides.Args = &args
	return &cp
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
