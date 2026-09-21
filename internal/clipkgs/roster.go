package clipkgs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// The roster parsers. Each one reads the vendor's own output, and each one
// refuses loudly rather than answering with a list it did not recognize: an
// empty roster that came from a shape change reads as "nothing is installed",
// which is the one wrong answer this pane must never give (ADR-0167).

// parseClaude reads `claude plugin list --json`, observed 2026-09-20:
//
//	[{id, version, scope, enabled, installPath, projectPath?, installedAt, lastUpdated}]
//
// where id is "name@marketplace" and scope is user | project | local | synced.
// `--available` answers a different envelope with two halves (installed and
// the marketplace catalog) and its own row shape — see
// claudeAvailableEnvelope; an unrecognized catalog shape falls through to the
// tolerant reader.
func parseClaude(out, scope string, available bool) ([]Row, string, error) {
	var raw []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &raw); err != nil {
		if rows, ok := claudeAvailableEnvelope(out, scope, available); ok {
			return rows, "", nil
		}
		return parseVendorRows("claude-code", out, available)
	}
	rows := []Row{}
	for _, m := range raw {
		row := claudeRow(m)
		if row.ID == "" {
			continue
		}
		if !available && !claudeRowInScope(row.Scope, scope) {
			continue
		}
		if available && row.Scope == "" {
			// A catalog row is not an installed one unless the vendor says so.
			row.Installed = truthy(m["installed"])
		}
		if row.Scope == "synced" {
			row.Note = "Synced from claude.ai; PiCode lists it and cannot change it here."
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 && len(raw) > 0 && !available {
		// Every row belonged to another scope: that is an empty scope, not an
		// unread list.
		return rows, "", nil
	}
	return rows, "", nil
}

// claudeAvailableEnvelope reads the shape `claude plugin list --json
// --available` really prints (measured 2026-09-20, Claude Code on this
// machine): the two halves are *different* row shapes, not the installed
// array with extra fields.
//
//	{"installed":[{id,version,scope,enabled,installPath,installedAt,…}],
//	 "available":[{pluginId,name,description,marketplaceName,version,source}]}
//
// A catalog row is not installed and has no scope or enabled flag; its
// `source` is where the plugin sits inside that marketplace, so the row's
// origin is the marketplace name and its installable spec is the id. Reading
// the envelope through the tolerant mapper instead flattened both halves into
// one list, marked installed plugins as not installed, and lost the scope.
func claudeAvailableEnvelope(out, scope string, available bool) ([]Row, bool) {
	var envelope struct {
		Installed []map[string]any `json:"installed"`
		Available []map[string]any `json:"available"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
		return nil, false
	}
	if envelope.Installed == nil && envelope.Available == nil {
		return nil, false
	}
	rows := []Row{}
	for _, m := range envelope.Installed {
		row := claudeRow(m)
		if row.ID == "" || !claudeRowInScope(row.Scope, scope) {
			continue
		}
		if row.Scope == "synced" {
			row.Note = "Synced from claude.ai; PiCode lists it and cannot change it here."
		}
		rows = append(rows, row)
	}
	if available {
		for _, m := range envelope.Available {
			id := firstText(m["pluginId"], m["id"])
			if id == "" {
				continue
			}
			name, marketplace := splitAtMarketplace(id)
			rows = append(rows, Row{
				ID:          id,
				Name:        firstText(m["name"], name),
				Version:     text(m["version"]),
				Description: text(m["description"]),
				Scope:       scope,
				Marketplace: firstText(m["marketplaceName"], marketplace),
				Source:      id,
				SourceKind:  "marketplace",
			})
		}
	}
	return rows, true
}

func claudeRow(m map[string]any) Row {
	id := text(m["id"])
	name, marketplace := splitAtMarketplace(id)
	row := Row{
		ID:          id,
		Name:        name,
		Version:     text(m["version"]),
		Scope:       text(m["scope"]),
		Enabled:     truthy(m["enabled"]),
		Marketplace: marketplace,
		InstallPath: firstText(m["installPath"], m["projectPath"]),
		Installed:   true,
		Source:      id,
		SourceKind:  "marketplace",
	}
	if v, ok := m["installed"]; ok {
		row.Installed = truthy(v)
	}
	if marketplace == "synced" {
		row.SourceKind = "synced"
	}
	return row
}

// claudeRowInScope answers whether a row belongs in the requested scope's
// view. Claude's machine-level `synced` plugins are shown with the machine
// scope, which is the only place PiCode could ever act on them anyway.
func claudeRowInScope(rowScope, want string) bool {
	if rowScope == want {
		return true
	}
	return want == "user" && rowScope == "synced"
}

// parseCodex reads `codex plugin list --json`, observed 2026-09-20:
//
//	{"installed":[{"pluginId","name","marketplaceName","version","installed",
//	               "enabled","source":{"source","id"},"installPolicy","authPolicy"}]}
//
// with `--available` adding the uninstalled marketplace entries. Keys are
// read by name; an envelope PiCode does not recognize is not turned into an
// empty marketplace.
func parseCodex(out string, available bool) ([]Row, string, error) {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil, "", fmt.Errorf("%w: codex printed nothing", ErrRosterShape)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return parseVendorRows("codex", out, available)
	}
	keys := make([]string, 0, len(envelope))
	for k := range envelope {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := []Row{}
	note := ""
	for _, k := range keys {
		var objs []map[string]any
		if err := json.Unmarshal(envelope[k], &objs); err != nil {
			continue
		}
		for _, o := range objs {
			row := Row{
				ID:          firstText(o["pluginId"], o["id"], o["name"]),
				Name:        text(o["name"]),
				Version:     text(o["version"]),
				Enabled:     truthy(o["enabled"]),
				Installed:   truthy(o["installed"]),
				Marketplace: firstText(o["marketplaceName"], o["marketplace"]),
				Scope:       "user",
				SourceKind:  "marketplace",
			}
			if row.Name == "" {
				row.Name = row.ID
			}
			if src, ok := o["source"].(map[string]any); ok {
				// Measured 2026-09-20: {"source":"local","path":"/<root>/plugins/<name>"}
				// — `source` is the kind, `path` is what the plugin came from.
				// Reading the kind as the source showed "local" as a plugin's
				// origin, which is not a location at all.
				row.Source = firstText(src["path"], src["url"], src["id"], src["source"])
				if kind := text(src["source"]); kind != "" {
					row.SourceKind = kind
				}
			}
			if row.Source == "" {
				row.Source = row.ID
			}
			policy := []string{}
			for _, key := range []string{"installPolicy", "authPolicy"} {
				if v := text(o[key]); v != "" && v != "NONE" {
					policy = append(policy, strings.ToLower(strings.ReplaceAll(key, "Policy", ""))+" "+v)
				}
			}
			row.Status = strings.Join(policy, " · ")
			rows = append(rows, row)
		}
		// Measured 2026-09-20: the envelope carries `available` (empty) even for
		// a plain `plugin list --json` read, so the key alone cannot mean the
		// caller asked for the catalog. The note belongs to the marketplace
		// read only; on an installed-roster read it read as a network warning
		// about a catalog nobody requested.
		if k == "available" && available {
			note = "Codex resolves this catalog over the network; an offline or signed-out read fails rather than reporting an empty marketplace."
		}
	}
	return rows, note, nil
}

// parseHermes reads `hermes plugins list --json` (observed 2026-09-20:
// [{name,status,version,description,source,removed}]) and
// `hermes plugins search --json` for the curated catalog. `status` is the
// vendor's own word and is what the pane shows; enabled is derived from it
// rather than guessed.
func parseHermes(out string, available bool) ([]Row, string, error) {
	objs, err := vendorObjects(out)
	if err != nil {
		return nil, "", err
	}
	rows := make([]Row, 0, len(objs))
	for _, o := range objs {
		name := firstText(o["name"], o["id"])
		if name == "" {
			continue
		}
		status := text(o["status"])
		source := text(o["source"])
		// The curated catalog names each plugin's repository — the spec
		// `hermes plugins install` takes — and prints no `source` word at all
		// (measured 2026-09-20). Without reading it, a catalog row showed its
		// own name as its origin, which is not something anyone can install
		// from.
		repo := firstText(o["repo"], o["url"])
		row := Row{
			ID:          name,
			Name:        name,
			Version:     text(o["version"]),
			Description: firstText(o["description"], o["summary"]),
			Status:      status,
			Source:      firstText(repo, name),
			SourceKind:  firstText(source, kindOfSource(repo, "")),
			Scope:       "user",
			Enabled:     hermesEnabled(status),
			Installed:   !available,
		}
		if source == "bundled" {
			row.Note = "Ships with Hermes; installing is enabling it."
		}
		rows = append(rows, row)
	}
	note := ""
	if available {
		note = "Hermes' curated catalog; a Git URL or owner/repo installs anything not listed here."
	}
	return rows, note, nil
}

// hermesEnabled reads the vendor's status word. "not enabled" is the state
// this machine showed for bundled plugins, and it contains "enabled" — the
// naive substring test would report the opposite.
func hermesEnabled(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "not ") || strings.HasPrefix(s, "un") || strings.HasPrefix(s, "dis") {
		return false
	}
	switch {
	case strings.Contains(s, "enabled"), strings.Contains(s, "active"), strings.Contains(s, "loaded"), strings.Contains(s, "installed"):
		return true
	}
	return false
}

// parseAgy reads `agy plugin list`. The CLI has no --json flag, and its
// subcommands read a leading flag as the plugin name (measured: `agy plugin
// uninstall --help` really tried to uninstall a plugin called `--help`), but
// its two output shapes are both machine-text: a JSON envelope once anything
// is imported, and the sentence "No imported plugins." while nothing is
// (measured 2026-09-20 on Antigravity 1.2.7). Anything else is read line by
// line, and a line PiCode cannot read is kept as the row's status rather than
// dropped.
func parseAgy(out string) ([]Row, string, error) {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" || isNoneLine(trimmed) {
		return []Row{}, "", nil
	}
	if rows, ok, err := agyImports(trimmed); ok {
		return rows, agyNote, err
	}
	rows := []Row{}
	for _, line := range strings.Split(trimmed, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		name := strings.Trim(fields[0], "*•-")
		if name == "" || strings.HasSuffix(name, ":") {
			continue
		}
		row := Row{ID: name, Name: name, Scope: "user", Installed: true, SourceKind: "marketplace"}
		rest := fields[1:]
		for _, f := range rest {
			switch {
			case row.Version == "" && looksLikeVersion(f):
				row.Version = f
			case strings.EqualFold(f, "enabled"), strings.EqualFold(f, "disabled"):
				row.Status = f
				row.Enabled = strings.EqualFold(f, "enabled")
			}
		}
		if row.Status == "" {
			row.Status = strings.Join(rest, " ")
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, "", fmt.Errorf("%w: agy printed %s", ErrRosterShape, firstLine(trimmed))
	}
	return rows, agyNote, nil
}

// agyNote is the one line the pane shows about an agy roster read: the CLI has
// no machine-readable flag, so the shape is the vendor's text either way.
const agyNote = "Antigravity's plugin list has no machine-readable flag: PiCode reads the CLI's own output, both the envelope it prints with plugins and its one-sentence empty answer."

// agyImports reads the envelope `agy plugin list` prints once anything is
// imported (measured 2026-09-20, Antigravity 1.2.7):
//
//	{"imports":[{name,source,importedAt,components}]}
//
// `source` is where the plugin came from ("claude-code", "antigravity"), which
// is the origin the row shows; the row is installed and live, since the only
// way into this list is a successful install or import. An envelope whose rows
// carry no name is refused rather than reported as an empty roster. ok=false
// means "not this shape", and the caller falls back to the line reader.
func agyImports(out string) ([]Row, bool, error) {
	var envelope struct {
		Imports []map[string]any `json:"imports"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil || envelope.Imports == nil {
		return nil, false, nil
	}
	rows := make([]Row, 0, len(envelope.Imports))
	for _, m := range envelope.Imports {
		name := firstText(m["name"], m["id"], m["plugin"])
		if name == "" {
			continue
		}
		rows = append(rows, Row{
			ID:         name,
			Name:       name,
			Scope:      "user",
			Enabled:    true,
			Installed:  true,
			Source:     text(m["source"]),
			SourceKind: "import",
		})
	}
	if len(rows) == 0 && len(envelope.Imports) > 0 {
		return nil, true, fmt.Errorf("%w: agy listed %d import(s) with no name PiCode can read", ErrRosterShape, len(envelope.Imports))
	}
	return rows, true, nil
}

// parseMarketplaces reads a `<cli> plugin marketplace list --json` into rows
// the pane's Marketplace tab can show: the vendor's JSON where there is any,
// its plain-text source list where the flag is accepted but ignored (Omp), and
// its one-sentence "none" answer. A source is a name plus where it came from,
// nothing more.
func parseMarketplaces(out string) ([]Row, error) {
	objs, err := vendorObjects(out)
	if err != nil {
		if isNoneLine(out) {
			return []Row{}, nil
		}
		if rows, ok := proseMarketplaces(out); ok {
			return rows, nil
		}
		return nil, err
	}
	rows := make([]Row, 0, len(objs))
	for _, o := range objs {
		name := firstText(o["name"], o["id"], o["marketplace"], o["marketplaceName"])
		if name == "" {
			continue
		}
		source := ""
		if sm, ok := o["source"].(map[string]any); ok {
			source = firstText(sm["url"], sm["root"], sm["path"], sm["source"], sm["type"])
		}
		// A vendor prints a bare `source` as the *kind* of source (Claude Code:
		// "github" or "directory") and the location under its own key, so the
		// location candidates come first: reading `source` before them showed
		// "github" as where an anthropics marketplace lives (measured
		// 2026-09-20 against the real `claude plugin marketplace list --json`).
		source = firstText(source, o["url"], o["repo"], o["root"], o["path"], o["source"], o["installLocation"], o["location"])
		rows = append(rows, Row{
			ID:         name,
			Name:       name,
			Scope:      "user",
			Source:     source,
			SourceKind: "marketplace",
			Enabled:    true,
			Installed:  true,
			Status:     firstText(o["version"], o["ref"]),
		})
	}
	return rows, nil
}

// proseMarketplaces reads a marketplace list a vendor printed as lines. Omp
// takes `--json` on `plugin marketplace list` and prints a "Configured
// Marketplaces:" block of "name  location" lines anyway (measured 2026-09-20
// on 18.2.6), so the flag alone cannot be trusted to produce JSON. A block
// that does not yield one name-and-location line is refused and the caller
// reports the vendor's output unread.
func proseMarketplaces(out string) ([]Row, bool) {
	rows := []Row{}
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasSuffix(line, ":") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, false
		}
		name := strings.TrimSuffix(fields[0], ":")
		location := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line[len(fields[0]):]), ":"))
		// A source is a path, a URL or an owner/repo; without one of those
		// this is someone's prose, not a marketplace list.
		if name == "" || !strings.ContainsAny(location, "/@") {
			return nil, false
		}
		rows = append(rows, Row{
			ID:         name,
			Name:       name,
			Scope:      "user",
			Source:     location,
			SourceKind: "marketplace",
			Enabled:    true,
			Installed:  true,
		})
	}
	if len(rows) == 0 {
		return nil, false
	}
	return rows, true
}

// parseVendorRows is the tolerant reader behind Grok and Omp. Both shapes are
// measured now — Grok's installed and available rows (status/name/version/
// path/source/marketplace), and Omp's {npm:[…], marketplace:[…]} envelope,
// whose marketplace rows nest their version and install path in `entries` —
// and each is pinned by a fixture in testdata and by a live subtest. Muse is
// the one reader left unmeasured: the installed build refuses every plugin
// verb ("plugins are not available in this build"), so nothing about its
// non-empty shape could be observed. It accepts an array of objects or an
// object holding one or more arrays of objects, maps the field names by
// candidate, and refuses anything it cannot recognize: a wrong guess fails
// instead of reporting an empty list.
func parseVendorRows(cli, out string, available bool) ([]Row, string, error) {
	objs, err := vendorObjects(out)
	if err != nil {
		return nil, "", err
	}
	rows := make([]Row, 0, len(objs))
	for _, o := range objs {
		row, ok := mapVendorRow(o)
		if !ok {
			continue
		}
		if available {
			// Grok prints no `installed` boolean: its rows carry a status word
			// ("installed" / "available") instead, so reading the missing key as
			// false reported a plugin the vendor lists as installed as not
			// installed in the marketplace view (measured 2026-09-20).
			if v, ok := o["installed"]; ok && v != nil {
				row.Installed = truthy(v)
			} else {
				row.Installed = row.Enabled || hermesEnabled(row.Status)
			}
		} else {
			row.Installed = true
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 && len(objs) > 0 {
		return nil, "", fmt.Errorf("%w: %s rows carry no name PiCode can read", ErrRosterShape, cli)
	}
	return rows, "", nil
}

// vendorObjects decodes either an array of objects or an object whose values
// are arrays of objects, in a stable order. A prose answer PiCode recognizes
// as "there are none" is an empty list; any other prose is refused.
func vendorObjects(out string) ([]map[string]any, error) {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: the CLI printed nothing", ErrRosterShape)
	}
	if err := json.Unmarshal([]byte(trimmed), &[]map[string]any{}); err == nil {
		var arr []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrRosterShape, firstLine(trimmed))
		}
		return arr, nil
	}
	if isNoneLine(trimmed) {
		return nil, nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRosterShape, firstLine(trimmed))
	}
	keys := make([]string, 0, len(envelope))
	for k := range envelope {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out2 := []map[string]any{}
	for _, k := range keys {
		var arr []map[string]any
		if err := json.Unmarshal(envelope[k], &arr); err != nil {
			continue
		}
		out2 = append(out2, arr...)
	}
	return out2, nil
}

// mapVendorRow maps one vendor object onto a row, by the field names the
// vendors use for the same facts. A row without a name is skipped by the
// caller; a whole answer without one readable row is refused.
func mapVendorRow(o map[string]any) (Row, bool) {
	id := firstText(o["id"], o["pluginId"], o["plugin_id"], o["name"], o["plugin"], o["slug"], o["package"])
	if id == "" {
		return Row{}, false
	}
	// Omp publishes a marketplace-installed plugin as one row carrying the id
	// and scope, with the version and install path on the first of its
	// `entries` elements (measured 2026-09-20 on 18.2.6). Reading only the row
	// lost the version and path and reported an installed plugin as disabled —
	// there is no `enabled` key on such a row at all.
	entry, nested := firstEntry(o)
	name := firstText(o["name"], o["displayName"], o["title"])
	marketplace := firstText(o["marketplace"], o["marketplaceName"], o["marketplace_name"])
	if marketplace == "" {
		if n, mp := splitAtMarketplace(id); mp != "" {
			name, marketplace = firstText(name, n), mp
		}
	}
	row := Row{
		ID:          id,
		Name:        firstText(name, id),
		Version:     firstText(o["version"], o["latestVersion"], o["latest"], o["release"], entry["version"]),
		Description: firstText(o["description"], o["summary"]),
		Scope:       firstText(o["scope"], o["layer"], "user"),
		InstallPath: firstText(o["installPath"], o["path"], o["dir"], o["root"], o["location"], entry["installPath"]),
		Marketplace: marketplace,
		Status:      text(o["status"]),
	}
	if v, ok := o["enabled"]; ok && v != nil {
		row.Enabled = truthy(v)
	} else {
		row.Enabled = nested || hermesEnabled(row.Status)
	}
	source := o["source"]
	row.Source, row.SourceKind = sourceOf(source, id, row.Marketplace)
	if row.Source == "" {
		row.Source = id
	}
	return row, true
}

// firstEntry returns the first element of a vendor's nested `entries` array.
// Omp nests a marketplace plugin's scope, version and install path there while
// the id and scope sit on the row itself.
func firstEntry(o map[string]any) (map[string]any, bool) {
	arr, ok := o["entries"].([]any)
	if !ok || len(arr) == 0 {
		return nil, false
	}
	first, ok := arr[0].(map[string]any)
	return first, ok
}

// sourceOf reads the many shapes a vendor prints a plugin's origin in: a
// bare string, or an object with a kind and a location.
func sourceOf(v any, fallbackID, marketplace string) (string, string) {
	switch s := v.(type) {
	case string:
		return s, kindOfSource(s, marketplace)
	case map[string]any:
		kind := firstText(s["source"], s["type"], s["kind"])
		loc := firstText(s["id"], s["url"], s["path"], s["root"], s["location"], s["package"], fallbackID)
		if kind == "" {
			kind = kindOfSource(loc, marketplace)
		}
		return loc, kind
	}
	return "", kindOfSource("", marketplace)
}

func kindOfSource(source, marketplace string) string {
	switch {
	case marketplace != "":
		return "marketplace"
	case strings.HasPrefix(source, "npm:"):
		return "npm"
	case strings.Contains(source, "://"), strings.HasPrefix(source, "git@"):
		return "git"
	case strings.HasPrefix(source, "/"), strings.HasPrefix(source, "./"), strings.HasPrefix(source, "~"):
		return "path"
	case source == "":
		return ""
	}
	return "marketplace"
}

// splitAtMarketplace splits "name@marketplace". A scoped npm package keeps
// its own scope (@scope/name) — only a trailing @ separates a marketplace.
func splitAtMarketplace(id string) (string, string) {
	if i := strings.LastIndex(id, "@"); i > 0 {
		return id[:i], id[i+1:]
	}
	return id, ""
}

func isNoneLine(s string) bool {
	first := strings.ToLower(firstLine(s))
	for _, phrase := range []string{"no plugins", "no imported plugins", "no marketplaces", "none installed", "nothing installed"} {
		if strings.Contains(first, phrase) {
			return true
		}
	}
	return false
}

func looksLikeVersion(s string) bool {
	if s == "" || (s[0] < '0' || s[0] > '9') {
		return false
	}
	return strings.Contains(s, ".")
}

func text(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func firstText(vs ...any) string {
	for _, v := range vs {
		if s := text(v); s != "" {
			return s
		}
	}
	return ""
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return hermesEnabled(t)
	}
	return false
}
