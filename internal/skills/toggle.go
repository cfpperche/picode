package skills

// Per-skill switches (ADR-0196 slice 3). Each CLI that has one keeps it in its
// own words; PiCode reads the switch into the report and writes the vendor's
// key, never a PiCode setting. Measured 2026-09-24 in a throwaway HOME against
// each vendor's own roster (docs/plans/skills.md, "Per-CLI declaration"):
//
//	claude-code  skillOverrides.<name> = "off"   user settings.json / workspace settings.local.json
//	codex        [[skills.config]] path=<SKILL.md> enabled=false   user config.toml only
//	opencode     permission.skill.<name> = "deny" user opencode.json / workspace opencode.json
//	omp          skills.ignoredSkills (globs)    user config.yml / workspace .omp/config.yml (replaces the user list)
//	grok         [skills] disabled = [names]     user config.toml only
//	hermes       skills.disabled = [names]       user config.yaml only
//	muse         `muse skills disable|enable`    the CLI writes its own settings.json
//
// A write is read back through the same reader; when another layer still
// decides the other way, the result says which file does.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisettings"
)

// Switch kinds, one per vendor mechanism.
const (
	SwitchClaude   = "claude-overrides"
	SwitchCodex    = "codex-config"
	SwitchOpenCode = "opencode-permission"
	SwitchOmp      = "omp-ignored"
	SwitchGrok     = "grok-disabled"
	SwitchHermes   = "hermes-disabled"
	SwitchMuse     = "muse-verb"
)

// ErrNoSwitch is a CLI (or a row) with no per-skill switch.
var ErrNoSwitch = errors.New("this CLI has no per-skill switch")

// ErrStale is a settings file that changed between the read and the write.
var ErrStale = clisettings.ErrStale

// ToggleReq is one switch flip.
type ToggleReq struct {
	CLI       string
	Home      string
	Workspace string
	Row       Row
	Enabled   bool
	// Muse runs the vendor's verb with HOME set to Home; tests replace it.
	Muse func(ctx context.Context, home string, args ...string) error
}

// ToggleResult is the state after the write, read back.
type ToggleResult struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	File    string `json:"file,omitempty"` // the file PiCode wrote
	Note    string `json:"note,omitempty"` // another layer still decides, or the switch is machine-wide
}

// switchable says whether a row carries a switch: a folder skill the CLI
// would load (or would load once trusted), not a shadowed copy, an invalid
// folder or one of an agent's own skills.
func switchable(r Row) bool {
	if r.Scope == Agent {
		return false
	}
	switch r.Status {
	case StatusLoaded, StatusDisabled, StatusNeedsTrust, StatusIfTrusted:
		return true
	}
	return false
}

// applySwitches reads each row's switch into Enabled, and marks a skill the
// CLI would load but whose switch is off as disabled.
func applySwitches(spec Spec, homeDir, workspace string, rows []Row) {
	if spec.Switch == "" {
		return
	}
	for i := range rows {
		if !switchable(rows[i]) {
			continue
		}
		on, _, err := switchState(spec.Switch, homeDir, workspace, rows[i])
		if err != nil {
			continue // an unreadable file: no switch, the Settings pane says why
		}
		v := on
		rows[i].Enabled = &v
		if !on && rows[i].Status == StatusLoaded {
			rows[i].Status = StatusDisabled
		}
	}
}

// switchState is a row's effective switch and the file that decides it ("" when
// nothing sets it and the CLI's default, on, holds).
func switchState(kind, homeDir, workspace string, r Row) (bool, string, error) {
	switch kind {
	case SwitchClaude:
		for _, f := range claudeFiles(homeDir, workspace) {
			d, err := clisettings.OpenDoc(f, clisettings.FormatJSON)
			if err != nil {
				return true, "", err
			}
			if v, ok := d.Scalar("skillOverrides", r.Name); ok {
				return v != "off", f, nil
			}
		}
		return true, "", nil
	case SwitchCodex:
		f := codexFile(homeDir)
		d, err := clisettings.OpenDoc(f, clisettings.FormatTOML)
		if err != nil {
			return true, "", err
		}
		on, set := true, false
		for _, e := range d.ArrayTables("skills.config") {
			if codexEntryFor(e, r) {
				if b, ok := e["enabled"].(bool); ok {
					on, set = b, true
				}
			}
		}
		if set {
			return on, f, nil
		}
		return true, "", nil
	case SwitchOpenCode:
		// OpenCode deep-merges the user file, then every project file from
		// the repository root down to the workspace: a later file overrides
		// a key in place, and a new key joins at the end. The last matching
		// rule of that merged order wins.
		type rule struct{ action, file string }
		order, rules := []string{}, map[string]rule{}
		for _, f := range openCodeFiles(homeDir, workspace) {
			d, err := clisettings.OpenDoc(f, clisettings.FormatJSONC)
			if err != nil {
				return true, "", err
			}
			if v, ok := d.Scalar("permission", "skill"); ok {
				if s, isStr := v.(string); isStr {
					order, rules = []string{"*"}, map[string]rule{"*": {s, f}}
					continue
				}
			}
			for _, pattern := range d.Keys("permission", "skill") {
				v, _ := d.Scalar("permission", "skill", pattern)
				action, _ := v.(string)
				if _, seen := rules[pattern]; !seen {
					order = append(order, pattern)
				}
				rules[pattern] = rule{action, f}
			}
		}
		on, file := true, ""
		for _, pattern := range order {
			if globMatch(pattern, r.Name) {
				on, file = rules[pattern].action != "deny", rules[pattern].file
			}
		}
		return on, file, nil
	case SwitchOmp:
		list, f, err := ompEffective(homeDir, workspace)
		if err != nil {
			return true, "", err
		}
		for _, g := range list {
			if globMatch(g, r.Name) {
				return false, f, nil
			}
		}
		return true, "", nil
	case SwitchGrok, SwitchHermes:
		f, format, key := nameListFile(kind, homeDir)
		d, err := clisettings.OpenDoc(f, format)
		if err != nil {
			return true, "", err
		}
		list, _, err := d.Strings(key...)
		if err != nil {
			return true, "", err
		}
		for _, n := range list {
			if n == r.Name {
				return false, f, nil
			}
		}
		return true, "", nil
	case SwitchMuse:
		f := museFile(homeDir)
		d, err := clisettings.OpenDoc(f, clisettings.FormatJSON)
		if err != nil {
			return true, "", err
		}
		scope, key, ok := museKey(homeDir, workspace, r)
		if !ok {
			return true, "", nil
		}
		var v any
		var set bool
		if scope == "user" {
			v, set = d.Scalar("skills", "activation", "user", key)
		} else {
			v, set = d.Scalar("skills", "activation", "projects", workspace, key)
		}
		if set {
			return v != "off", f, nil
		}
		return true, "", nil
	}
	return true, "", ErrNoSwitch
}

// SetEnabled flips one row's switch in the CLI's own file and reads it back.
func SetEnabled(ctx context.Context, q ToggleReq) (ToggleResult, error) {
	spec, ok := For(q.CLI)
	if !ok {
		return ToggleResult{}, ErrUnknownCLI
	}
	if spec.Switch == "" || !switchable(q.Row) {
		return ToggleResult{}, ErrNoSwitch
	}
	if q.Home == "" {
		q.Home, _ = os.UserHomeDir()
	}
	file, err := writeSwitch(ctx, spec.Switch, q)
	if err != nil {
		return ToggleResult{}, err
	}
	on, decider, err := switchState(spec.Switch, q.Home, q.Workspace, q.Row)
	if err != nil {
		return ToggleResult{}, err
	}
	res := ToggleResult{Name: q.Row.Name, Enabled: on, File: file}
	switch {
	case on != q.Enabled && decider != "":
		res.Note = fmt.Sprintf("%s still decides %s: its rule wins over the one PiCode wrote", decider, q.Row.Name)
	case on != q.Enabled:
		res.Note = fmt.Sprintf("%s still reads %s as %s after the change", spec.CLI, q.Row.Name, map[bool]string{true: "on", false: "off"}[on])
	case spec.SwitchMachineOnly && q.Row.Scope == Workspace:
		res.Note = spec.SwitchNote
	}
	return res, nil
}

func writeSwitch(ctx context.Context, kind string, q ToggleReq) (string, error) {
	r := q.Row
	switch kind {
	case SwitchClaude:
		// The vendor's own /skills menu writes the workspace's local file;
		// the machine switch is the user file.
		f := filepath.Join(q.Home, ".claude", "settings.json")
		if r.Scope == Workspace {
			f = filepath.Join(q.Workspace, ".claude", "settings.local.json")
		}
		return f, editDoc(f, clisettings.FormatJSON, func(d *clisettings.Doc) error {
			if q.Enabled {
				return d.Remove("skillOverrides", r.Name)
			}
			return d.SetScalar([]string{"skillOverrides", r.Name}, "off")
		})
	case SwitchCodex:
		f := codexFile(q.Home)
		return f, editDoc(f, clisettings.FormatTOML, func(d *clisettings.Doc) error {
			// Turning on also drops a name-only entry for this skill: it is
			// the rule the person asked PiCode to lift.
			if _, err := d.RemoveArrayTables("skills.config", func(e map[string]any) bool { return codexEntryFor(e, r) }); err != nil {
				return err
			}
			if q.Enabled {
				return nil
			}
			return d.AppendArrayTable("skills.config", [][2]any{{"path", skillFile(r)}, {"enabled", false}})
		})
	case SwitchOpenCode:
		f := openCodeUserFile(q.Home)
		if r.Scope == Workspace {
			f = openCodeProjectFile(q.Workspace)
		}
		return f, editDoc(f, clisettings.FormatJSONC, func(d *clisettings.Doc) error {
			if v, ok := d.Scalar("permission", "skill"); ok {
				if _, isStr := v.(string); isStr {
					return fmt.Errorf("permission.skill in %s is one rule for every skill; edit it there", f)
				}
			}
			if q.Enabled {
				return d.Remove("permission", "skill", r.Name)
			}
			return d.SetScalar([]string{"permission", "skill", r.Name}, "deny")
		})
	case SwitchOmp:
		f := filepath.Join(q.Home, ".omp", "agent", "config.yml")
		var base []string
		if r.Scope == Workspace {
			f = filepath.Join(q.Workspace, ".omp", "config.yml")
			// A workspace list replaces the machine list in Omp, so the
			// first one starts from what the machine already turns off.
			user, err := clisettings.OpenDoc(filepath.Join(q.Home, ".omp", "agent", "config.yml"), clisettings.FormatYAML)
			if err != nil {
				return "", err
			}
			base, _, _ = user.Strings("skills", "ignoredSkills")
		}
		return f, editDoc(f, clisettings.FormatYAML, func(d *clisettings.Doc) error {
			list, found, err := d.Strings("skills", "ignoredSkills")
			if err != nil {
				return err
			}
			if !found {
				list = append([]string{}, base...)
			}
			next := withName(list, r.Name, !q.Enabled)
			// An empty workspace list still matters when the machine list is
			// not empty: it is how Omp turns those skills back on here.
			if len(next) == 0 && len(base) == 0 {
				return d.Remove("skills", "ignoredSkills")
			}
			return d.SetStrings([]string{"skills", "ignoredSkills"}, next)
		})
	case SwitchGrok, SwitchHermes:
		f, format, key := nameListFile(kind, q.Home)
		return f, editDoc(f, format, func(d *clisettings.Doc) error {
			list, _, err := d.Strings(key...)
			if err != nil {
				return err
			}
			next := withName(list, r.Name, !q.Enabled)
			if len(next) == 0 {
				return d.Remove(key...)
			}
			return d.SetStrings(key, next)
		})
	case SwitchMuse:
		verb := "disable"
		if q.Enabled {
			verb = "enable"
		}
		scope, _, ok := museKey(q.Home, q.Workspace, r)
		if !ok {
			return "", ErrNoSwitch
		}
		args := []string{"skills", verb, skillFile(r), "--scope", scope, "--json"}
		if scope == "project" {
			args = append(args, "--workspace", q.Workspace)
		}
		run := q.Muse
		if run == nil {
			run = runMuse
		}
		return museFile(q.Home), run(ctx, q.Home, args...)
	}
	return "", ErrNoSwitch
}

// editDoc opens, edits and saves one file against the revision it read.
func editDoc(file string, f clisettings.Format, edit func(*clisettings.Doc) error) error {
	d, err := clisettings.OpenDoc(file, f)
	if err != nil {
		return err
	}
	rev, before := d.Revision(), d.Text()
	if err := edit(&d); err != nil {
		return err
	}
	if string(d.Text()) == string(before) {
		return nil // nothing to write: a no-op never touches the file
	}
	if !d.Exists() {
		// No revision guards a file that was not there: refuse if the CLI's
		// own menu created it meanwhile rather than overwrite it.
		if _, err := os.Stat(file); err == nil {
			return ErrStale
		}
	}
	if d.Empty() {
		// Nothing but the braces PiCode's own first write created: the file
		// goes, so turning a skill back on leaves the folder as it was.
		if !d.Exists() {
			return nil
		}
		now, err := clisettings.OpenDoc(file, f)
		if err != nil {
			return err
		}
		if now.Revision() != rev {
			return ErrStale
		}
		return os.Remove(file)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return d.Save(rev)
}

// withName adds or removes one exact name, keeping every other entry (and a
// glob that also matches) as the user wrote it.
func withName(list []string, name string, present bool) []string {
	out := make([]string, 0, len(list)+1)
	has := false
	for _, n := range list {
		if n == name {
			if !present {
				continue
			}
			has = true
		}
		out = append(out, n)
	}
	if present && !has {
		out = append(out, name)
	}
	return out
}

// globMatch is the vendors' name pattern: `*` and `?` over a skill name. A
// pattern path.Match cannot parse matches only itself.
func globMatch(pattern, name string) bool {
	if ok, err := path.Match(pattern, name); err == nil {
		return ok
	}
	return pattern == name
}

func skillFile(r Row) string { return filepath.Join(r.Dir, "SKILL.md") }

// codexEntryFor says whether a [[skills.config]] element addresses this row:
// by its SKILL.md path or by its name — never both, which Codex ignores.
func codexEntryFor(e map[string]any, r Row) bool {
	path, hasPath := e["path"]
	name, hasName := e["name"]
	switch {
	case hasPath && hasName:
		return false
	case hasPath:
		return path == skillFile(r)
	case hasName:
		return name == r.Name
	}
	return false
}

func claudeFiles(homeDir, workspace string) []string {
	var out []string
	if workspace != "" {
		out = append(out, filepath.Join(workspace, ".claude", "settings.local.json"), filepath.Join(workspace, ".claude", "settings.json"))
	}
	return append(out, filepath.Join(homeDir, ".claude", "settings.json"))
}

func codexFile(homeDir string) string { return filepath.Join(homeDir, ".codex", "config.toml") }

// openCodeUserFile prefers the .jsonc the user already keeps.
func openCodeUserFile(homeDir string) string {
	dir := filepath.Join(homeDir, ".config", "opencode")
	if _, err := os.Stat(filepath.Join(dir, "opencode.jsonc")); err == nil {
		return filepath.Join(dir, "opencode.jsonc")
	}
	return filepath.Join(dir, "opencode.json")
}

// openCodeFiles is every file OpenCode merges, in merge order: the user's,
// then each project folder's from the repository root down to the workspace.
func openCodeFiles(homeDir, workspace string) []string {
	out := []string{openCodeUserFile(homeDir)}
	if workspace == "" {
		return out
	}
	dirs := walkUp(workspace)
	for i := len(dirs) - 1; i >= 0; i-- {
		for _, rel := range []string{"opencode.jsonc", "opencode.json", filepath.Join(".opencode", "opencode.jsonc"), filepath.Join(".opencode", "opencode.json")} {
			if _, err := os.Stat(filepath.Join(dirs[i], rel)); err == nil {
				out = append(out, filepath.Join(dirs[i], rel))
			}
		}
	}
	return out
}

func openCodeProjectFile(workspace string) string {
	if workspace == "" {
		return ""
	}
	for _, rel := range []string{"opencode.jsonc", "opencode.json", filepath.Join(".opencode", "opencode.jsonc"), filepath.Join(".opencode", "opencode.json")} {
		if _, err := os.Stat(filepath.Join(workspace, rel)); err == nil {
			return filepath.Join(workspace, rel)
		}
	}
	return filepath.Join(workspace, "opencode.json")
}

// ompEffective is the list Omp applies: the workspace's when it sets one, the
// machine's otherwise (measured: the project array replaces the user array).
func ompEffective(homeDir, workspace string) ([]string, string, error) {
	files := []string{}
	if workspace != "" {
		files = append(files, filepath.Join(workspace, ".omp", "config.yml"))
	}
	files = append(files, filepath.Join(homeDir, ".omp", "agent", "config.yml"))
	for _, f := range files {
		d, err := clisettings.OpenDoc(f, clisettings.FormatYAML)
		if err != nil {
			return nil, "", err
		}
		list, found, err := d.Strings("skills", "ignoredSkills")
		if err != nil {
			return nil, "", err
		}
		if found {
			return list, f, nil
		}
	}
	return nil, "", nil
}

func nameListFile(kind, homeDir string) (string, clisettings.Format, []string) {
	if kind == SwitchGrok {
		return filepath.Join(homeDir, ".grok", "config.toml"), clisettings.FormatTOML, []string{"skills", "disabled"}
	}
	return filepath.Join(homeDir, ".hermes", "config.yaml"), clisettings.FormatYAML, []string{"skills", "disabled"}
}

// museFile follows XDG_CONFIG_HOME for the real home only; a test home is its
// own world.
func museFile(homeDir string) string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		if real, _ := os.UserHomeDir(); real == homeDir {
			return filepath.Join(x, "muse", "settings.json")
		}
	}
	return filepath.Join(homeDir, ".config", "muse", "settings.json")
}

// museKey is how Muse names a skill in its settings: `$HOME/…/SKILL.md` for a
// machine skill, workspace-relative under projects.<workspace> for a
// workspace skill.
func museKey(homeDir, workspace string, r Row) (scope, key string, ok bool) {
	file := skillFile(r)
	if r.Scope == Workspace {
		rel, err := filepath.Rel(workspace, file)
		if err != nil || workspace == "" || strings.HasPrefix(rel, "..") {
			return "", "", false
		}
		return "project", filepath.ToSlash(rel), true
	}
	rel, err := filepath.Rel(homeDir, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "", false
	}
	return "user", "$HOME/" + filepath.ToSlash(rel), true
}

// runMuse runs Muse's own verb; its settings file is Muse's to write. HOME is
// the request's, so the file Muse writes is the one the reader reads.
func runMuse(ctx context.Context, home string, args ...string) error {
	bin, err := exec.LookPath("muse")
	if err != nil {
		return errors.New("Muse is not installed on this machine")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		var msg struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(out, &msg) == nil && msg.Error != "" {
			return errors.New("muse: " + msg.Error)
		}
		return fmt.Errorf("muse skills %s: %s", args[1], strings.TrimSpace(string(out)))
	}
	return nil
}
