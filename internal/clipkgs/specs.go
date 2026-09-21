package clipkgs

import (
	"context"
	"fmt"
	"strings"
)

// The declarations, one per CLI, in the order the Agent CLIs surface lists
// them. Every binary, subcommand and flag below was read on 2026-09-20 from
// the CLI installed on this machine (`<cli> plugin <verb> --help`) or from the
// vendor's own documentation where the CLI prints none.
//
// A verb with a nil builder is not implemented by us and not pretended: the
// note next to it is what the pane shows where the control would be. Flags a
// vendor offers to skip a human decision (Claude's `-y` and
// `--accept-command`, Grok's `--trust` and `--confirm`, Hermes'
// `--allow-tool-override`) are deliberately absent from every argv here —
// PiCode never answers a consent prompt for the user. Antigravity is the one
// CLI whose subcommands take no flags at all: its parser reads the first
// argument as the target, so `agy plugin uninstall --help` really uninstalls
// `--help` (measured 2026-09-20). Its argv stays positional.
//
// Pi is not declared here: its pane, API and semantics are its own
// (ADR-0102/0099) and `For("pi")` answers nil on purpose.

// The vendor binaries, as constants so a roster closure never reads a spec var
// it is itself initializing (Go refuses that cycle).
const (
	binClaude   = "claude"
	binCodex    = "codex"
	binGrok     = "grok"
	binHermes   = "hermes"
	binOpencode = "opencode"
	binMuse     = "muse"
	binAgy      = "agy"
	binOmp      = "omp"
)

// dirForScope resolves the working directory a scope's action runs in. The
// two project-ish scopes need the workspace folder: running them from the
// daemon's directory would install into whatever repository that happens to
// be, or fail with the vendor's own confusing error.
func dirForScope(p Paths, scope string) (string, error) {
	switch scope {
	case "", "user":
		return "", nil
	case "project", "local":
		if p.Cwd == "" {
			return "", ErrNoWorkspace
		}
		return p.Cwd, nil
	}
	return "", fmt.Errorf("%w: %s", ErrScope, scope)
}

func clean(s string) string { return strings.TrimSpace(s) }

// verbArgs is the shape every simple builder shares: a fixed prefix, the
// target, and scope flags at the end.
func verbArgs(prefix []string, target string, tail ...string) []string {
	args := append([]string{}, prefix...)
	if clean(target) != "" {
		args = append(args, clean(target))
	}
	return append(args, tail...)
}

// --- Claude Code ------------------------------------------------------------

var claude = &spec{
	cli:  "claude-code",
	name: "Claude Code",
	bin:  binClaude,
	scopes: []Scope{
		{ID: "user", Label: "This machine"},
		{ID: "project", Label: "This workspace", Note: "Declared in the repository's .claude/settings.json, so it travels with the checkout."},
		{ID: "local", Label: "This workspace (local)", Note: "Kept in .claude/settings.local.json, which is not committed."},
	},
	roster:         claudeRoster,
	available:      true,
	catalogInstall: true,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "install"}, t.Source, "-s", scopeOf(t.Scope), "--json"), err
		},
		VerbRemove: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "uninstall"}, t.Name, "-s", scopeOf(t.Scope), "--json"), err
		},
		VerbEnable: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "enable"}, t.Name, "-s", scopeOf(t.Scope), "--json"), err
		},
		VerbDisable: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "disable"}, t.Name, "-s", scopeOf(t.Scope), "--json"), err
		},
		VerbUpdate: func(p Paths, t Target) (string, []string, error) {
			// No scope flag on update: the vendor resolves it from the id.
			return "", verbArgs([]string{"plugin", "update"}, t.Name, "--json"), nil
		},
	},
	market: map[string]func(Paths, MarketRequest) (string, []string, error){
		"add": func(p Paths, r MarketRequest) (string, []string, error) {
			dir, err := dirForScope(p, r.Ref)
			return dir, verbArgs([]string{"plugin", "marketplace", "add"}, r.Source, "--scope", scopeOf(r.Ref)), err
		},
		"list": func(Paths, MarketRequest) (string, []string, error) {
			return "", []string{"plugin", "marketplace", "list", "--json"}, nil
		},
		"update": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "marketplace", "update"}, r.Name), nil
		},
		"remove": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "marketplace", "remove"}, r.Name), nil
		},
	},
	notes: map[string]string{
		"consent": "Claude Code's marketplace can run a declared install command, which its CLI asks a person to confirm. PiCode never answers that prompt: the row shows the command and this refusal instead.",
	},
}

// --- Codex ------------------------------------------------------------------

var codex = &spec{
	cli:  "codex",
	name: "Codex",
	bin:  binCodex,
	scopes: []Scope{
		{ID: "user", Label: "This machine"},
	},
	roster:         codexRoster,
	available:      true,
	catalogInstall: true,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "add"}, t.Source, "--json"), nil
		},
		VerbRemove: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "remove"}, t.Name, "--json"), nil
		},
	},
	market: map[string]func(Paths, MarketRequest) (string, []string, error){
		"add": func(_ Paths, r MarketRequest) (string, []string, error) {
			if clean(r.Source) == "" {
				return "", nil, fmt.Errorf("%w: name the marketplace source", ErrBadTarget)
			}
			args := verbArgs([]string{"plugin", "marketplace", "add"}, r.Source, "--json")
			if clean(r.Ref) != "" {
				args = append(args, "--ref", clean(r.Ref))
			}
			return "", args, nil
		},
		"list": func(Paths, MarketRequest) (string, []string, error) {
			return "", []string{"plugin", "marketplace", "list", "--json"}, nil
		},
		"update": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "marketplace", "upgrade"}, r.Name), nil
		},
		"remove": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "marketplace", "remove"}, r.Name), nil
		},
	},
	notes: map[string]string{
		"enable":  "Codex installs and removes plugins; its CLI exposes no enable or disable.",
		"disable": "Codex installs and removes plugins; its CLI exposes no enable or disable.",
		"update":  "Codex exposes no plugin update of its own: a configured marketplace refreshes with `codex plugin marketplace upgrade`.",
	},
}

// --- Grok -------------------------------------------------------------------

var grok = &spec{
	cli:  "grok",
	name: "Grok",
	bin:  binGrok,
	scopes: []Scope{
		{ID: "user", Label: "This machine"},
	},
	roster:         grokRoster,
	available:      true,
	catalogInstall: true,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "install"}, t.Source), nil
		},
		VerbRemove: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "uninstall"}, t.Name), nil
		},
		VerbEnable: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "enable"}, t.Name), nil
		},
		VerbDisable: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "disable"}, t.Name), nil
		},
		VerbUpdate: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "update"}, t.Name), nil
		},
		VerbInspect: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "details"}, t.Name), nil
		},
	},
	market: map[string]func(Paths, MarketRequest) (string, []string, error){
		"add": func(_ Paths, r MarketRequest) (string, []string, error) {
			if clean(r.Source) == "" {
				return "", nil, fmt.Errorf("%w: name the marketplace source", ErrBadTarget)
			}
			return "", verbArgs([]string{"plugin", "marketplace", "add"}, r.Source), nil
		},
		"list": func(Paths, MarketRequest) (string, []string, error) {
			return "", []string{"plugin", "marketplace", "list", "--json"}, nil
		},
		"update": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "marketplace", "update"}, r.Name), nil
		},
		"remove": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "marketplace", "remove"}, r.Name), nil
		},
	},
}

// --- Hermes -----------------------------------------------------------------

var hermes = &spec{
	cli:  "hermes",
	name: "Hermes Agent",
	bin:  binHermes,
	scopes: []Scope{
		{ID: "user", Label: "This machine"},
	},
	roster:         hermesRoster,
	available:      true,
	catalogInstall: true,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(_ Paths, t Target) (string, []string, error) {
			// --no-enable is the conservative half of the vendor's own prompt:
			// it installs the plugin without letting its code run, and the
			// pane's Enable action is where the user decides.
			return "", verbArgs([]string{"plugins", "install"}, t.Source, "--no-enable"), nil
		},
		VerbRemove: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugins", "remove"}, t.Name), nil
		},
		VerbEnable: func(_ Paths, t Target) (string, []string, error) {
			// No --allow-tool-override: that grants the plugin permission to
			// replace built-in tools, and it is the user's answer to give.
			return "", verbArgs([]string{"plugins", "enable"}, t.Name), nil
		},
		VerbDisable: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugins", "disable"}, t.Name), nil
		},
		VerbUpdate: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugins", "update"}, t.Name), nil
		},
	},
	notes: map[string]string{
		"marketplace-add": "Hermes has no marketplace sources: its plugins come from a curated catalog, a Git URL or an owner/repo.",
		"capabilities":    "Hermes grants capabilities per plugin. `hermes plugins capabilities` prints what is declared and what is granted; enabling a plugin that asks to replace built-in tools has to be confirmed in a terminal.",
	},
}

// --- OpenCode ---------------------------------------------------------------

var opencode = &spec{
	cli:  "opencode",
	name: "OpenCode",
	bin:  binOpencode,
	scopes: []Scope{
		{ID: "user", Label: "This machine", Note: "Global config: ~/.config/opencode/opencode.json and its plugins/ directory."},
		{ID: "project", Label: "This workspace", Note: "Project config: opencode.json and .opencode/plugins/."},
	},
	roster: opencodeRoster,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(p Paths, t Target) (string, []string, error) {
			if t.Scope == "user" {
				return "", verbArgs([]string{"plugin"}, t.Source, "-g"), nil
			}
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin"}, t.Source), err
		},
		VerbEnable: func(p Paths, t Target) (string, []string, error) {
			// A removed npm plugin comes back through the same vendor command.
			if t.Scope == "user" {
				return "", verbArgs([]string{"plugin"}, t.Source, "-g"), nil
			}
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin"}, t.Source), err
		},
	},
	// OpenCode exposes no remove command: its plugin list is its own config
	// array, and that one splice is PiCode's only plugin-list write (ADR-0167).
	writer: opencodeRemove,
	notes: map[string]string{
		"update":      "OpenCode installs npm plugins at startup with Bun, so the next start already uses the version its config names.",
		"marketplace": "OpenCode has no plugin marketplace: plugins are npm modules named in opencode.json. Its ecosystem page lists community plugins.",
		"local":       "Files in a plugins/ directory load by their presence: OpenCode reads them at startup and has no command to turn one off.",
	},
}

// --- Muse Code --------------------------------------------------------------

var muse = &spec{
	cli:  "muse",
	name: "Muse Code",
	bin:  binMuse,
	scopes: []Scope{
		{ID: "user", Label: "This machine"},
		{ID: "project", Label: "This workspace"},
	},
	roster:         museRoster,
	available:      true,
	catalogInstall: true,
	// Measured 2026-09-21: an installed plugin keeps reading
	// `status: "available"` in Muse's catalog, so the join is what keeps the
	// pane from offering Install for it.
	catalogNeedsRoster: true,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			// A marketplace ref (name@marketplace) carries no scope; a local
			// bundle does.
			if strings.Contains(t.Source, "@") && !strings.HasPrefix(t.Source, "npm:") {
				return dir, verbArgs([]string{"plugins", "install"}, t.Source, "--json"), err
			}
			return dir, verbArgs([]string{"plugins", "install"}, t.Source, "--scope", scopeOf(t.Scope), "--json"), err
		},
		VerbRemove: func(p Paths, t Target) (string, []string, error) {
			// --delete-data is never passed: it destroys the plugin's stored
			// data, and that decision is the user's.
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugins", "remove"}, t.Name, "--json"), err
		},
		VerbEnable: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugins", "enable"}, t.Name, "--json"), err
		},
		VerbDisable: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugins", "disable"}, t.Name, "--json"), err
		},
		VerbUpdate: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugins", "update"}, t.Name, "--json"), err
		},
		VerbInspect: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugins", "inspect"}, t.Name, "--json"), nil
		},
	},
	market: map[string]func(Paths, MarketRequest) (string, []string, error){
		"add": func(p Paths, r MarketRequest) (string, []string, error) {
			if clean(r.Name) == "" || clean(r.Source) == "" {
				return "", nil, fmt.Errorf("%w: a marketplace needs a name and a source", ErrBadTarget)
			}
			dir, err := dirForScope(p, r.Ref)
			return dir, []string{"plugins", "marketplace", "add", clean(r.Name), clean(r.Source), "--json"}, err
		},
		"list": func(Paths, MarketRequest) (string, []string, error) {
			return "", []string{"plugins", "marketplace", "list", "--json"}, nil
		},
		"update": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugins", "marketplace", "update"}, r.Name, "--json"), nil
		},
		"remove": func(_ Paths, r MarketRequest) (string, []string, error) {
			return "", verbArgs([]string{"plugins", "marketplace", "remove"}, r.Name, "--json"), nil
		},
	},
	notes: map[string]string{
		"approve": "Muse gates a plugin's runtime capabilities: `muse plugins inspect <id>` lists what it asks for, and `muse plugins approve <id>` trusts it. PiCode shows both commands instead of answering for you.",
	},
}

// --- Antigravity ------------------------------------------------------------

var agy = &spec{
	cli:  "agy",
	name: "Antigravity",
	bin:  binAgy,
	scopes: []Scope{
		{ID: "user", Label: "This machine"},
	},
	roster: agyRoster,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "install"}, t.Source), nil
		},
		VerbRemove: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "uninstall"}, t.Name), nil
		},
		VerbEnable: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "enable"}, t.Name), nil
		},
		VerbDisable: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "disable"}, t.Name), nil
		},
	},
	notes: map[string]string{
		"update":          "Antigravity exposes no plugin update command.",
		"marketplace-add": "Antigravity has no marketplace sources here: `agy plugin link <marketplace> <target>` prints a link and `agy plugin import` copies plugins from Gemini or Claude.",
	},
}

// --- Omp --------------------------------------------------------------------

var omp = &spec{
	cli:  "omp",
	name: "Omp",
	bin:  binOmp,
	scopes: []Scope{
		{ID: "user", Label: "This machine"},
		{ID: "project", Label: "This workspace", Note: "Omp applies a project scope to marketplace installs; a local path or npm package installs for the machine."},
	},
	roster:    ompRoster,
	available: true,
	// Measured 2026-09-21: `omp plugin discover` lists what the configured
	// marketplaces offer as `<name>@<version>` plus a description line, and
	// never says which source provides it; `omp plugin install <name>` without
	// a marketplace resolves through npm instead (it 404s on the registry). So
	// the catalog is offered as information, and only the installed list — plus
	// the catalog join, since discover keeps listing an installed plugin — is
	// acted on.
	catalogInstall:     false,
	catalogNeedsRoster: true,
	argv: map[Verb]func(Paths, Target) (string, []string, error){
		VerbInstall: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			args := verbArgs([]string{"plugin", "install"}, t.Source, "--json")
			if t.Scope == "project" {
				args = append(args, "--scope=project")
			}
			return dir, args, err
		},
		VerbRemove: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "uninstall"}, t.Name, "--json"), err
		},
		VerbEnable: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "enable"}, t.Name, "--json"), err
		},
		VerbDisable: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "disable"}, t.Name, "--json"), err
		},
		VerbUpdate: func(p Paths, t Target) (string, []string, error) {
			dir, err := dirForScope(p, t.Scope)
			return dir, verbArgs([]string{"plugin", "upgrade"}, t.Name, "--json"), err
		},
		VerbInspect: func(_ Paths, t Target) (string, []string, error) {
			return "", verbArgs([]string{"plugin", "doctor"}, t.Name, "--json"), nil
		},
	},
	market: map[string]func(Paths, MarketRequest) (string, []string, error){
		"add": func(p Paths, r MarketRequest) (string, []string, error) {
			if clean(r.Source) == "" {
				return "", nil, fmt.Errorf("%w: name the marketplace source", ErrBadTarget)
			}
			dir, err := dirForScope(p, r.Ref)
			return dir, verbArgs([]string{"plugin", "marketplace", "add"}, r.Source, "--json"), err
		},
		"list": func(Paths, MarketRequest) (string, []string, error) {
			return "", []string{"plugin", "marketplace", "list", "--json"}, nil
		},
		"update": func(p Paths, r MarketRequest) (string, []string, error) {
			dir, err := dirForScope(p, r.Ref)
			return dir, verbArgs([]string{"plugin", "marketplace", "update"}, r.Name, "--json"), err
		},
		"remove": func(p Paths, r MarketRequest) (string, []string, error) {
			dir, err := dirForScope(p, r.Ref)
			return dir, verbArgs([]string{"plugin", "marketplace", "remove"}, r.Name, "--json"), err
		},
	},
}

// scopeOf normalizes a scope for a vendor flag.
func scopeOf(scope string) string {
	if clean(scope) == "" {
		return "user"
	}
	return clean(scope)
}

// rosters reads the vendor's own list for the CLIs whose command takes the
// include-available flag; each parser lives in roster.go.
func claudeRoster(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	args := []string{"plugin", "list", "--json"}
	if available {
		args = append(args, "--available")
	}
	out, err := runVendor(ctx, binClaude, "", args...)
	if err != nil {
		return nil, "", err
	}
	return parseClaude(out, scope, available)
}

func codexRoster(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	args := []string{"plugin", "list", "--json"}
	if available {
		args = append(args, "--available")
	}
	out, err := runVendor(ctx, binCodex, "", args...)
	if err != nil {
		return nil, "", err
	}
	return parseCodex(out, available)
}

func grokRoster(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	args := []string{"plugin", "list", "--json"}
	if available {
		args = append(args, "--available")
	}
	out, err := runVendor(ctx, binGrok, "", args...)
	if err != nil {
		return nil, "", err
	}
	return parseVendorRows("grok", out, available)
}

func hermesRoster(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	if available {
		out, err := runVendor(ctx, binHermes, "", "plugins", "search", "--json")
		if err != nil {
			return nil, "", err
		}
		return parseHermes(out, true)
	}
	out, err := runVendor(ctx, binHermes, "", "plugins", "list", "--json")
	if err != nil {
		return nil, "", err
	}
	return parseHermes(out, false)
}

func museRoster(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	args := []string{"plugins", "list", "--json"}
	if available {
		args = append(args, "--available")
	}
	out, err := runVendor(ctx, binMuse, "", args...)
	if err != nil {
		return nil, "", err
	}
	return parseMuse(out, available)
}

func agyRoster(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	out, err := runVendor(ctx, binAgy, "", "plugin", "list")
	if err != nil {
		return nil, "", err
	}
	return parseAgy(out)
}

func ompRoster(ctx context.Context, p Paths, scope string, available bool) ([]Row, string, error) {
	if available {
		// Measured 2026-09-21: `omp plugin discover` is the catalog and
		// `--json` changes nothing about it (it prints the same prose).
		out, err := runVendor(ctx, binOmp, "", "plugin", "discover")
		if err != nil {
			return nil, "", err
		}
		return parseOmpDiscover(out)
	}
	out, err := runVendor(ctx, binOmp, "", "plugin", "list", "--json")
	if err != nil {
		return nil, "", err
	}
	return parseVendorRows("omp", out, available)
}
