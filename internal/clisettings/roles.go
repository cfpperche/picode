package clisettings

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Model roles for a CLI that keeps a role → model-selector map in its own
// settings file (ADR-0181; omp is the first and only declaration today).
//
// A role row is a **scalar at a dynamic path** inside that map, so it rides the
// same parser, byte-span splice, revision check and atomic write every declared
// field has used since ADR-0163 — the only thing that is new is that the set of
// paths comes from the vendor's catalog plus the file, instead of a fixed list.
// A fallback chain and the quick-switch cycle are **lists of strings**, the one
// shape ADR-0174's key-map engine already writes through these same primitives.
//
// What this file deliberately does not do is judge a model selector. omp's own
// grammar is wider than it looks — `openrouter/z-ai/glm-4.7@cerebras:high` is
// one model id with a routing suffix and a thinking suffix, and only the first
// slash splits the provider (`pi-tui/src/overlays/model-selector.ts`,
// `parseModelString`/`splitUpstreamRouting`, read from the installed 18.2.8).
// A validator that rejected what it did not recognise would refuse values the
// CLI accepts, so PiCode checks only what its own writer must guarantee and
// leaves the meaning to omp.

// RoleDef is one role the CLI itself defines: the vendor's id, the tag its own
// UI prints, and the name it gives the role. Transcribed from the installed
// bundle and pinned by a test, the way every vendor fact in this package is.
type RoleDef struct {
	ID      string `json:"id"`
	Tag     string `json:"tag"`
	Name    string `json:"name"`
	Section string `json:"section"`
	Help    string `json:"help,omitempty"`
}

// rolesSpec is one CLI's declaration. Every path is the vendor's own.
type rolesSpec struct {
	// path is the map of role id → selector (omp: `modelRoles`).
	path []string
	// catalog is the vendor's built-in roles, in the vendor's own order.
	catalog []RoleDef
	// group names the section the role rows render in.
	group string
	// tagsPath is the map a user names custom roles in (omp: `modelTags`),
	// read so a custom role is listed even when nothing assigns it a model.
	tagsPath []string
	// cyclePath is the ordered list of roles the CLI's quick-switch cycles
	// through (omp: `cycleOrder`, the `⟳ N` badge in its own model hub).
	cyclePath  []string
	cycleLabel string
	cycleHelp  string
	// chainsPath is the map of role, model or provider → ordered fallbacks
	// (omp: `retry.fallbackChains`).
	chainsPath   []string
	chainGroup   string
	chainHelp    string
	chainAddHint string
	// levels are the thinking suffixes the CLI accepts after a selector.
	levels []string
	// pane is where the matrix is drawn. omp's own model hub keeps roles and
	// the model list on one screen, and so does PiCode (owner, 2026-09-22):
	// a role and the catalog it picks from answer one question.
	pane string
}

// report is the part of a role matrix that is not a row: what the pane needs to
// build a key that does not exist yet, and the vocabulary the CLI accepts.
func (rs *rolesSpec) report() *RolesReport {
	out := &RolesReport{
		Group:      rs.group,
		ChainGroup: rs.chainGroup,
		ChainHelp:  rs.chainHelp,
		ChainHint:  rs.chainAddHint,
		RolePrefix: strings.Join(rs.path, ".") + ".",
		TagPrefix:  strings.Join(rs.tagsPath, ".") + ".",
		Catalog:    rs.catalog,
		Levels:     rs.levels,
	}
	if len(rs.chainsPath) > 0 {
		out.ChainPrefix = strings.Join(rs.chainsPath, ".") + "."
	}
	if len(rs.cyclePath) > 0 {
		out.CyclePrefix = strings.Join(rs.cyclePath, ".")
	}
	return out
}

// roleIDPattern is the vendor's own rule for a role name, read from omp's model
// hub (`#submitRoleName`, `/^[a-zA-Z][\w-]*$/`). A name the CLI would refuse is
// not a row PiCode creates.
var roleIDPattern = regexp.MustCompile(`^[a-zA-Z][A-Za-z0-9_-]*$`)

// chainKeyPattern is narrower than the vendor's, on purpose. A chain is keyed by
// a role name, a `provider/model-id` or a `provider/*`, and those all fit here.
// What it excludes is a key this package's writer cannot address as one line —
// a colon (which is the key/value separator the YAML walker cuts on), a quote, a
// comment mark, or a YAML indicator in first position. Such a key is refused by
// name instead of written somewhere it does not belong.
var chainKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/*+-]*$`)

// roleFields builds the dynamic half of a report: one row per role, the cycle
// row, and one row per fallback chain the files already carry.
//
// The role order is the vendor's own (`getKnownRoleIds`, `src/config/
// model-roles.ts`): built-ins first, then any role named by the cycle, then by
// an assignment, then by a tag. Map keys are sorted before they are appended, so
// two reads of one file always produce the same report.
func (rs *rolesSpec) roleFields(docs []map[string]any) []Field {
	var out []Field
	seen := map[string]bool{}
	add := func(def RoleDef) {
		if seen[def.ID] {
			return
		}
		seen[def.ID] = true
		label := def.Name
		if label == "" {
			// A role the user invented has no vendor name; its id, capitalised,
			// reads like the built-ins beside it ("Review" next to "Judge").
			label = strings.ToUpper(def.ID[:1]) + def.ID[1:]
		}
		out = append(out, Field{
			Key:      strings.Join(rs.path, ".") + "." + def.ID,
			Path:     append(append([]string{}, rs.path...), def.ID),
			Label:    label,
			Tag:      def.Tag,
			Kind:     KindRole,
			Pane:     rs.pane,
			Group:    rs.group,
			Help:     def.Help,
			Fallback: "auto",
		})
	}
	for _, def := range rs.catalog {
		add(def)
	}
	// A role the user invented shows up in one of three places. The tag map is
	// where its display name lives, so it is read for the label too.
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		for _, id := range stringsAt(doc, rs.cyclePath) {
			if roleIDPattern.MatchString(id) {
				add(RoleDef{ID: id, Tag: strings.ToUpper(id)})
			}
		}
		for _, id := range sortedKeysAt(doc, rs.path) {
			if roleIDPattern.MatchString(id) {
				add(RoleDef{ID: id, Tag: strings.ToUpper(id)})
			}
		}
		for _, id := range sortedKeysAt(doc, rs.tagsPath) {
			if !roleIDPattern.MatchString(id) {
				continue
			}
			def := RoleDef{ID: id, Tag: strings.ToUpper(id)}
			if name, ok := lookup(doc, append(append([]string{}, rs.tagsPath...), id, "name")); ok {
				if s, ok := name.(string); ok && s != "" {
					def.Tag = s
				}
			}
			add(def)
		}
	}
	if len(rs.cyclePath) > 0 {
		out = append(out, Field{
			Key:   strings.Join(rs.cyclePath, "."),
			Path:  append([]string{}, rs.cyclePath...),
			Label: rs.cycleLabel,
			Kind:  KindList,
			Pane:  rs.pane,
			Group: rs.group,
			Help:  rs.cycleHelp,
		})
	}
	for _, key := range rs.chainKeys(docs) {
		out = append(out, Field{
			Key:   strings.Join(rs.chainsPath, ".") + "." + key,
			Path:  append(append([]string{}, rs.chainsPath...), key),
			Label: chainLabel(key),
			Kind:  KindList,
			Pane:  rs.pane,
			Group: rs.chainGroup,
		})
	}
	return out
}

// chainKeys is every fallback chain any layer sets, sorted so the report is
// stable. A chain nobody has written yet has no row until the user adds one.
func (rs *rolesSpec) chainKeys(docs []map[string]any) []string {
	if len(rs.chainsPath) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		for _, key := range sortedKeysAt(doc, rs.chainsPath) {
			if seen[key] || !chainKeyPattern.MatchString(key) {
				continue
			}
			seen[key] = true
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// synth builds the field for a key the report did not carry, so a role or a
// chain can be created. It is the one place that decides a key is writable: a
// name outside the vendor's own rule, or outside what this writer can address,
// is refused by name rather than written somewhere it does not belong.
func (rs *rolesSpec) synth(key string) (Field, bool, error) {
	rolePrefix := strings.Join(rs.path, ".") + "."
	if strings.HasPrefix(key, rolePrefix) {
		id := strings.TrimPrefix(key, rolePrefix)
		if !roleIDPattern.MatchString(id) {
			return Field{}, true, fmt.Errorf("%q is not a role name: a role starts with a letter and holds letters, digits, dashes and underscores", id)
		}
		return Field{
			Key:      key,
			Path:     append(append([]string{}, rs.path...), id),
			Label:    id,
			Tag:      strings.ToUpper(id),
			Kind:     KindRole,
			Pane:     rs.pane,
			Group:    rs.group,
			Fallback: "auto",
		}, true, nil
	}
	// A role declared by its tag alone — the vendor's own way to name a role
	// before assigning it a model (`getKnownRoleIds` reads `modelTags` keys),
	// and what "New role" writes so a fresh row is not a "Set here" over an
	// empty string. The tag's name is a scalar at a nested path; nothing else
	// under the tag is addressed.
	if len(rs.tagsPath) > 0 {
		tagPrefix := strings.Join(rs.tagsPath, ".") + "."
		if strings.HasPrefix(key, tagPrefix) && strings.HasSuffix(key, ".name") {
			id := strings.TrimSuffix(strings.TrimPrefix(key, tagPrefix), ".name")
			if !roleIDPattern.MatchString(id) {
				return Field{}, true, fmt.Errorf("%q is not a role name: a role starts with a letter and holds letters, digits, dashes and underscores", id)
			}
			return Field{
				Key:   key,
				Path:  append(append([]string{}, rs.tagsPath...), id, "name"),
				Label: id,
				Kind:  KindText,
				Pane:  rs.pane,
				Group: rs.group,
			}, true, nil
		}
	}
	if len(rs.cyclePath) > 0 && key == strings.Join(rs.cyclePath, ".") {
		return Field{
			Key:   key,
			Path:  append([]string{}, rs.cyclePath...),
			Label: rs.cycleLabel,
			Kind:  KindList,
			Pane:  rs.pane,
			Group: rs.group,
			Help:  rs.cycleHelp,
		}, true, nil
	}
	if len(rs.chainsPath) > 0 {
		chainPrefix := strings.Join(rs.chainsPath, ".") + "."
		if strings.HasPrefix(key, chainPrefix) {
			name := strings.TrimPrefix(key, chainPrefix)
			if !chainKeyPattern.MatchString(name) {
				return Field{}, true, fmt.Errorf("%q is not a fallback key PiCode writes: name a role, a provider/model-id, or a provider/*", name)
			}
			return Field{
				Key:   key,
				Path:  append(append([]string{}, rs.chainsPath...), name),
				Label: chainLabel(name),
				Kind:  KindList,
				Pane:  rs.pane,
				Group: rs.chainGroup,
			}, true, nil
		}
	}
	return Field{}, false, nil
}

// chainLabel says what a chain is for, in words: the key alone ("default",
// "openai/*") read as a bare word beside the role rows (visual review,
// 2026-09-22).
func chainLabel(key string) string {
	if strings.HasSuffix(key, "/*") {
		return "When any " + strings.TrimSuffix(key, "/*") + " model fails"
	}
	return "When " + key + " fails"
}

// stringsAt reads a list of strings at a path, ignoring any other shape: a
// report is never blocked by a key the user wrote as something else.
func stringsAt(doc map[string]any, path []string) []string {
	if len(path) == 0 {
		return nil
	}
	v, ok := lookup(doc, path)
	if !ok {
		return nil
	}
	out, err := asStrings(v)
	if err != nil {
		return nil
	}
	return out
}

// sortedKeysAt reads the keys of a map at a path in sorted order. Go randomises
// map iteration, and a report whose row order moved between two reads would
// scroll under the reader's cursor.
func sortedKeysAt(doc map[string]any, path []string) []string {
	if len(path) == 0 {
		return nil
	}
	v, ok := lookup(doc, path)
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
