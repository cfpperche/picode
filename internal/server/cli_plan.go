package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
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
		p.Summary = "Activity extension via -e"
		p.Branches = append(p.Branches, clilaunch.Injection{When: "Interactive invocation (not maintenance or help/version)", Args: []string{"-e", piTerminalStateExtensionFile(dir)}})
		p.Files = append(p.Files, piTerminalStateExtensionFile(dir))
	case "codex":
		p.Summary = "Hooks or completion fallback via -c"
		full := []string{}
		overrides := codexHookOverrides(hook)
		for _, v := range overrides {
			full = append(full, "-c", v)
		}
		p.Branches = append(p.Branches, clilaunch.Injection{When: "When the installed CLI advertises hook trust support", Args: full}, clilaunch.Injection{When: "Otherwise: completion notification only", Args: []string{"-c", overrides[len(overrides)-1]}})
	case "grok":
		p.Summary = "Private activity overlay via GROK_HOME"
		p.Environment["GROK_HOME"] = grokHomeDir(dir)
		p.Files = append(p.Files, filepath.Join(grokHomeDir(dir), "hooks", "picode.json"))
	}
	return p
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
		p.Injection.Files = append(p.Injection.Files, wrapperPath(dir, cli.Command), hookScriptPath(deps.DataDir))
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
