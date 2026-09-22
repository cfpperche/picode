package clipkgs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Omp's own extension list (packages-unification slice 5). Omp loads
// extensions from its own settings, not from its plugin store, so the roster
// command alone leaves them invisible. The project layer is
// `<ws>/.omp/settings.json` — JSON, read and written here with the standard
// library — and the user layer is `~/.omp/agent/config.yml`, which PiCode never
// parses: it asks the CLI for that one. Measured 2026-09-21 on 18.2.8:
//
//   - `omp plugin list --json` never names a configured extension (the plugin
//     store and the settings list are two different things);
//   - `omp config get <key> --json` answers
//     `{"key":…,"value":[…],"type":"array","description":""}` for the layer the
//     working directory resolves to: run inside a project that declares
//     `extensions` it answers *that project's* array — the user's is replaced,
//     not merged — and run where no project layer exists it answers the user's;
//   - `omp config set <key> '<json array>'` writes `~/.omp/agent/config.yml`
//     wherever it runs, and refuses a project form (`--scope` is an unknown
//     option);
//   - `omp install <path>` links a *package* — it fails on a single `.ts` file
//     (ENOTDIR opening `<file>/package.json`) — so nothing in the CLI's own
//     verb set adds or removes a configured extension.
//
// A configured entry resolves against the working directory (the vendor's
// loader calls resolvePath(entry, cwd)) and its CLI id is
// `extension-module:<name>`, the name being the path's base without its
// extension (`index.ts`/`index.js` → its directory). That id is what
// `disabledExtensions` names.

// ompExtensionDisabledPrefix is the kind prefix the CLI's own ids carry
// (`makeExtensionId("extension-module", name)`).
const ompExtensionDisabledPrefix = "extension-module:"

// ompProjectSettings is the workspace layer PiCode reads and writes itself.
// Anywhere but a workspace there is none: the settings file belongs to a
// project, and the CLI applies it only to a session rooted there.
func ompProjectSettings(p Paths) string {
	if p.Cwd == "" {
		return ""
	}
	return filepath.Join(p.Cwd, ".omp", "settings.json")
}

// ompExtensionName is the CLI's own name for one configured extension: the
// path's base without its extension, and the directory for an `index.ts` (the
// vendor's getExtensionNameFromPath, mirrored so `disabledExtensions` and a row
// agree).
func ompExtensionName(entry string) string {
	entry = strings.ReplaceAll(strings.TrimSpace(entry), "\\", "/")
	base := entry
	if i := strings.LastIndex(entry, "/"); i >= 0 {
		base = entry[i+1:]
	}
	if base == "index.ts" || base == "index.js" {
		rest := strings.TrimSuffix(entry, "/"+base)
		if i := strings.LastIndex(rest, "/"); i >= 0 {
			return rest[i+1:]
		}
		if rest != "" {
			return rest
		}
		return base
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		return base[:i]
	}
	return base
}

// ompExtensionDisabledID is the id one configured extension carries in
// `disabledExtensions`.
func ompExtensionDisabledID(entry string) string {
	return ompExtensionDisabledPrefix + ompExtensionName(entry)
}

// ompExtensionDisabled reports whether one layer's `disabledExtensions` names
// this entry. The comparison is the CLI's own: the id, never the raw path —
// that is what its dashboard writes and its loader reads.
func ompExtensionDisabled(disabled []string, entry string) bool {
	id := ompExtensionDisabledID(entry)
	for _, d := range disabled {
		if strings.TrimSpace(d) == id {
			return true
		}
	}
	return false
}

// ompStringArray decodes one settings value: a list of strings, or nothing for
// a missing or null one. A value PiCode cannot read is refused by name — the
// file's shape is the vendor's, and guessing it would render rows the CLI never
// loads.
func ompStringArray(raw json.RawMessage, file, key string) ([]string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("%w: %s: %s is not a list of strings", ErrRosterShape, file, key)
	}
	out := make([]string, 0, len(arr))
	for _, s := range arr {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// ompProjectExtensions reads the workspace layer. A missing file is no
// extensions at all; a file that does not parse is an error naming it, never an
// empty list (ADR-0150's rule for a malformed user file, applied to a read).
func ompProjectExtensions(p Paths) (extensions, disabled []string, err error) {
	file := ompProjectSettings(p)
	if file == "" {
		return nil, nil, nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if strings.TrimSpace(string(b)) == "" {
		return nil, nil, nil
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, nil, fmt.Errorf("%w: %s is not valid JSON: %v", ErrRosterShape, file, err)
	}
	extensions, err = ompStringArray(doc["extensions"], file, "extensions")
	if err != nil {
		return nil, nil, err
	}
	disabled, err = ompStringArray(doc["disabledExtensions"], file, "disabledExtensions")
	if err != nil {
		return nil, nil, err
	}
	return extensions, disabled, nil
}

// parseOmpConfigArray reads one `omp config get <key> --json` answer:
//
//	{"key":"extensions","value":["a.ts"],"type":"array","description":""}
//
// measured 2026-09-21 on 18.2.8. Anything else is refused, so a shape PiCode
// does not recognize never reads as "no extensions".
func parseOmpConfigArray(out string) ([]string, error) {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: %s printed nothing", ErrRosterShape, binOmp)
	}
	var doc struct {
		Value []string `json:"value"`
	}
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return nil, fmt.Errorf("%w: %s config get answered %s", ErrRosterShape, binOmp, firstLine(trimmed))
	}
	out2 := make([]string, 0, len(doc.Value))
	for _, s := range doc.Value {
		if s = strings.TrimSpace(s); s != "" {
			out2 = append(out2, s)
		}
	}
	return out2, nil
}

// ompUserExtensions asks the CLI for the layer the given directory resolves to,
// which is how the user level is read without parsing the CLI's YAML
// config.yml: a read with no workspace asks in the user's own directory and
// gets the user's array.
func ompUserExtensions(ctx context.Context, p Paths, dir string) (extensions, disabled []string, err error) {
	if dir == "" {
		dir = p.home()
	}
	out, err := runVendor(ctx, binOmp, dir, "config", "get", "extensions", "--json")
	if err != nil {
		return nil, nil, err
	}
	if extensions, err = parseOmpConfigArray(out); err != nil {
		return nil, nil, err
	}
	out, err = runVendor(ctx, binOmp, dir, "config", "get", "disabledExtensions", "--json")
	if err != nil {
		return nil, nil, err
	}
	if disabled, err = parseOmpConfigArray(out); err != nil {
		return nil, nil, err
	}
	return extensions, disabled, nil
}

// ompExistingPath resolves one configured entry the way the CLI does — against
// the working directory — and answers it only when something is really there,
// so no row claims a path that does not exist.
func ompExistingPath(entry, base string) string {
	cand := entry
	if !filepath.IsAbs(cand) {
		if base == "" {
			return ""
		}
		cand = filepath.Join(base, cand)
	}
	if _, err := os.Stat(cand); err != nil {
		return ""
	}
	return cand
}

// ompExtensionRows builds one layer's rows. Every entry a layer declares is a
// row: the CLI loads it, and that settings file is the only place that says so.
func ompExtensionRows(entries, disabled []string, layer, base string) []Row {
	note := "Omp loads this extension from your user settings; removing the row runs the CLI's own config command."
	if layer == "project" {
		note = "Omp loads this extension from this workspace's .omp/settings.json; removing the row edits that file."
	}
	rows := make([]Row, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, Row{
			ID:          entry,
			Name:        ompExtensionName(entry),
			Source:      entry,
			SourceKind:  "extension",
			Scope:       layer,
			Enabled:     !ompExtensionDisabled(disabled, entry),
			Installed:   true,
			InstallPath: ompExistingPath(entry, base),
			Note:        note,
		})
	}
	return rows
}

// mergeOmpExtensions appends the extension rows the CLI's plugin roster does
// not already carry. The same package reached through both readers is one row,
// and the vendor's own row is the one that has the plugin verbs.
func mergeOmpExtensions(roster, extensions []Row) []Row {
	if len(extensions) == 0 {
		return roster
	}
	known := map[string]bool{}
	for _, r := range roster {
		for _, k := range []string{r.ID, r.Name, r.Source, r.InstallPath} {
			if k = ompIdentity(k); k != "" {
				known[k] = true
			}
		}
	}
	out := make([]Row, 0, len(roster)+len(extensions))
	out = append(out, roster...)
	for _, r := range extensions {
		if known[ompIdentity(r.ID)] || known[ompIdentity(r.Source)] || known[ompIdentity(r.InstallPath)] {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ompIdentity normalizes a row's identity for the overlap comparison: a path is
// compared as a path, anything else verbatim.
func ompIdentity(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, `/\`) {
		return filepath.Clean(s)
	}
	return s
}

// ompExtensions reads both layers for one read. The plugin roster is what the
// caller came for, so a user layer PiCode cannot read is reported in the read's
// note instead of failing the whole list — a malformed project file is not: it
// is the CLI's own store for this workspace, and refuses by name.
func ompExtensions(ctx context.Context, p Paths) ([]Row, string, error) {
	project, disabled, err := ompProjectExtensions(p)
	if err != nil {
		return nil, "", err
	}
	dir := p.Cwd
	if dir == "" {
		dir = p.home()
	}
	user, userDisabled, err := ompUserExtensions(ctx, p, dir)
	if err != nil {
		return ompExtensionRows(project, disabled, "project", p.Cwd),
			"Omp's user extensions could not be read: " + err.Error(), nil
	}
	rows := ompExtensionRows(project, disabled, "project", p.Cwd)
	// The two layers can name the same entry. The workspace row is the one that
	// stays: it is the layer PiCode reads and writes, and the layer the CLI's
	// own command reports for a session rooted here.
	named := map[string]bool{}
	for _, e := range project {
		named[ompIdentity(e)] = true
	}
	for _, r := range ompExtensionRows(user, userDisabled, "user", dir) {
		if !named[ompIdentity(r.Source)] {
			rows = append(rows, r)
		}
	}
	return rows, "", nil
}

// Removal is one entry's removal as a driver needs it: the vendor command that
// performs it, and — where the CLI has no verb at all — the fact that the
// engine has already performed it itself. InProcess is that fact: the workspace
// layer is a file PiCode writes, so Dir, Args and Line are empty and the caller
// hands nothing to a lane.
type Removal struct {
	InProcess bool
	Dir       string
	Args      []string
	Line      string
}

// OmpExtensionRemove removes one entry of Omp's own `extensions` list: from the
// workspace's `.omp/settings.json` when that file names it, and from the user
// layer through the CLI's own command otherwise. ok is false when neither layer
// names the target, which is the driver's signal that this is an ordinary
// plugin.
func OmpExtensionRemove(ctx context.Context, p Paths, t Target) (Removal, bool, error) {
	source, name := clean(t.Source), clean(t.Name)
	if source == "" && name == "" {
		return Removal{}, false, nil
	}
	names := func(entries []string) (string, bool) {
		for _, e := range entries {
			if e == source || e == name || ompExtensionName(e) == name {
				return e, true
			}
		}
		return "", false
	}

	file := ompProjectSettings(p)
	project, disabled, err := ompProjectExtensions(p)
	if err != nil {
		return Removal{}, true, err
	}
	if file != "" {
		if hit, ok := names(project); ok {
			if err := ompRemoveProjectEntry(file, hit, disabled); err != nil {
				return Removal{InProcess: true}, true, err
			}
			return Removal{InProcess: true}, true, nil
		}
	}

	dir := p.Cwd
	if dir == "" {
		dir = p.home()
	}
	user, _, err := ompUserExtensions(ctx, p, dir)
	if err != nil {
		return Removal{}, true, err
	}
	hit, ok := names(user)
	if !ok {
		return Removal{}, false, nil
	}
	rest := make([]string, 0, len(user))
	for _, e := range user {
		if e != hit {
			rest = append(rest, e)
		}
	}
	body, err := json.Marshal(rest)
	if err != nil {
		return Removal{}, true, err
	}
	args := []string{"config", "set", "extensions", string(body)}
	return Removal{Dir: dir, Args: args, Line: shellLine(dir, binOmp, args)}, true, nil
}

// ompRemoveProjectEntry takes one entry out of the workspace settings file: the
// entry itself from `extensions`, and its id from `disabledExtensions` so no
// disabled note outlives the extension it named. Every byte outside those two
// elements survives, and each edit is re-parsed and compared before the write.
func ompRemoveProjectEntry(file, entry string, disabled []string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	next, err := spliceArrayElement(b, "extensions", entry, "extension list")
	if err != nil {
		return err
	}
	if ompExtensionDisabled(disabled, entry) {
		cut, err := spliceArrayElement(next, "disabledExtensions", ompExtensionDisabledID(entry), "disabled list")
		switch {
		case err == nil:
			next = cut
		case !errors.Is(err, errNoElement):
			return err
		}
	}
	return writeAtomic(file, next)
}
