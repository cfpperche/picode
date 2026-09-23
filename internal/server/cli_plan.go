package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/communication"
)

var piPassthrough = []string{"auth", "config", "install", "list", "remove", "uninstall", "update", "--help", "-h", "--version", "-v"}

// The adapter writers and preview share these argument vectors. Codex chooses
// its capability branch inside the wrapper; read-only previews never run help.
func cliIntegrationPlan(cli, dir, hook string) clilaunch.IntegrationPlan {
	p := clilaunch.IntegrationPlan{Branches: []clilaunch.Injection{}, Files: []string{}, Environment: map[string]string{}}
	switch cli {
	case "claude-code":
		p.Summary = "Activity hooks via --settings"
		p.Branches = append(p.Branches, clilaunch.Injection{When: "Every launch", Args: []string{"--settings", claudeSettingsFile(dir)}})
		p.Files = append(p.Files, claudeSettingsFile(dir))
	case "pi":
		p.Summary = "Activity and Ask-receiver extensions via -e"
		// The receiver (ADR-0060) rides along since ADR-0089's amendment: it is
		// what lets the git graph and the Inspector ask this pi to do a git
		// action in its own turn, with the session row as proof.
		p.Branches = append(p.Branches, clilaunch.Injection{When: "Interactive invocation (not maintenance or help/version)", Args: []string{"-e", piTerminalStateExtensionFile(dir), "-e", piReplyExtensionFile(dir)}})
		p.Files = append(p.Files, piTerminalStateExtensionFile(dir), piReplyExtensionFile(dir))
	case "codex":
		p.Summary = "Hooks or completion fallback via -c"
		full := []string{}
		overrides := codexHookOverrides(hook)
		for _, v := range overrides {
			full = append(full, "-c", v)
		}
		p.Branches = append(p.Branches, clilaunch.Injection{When: "When the installed CLI advertises hook trust support", Args: full}, clilaunch.Injection{When: "Otherwise: completion notification only", Args: []string{"-c", codexNotifyOverride(hook)}})
	case "grok", "hermes":
		p.Summary = "Native session hooks and plugin (installed on launch)"
		p.Environment["PICODE_NATIVE_HOOK"] = hook
		p.Files = append(p.Files, filepath.Join(nativeAssetsDir(dir), "install.py"))
	case "opencode":
		p.Summary = "Activity plugin in PiCode terminals only (does not write your OpenCode config)"
		p.Environment["OPENCODE_CONFIG"] = opencodeConfigFile(dir)
		p.Environment["PICODE_OPENCODE_HOOK"] = hook
		p.Files = append(p.Files, opencodeConfigFile(dir), opencodePluginFile(dir))
	case "muse":
		// Presence lease and native runtime via the PATH wrapper (same
		// shape as every other CLI). Hook reports stay unattributable —
		// the CLI scrubs the hook env to PATH — so activity stays Open;
		// the wrapper file below is what cliIntegrationPrepared checks.
		p.Summary = "Presence lease via the PATH wrapper (hook reports carry no terminal identity)"
		p.Files = append(p.Files, wrapperPath(dir, "muse"))
	case "agy":
		// Reports through the settings.json title command installed by
		// installIntercept, plus the PATH wrapper for presence and precise
		// pins. The reporter file below is what cliIntegrationPrepared checks.
		p.Summary = "Activity via the title reporter in your Antigravity settings, presence via the PATH wrapper"
		p.Files = append(p.Files, agyTitleReporterPath(dir), wrapperPath(dir, "agy"))
	case "omp":
		// omp is a Pi fork: the same extension API, measured live (the
		// session events fire and PICODE_TERM_ID survives). The extension
		// self-guards on TUI mode, so headless -p runs report nothing.
		p.Summary = "Activity extension via -e (omp shares pi's extension API)"
		p.Branches = append(p.Branches, clilaunch.Injection{When: "Interactive invocation (not maintenance subcommands, protocol modes, help/version)", Args: []string{"-e", ompTerminalStateExtensionFile(dir)}})
		p.Files = append(p.Files, wrapperPath(dir, "omp"), ompTerminalStateExtensionFile(dir))
	}
	return p
}

// hasIntegrationMechanism says the Activity reporting toggle may be offered:
// a CLI whose plan carries nothing to install or inject has nothing to
// enable, so the web hides the toggle and the PUT guards refuse it.
// Summary alone is not a mechanism.
func hasIntegrationMechanism(id string) bool {
	p := cliIntegrationPlan(id, "", "")
	return len(p.Branches) > 0 || len(p.Files) > 0 || len(p.Environment) > 0
}

// ompTrustedExtensionConflict: omp refuses a run outright when a
// --trusted-extension launch argument meets PiCode's injected -e
// (measured on 18.2.4: "--trusted-extension cannot be combined with
// --extension, -e, or --hook").
func ompTrustedExtensionConflict(c clilaunch.Config) bool {
	for _, a := range c.Args {
		if a == "--trusted-extension" || strings.HasPrefix(a, "--trusted-extension=") {
			return true
		}
	}
	return false
}

func quotedCLIArgs(args []string) string {
	var b strings.Builder
	for _, a := range args {
		b.WriteByte(' ')
		b.WriteString(shellQuote(a))
	}
	return b.String()
}

func launchPlan(deps Deps, cli clilaunch.CLI, base clilaunch.Config, overrides clilaunch.Overrides, dir string) (clilaunch.Plan, clilaunch.Config) {
	c := clilaunch.Resolve(base, overrides)
	p := clilaunch.Plan{Origins: map[string]string{}, InheritedPath: filepath.SplitList(os.Getenv("PATH")), ManagedEnv: []string{"PICODE_TERM_ID", "PICODE_TERM_URL"}}
	binary, err := resolveCLIExecutable(cli, c)
	if validation := clilaunch.Validate(c); validation != nil {
		err = validation
	}
	p.Snapshot = clilaunch.Describe(c, binary, "")
	p.Identity = executableIdentity(binary)
	p.CLI = cli.ID
	if err != nil {
		p.Problem = err.Error()
	}
	if p.Problem == "" && cli.ID == "omp" && c.Integration && ompTrustedExtensionConflict(c) {
		p.Problem = "Your --trusted-extension argument conflicts with PiCode's Omp activity extension. Turn Activity reporting off for Omp, or drop the flag."
	}
	for _, key := range []string{"executable", "args", "path", "env", "integration"} {
		p.Origins[key] = "CLI defaults"
	}
	if c.Executable == "" {
		p.Origins["executable"] = "Automatic detection"
	}
	if overrides.Executable != nil && *overrides.Executable != "" {
		p.Origins["executable"] = "Terminal override"
	}
	if overrides.Args != nil {
		p.Origins["args"] = "Terminal override"
	}
	if overrides.Path != nil {
		p.Origins["path"] = "Terminal override"
	}
	if len(overrides.Env) > 0 {
		p.Origins["env"] = "CLI defaults + terminal overrides"
	}
	if overrides.Integration != nil {
		p.Origins["integration"] = "Terminal override"
	}
	p.Origins["tools"] = "CLI defaults"
	if overrides.Tools != nil {
		p.Origins["tools"] = "Terminal override"
	}
	if families, err := toolFamilies(c); err != nil {
		if p.Problem == "" {
			p.Problem = err.Error()
		}
	} else {
		p.ToolInjection = toolLaunchPlan(cli, families, dir, toolIdentityEnvPreview())
	}
	if c.Integration {
		p.Injection = cliIntegrationPlan(cli.ID, dir, hookScriptPath(deps.DataDir))
		if cli.ID == "pi" && len(c.Args) > 0 {
			for _, cmd := range piPassthrough {
				if c.Args[0] == cmd {
					p.Injection.Summary = "No injection for maintenance or help/version"
					p.Injection.Branches = []clilaunch.Injection{}
					break
				}
			}
		}
		if communication.NativeMessages(cli.ID) {
			if options, err := communication.NativeOptions(deps.DataDir, cli.ID); err == nil {
				for k, v := range options.Env {
					p.Injection.Environment[k] = v
				}
				if len(options.Args) > 0 {
					p.Injection.Branches = append(p.Injection.Branches, clilaunch.Injection{When: "Native terminal messages", Args: options.Args})
				}
			}
			p.Injection.Files = append(p.Injection.Files, filepath.Join(dir, "message-bin", "picode"))
		}
		for _, f := range []string{wrapperPath(dir, cli.Command), hookScriptPath(deps.DataDir)} {
			if !slices.Contains(p.Injection.Files, f) {
				p.Injection.Files = append(p.Injection.Files, f)
			}
		}
		p.ManagedEnv = append(p.ManagedEnv, "PICODE_TUI_RUN_ID")
	}
	return p, c
}

func executableIdentity(binary string) string {
	st, err := os.Stat(binary)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", binary, st.Size(), st.ModTime().UnixNano())
}

func handleCLIPreview(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, 404, "Unknown CLI.")
			return
		}
		var v struct {
			Config     *clilaunch.Config   `json:"config"`
			Overrides  clilaunch.Overrides `json:"overrides"`
			TerminalID string              `json:"terminalId"`
		}
		if !readCLIJSON(w, r, &v) {
			return
		}
		base, err := cliConfig(deps, cli.ID)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if v.Config != nil {
			base = *v.Config
		}
		id := "{terminal}"
		if v.TerminalID != "" {
			if _, err := deps.Store.GetTerminal(v.TerminalID); err != nil {
				writeStoreErr(w, err)
				return
			}
			id = v.TerminalID
		}
		p, c := launchPlan(deps, cli, base, v.Overrides, filepath.Join(deps.DataDir, "cli-launch", id, "run-{next}"))
		if err := clilaunch.Validate(c); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		affected := []map[string]string{}
		terms, err := deps.Store.ListTerminals()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		current, err := cliConfig(deps, cli.ID)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		for _, t := range terms {
			if t.Kind != "" {
				continue // a sign-in is the credential flow's, not a terminal to warn about
			}
			v, err := deps.Store.TerminalLaunch(t.ID)
			if err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			if v != nil && v.CLI == cli.ID && clilaunch.Fingerprint(clilaunch.Resolve(current, v.Overrides)) != clilaunch.Fingerprint(clilaunch.Resolve(base, v.Overrides)) {
				affected = append(affected, map[string]string{"id": t.ID, "name": t.Name})
			}
		}
		writeJSON(w, 200, map[string]any{"plan": p, "affected": affected})
	}
}
