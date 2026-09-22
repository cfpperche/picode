// Package clisettings manages agent CLIs' native settings files (ADR-0163).
// Each CLI owns its configuration; PiCode edits it in place. Reads use the
// real parser for the format; writes splice the exact bytes of one scalar, so
// comments, key order and every key PiCode does not know survive a save.
//
// The package has no schema of its own beyond what each CLI declares: a key
// nobody declared is never written, and a CLI without a declaration has no
// editor rather than a generic JSON box (ADR-0099 §5).
package clisettings

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Paths locates the files a request touches. Home empty → os.UserHomeDir.
// Cwd is the workspace folder; empty drops every project layer.
type Paths struct {
	Home string
	Cwd  string
}

func (p Paths) home() string {
	if p.Home != "" {
		return p.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

// Kind is the control a field renders as. Four kinds are scalars, where a write
// is one token and the surgical splice is trivially safe. Two are not, and they
// are the exception ADR-0181 makes to ADR-0163's "fields are scalars only":
//
//   - KindRole is still a scalar — a model selector — but at a path the
//     vendor's catalog and the file decide, not a fixed declaration.
//   - KindList is an ordered list of strings, written through the same
//     `listLiteral` splice ADR-0174's key-map engine has used since 2026-09-21.
//
// Nothing else is writable: a value that is a map, a block or anything the
// format cannot render on one line is reported and left alone.
type Kind string

const (
	KindBool   Kind = "bool"
	KindSelect Kind = "select"
	KindText   Kind = "text"
	KindNumber Kind = "number"
	KindRole   Kind = "role"
	KindList   Kind = "list"
)

// Option is one choice of a select field.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Field is one setting a CLI declares. Key is the dotted path into that CLI's
// own document — the name the vendor uses, never a PiCode alias, so the row
// and the file say the same thing.
type Field struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Kind     Kind     `json:"kind"`
	Options  []Option `json:"options,omitempty"`
	Help     string   `json:"help,omitempty"`
	Group    string   `json:"group,omitempty"`
	Fallback string   `json:"fallback,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	// Pane names the pane that draws this row when it is not Settings. The
	// row lives in the same report and the same file — one revision, one
	// writer — but a pane with its own surface for it (omp's Models pane for
	// `enabledModels` and `disabledProviders`) keeps it out of Settings, so a
	// key is never edited in two places that disagree about what it means.
	Pane string `json:"pane,omitempty"`
	// Tag is the short name the vendor's own UI prints for this row (omp's
	// DEFAULT, SMOL, SLOW). Empty for every field whose label is the name.
	Tag string `json:"tag,omitempty"`
	// Path addresses the value when the key cannot be split on dots — a
	// fallback chain is keyed by `openai/gpt-4.1-mini`, whose own name carries
	// one. It never leaves the server: the pane echoes Key, and the writer
	// rebuilds Path from the declaration that owns the key.
	Path []string `json:"-"`
	// Danger names the one value of this field that loosens a safety
	// boundary, so the row can carry a one-line warning beside it instead of
	// hiding the choice (benchmarks.md: danger named, not hidden).
	Danger string `json:"danger,omitempty"`
	// DangerNote is that value's own warning. Two rows in one group sharing a
	// generic sentence printed the same 17 words twice (visual review,
	// 2026-09-20), so each dangerous choice says what it specifically costs.
	DangerNote string `json:"dangerNote,omitempty"`
	// Secret marks a field whose value is credential-shaped and is reported
	// as set without reporting what it is.
	Secret bool `json:"secret,omitempty"`
	// DefaultOn is what the CLI does with a boolean nobody set. The pane
	// draws an unset switch at this value, so leaving a row alone shows the
	// truth rather than "off" (owner's call, 2026-09-20: one control for a
	// boolean in every pane, which means the switch has to carry the
	// effective value because it cannot show a third state).
	DefaultOn bool `json:"defaultOn,omitempty"`
}

func (f Field) path() []string {
	if len(f.Path) > 0 {
		return f.Path
	}
	return strings.Split(f.Key, ".")
}

func (f Field) allows(scope string) bool {
	if len(f.Scopes) == 0 {
		return true
	}
	for _, s := range f.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Layer is one file a CLI reads, with the values it sets. Values holds only
// the keys this layer actually sets, so the pane can mark a row as set here
// instead of reporting an inheritance that did not happen.
type Layer struct {
	Scope    string         `json:"scope"`
	Label    string         `json:"label"`
	Path     string         `json:"path"`
	Format   Format         `json:"format"`
	Exists   bool           `json:"exists"`
	Writable bool           `json:"writable"`
	Error    string         `json:"error,omitempty"`
	Values   map[string]any `json:"values"`
	Revision string         `json:"revision,omitempty"`
	// Unreadable names the rows this layer holds in a shape PiCode will not
	// rewrite — a chain written as a map, say. The row is dropped from Values
	// rather than drawn with a control that cannot save.
	Unreadable []string `json:"unreadable,omitempty"`
	// Note explains a layer that exists in the CLI but not in this request —
	// a workspace file with no workspace selected. Hiding it made a link to
	// that layer silently edit the machine file instead (found in live QA,
	// 2026-09-20), so the layer is reported and says what it needs.
	Note string `json:"note,omitempty"`
}

// Report is what the pane renders: the CLI's declared fields and one entry per
// file it reads, in the vendor's own precedence order (last wins).
type Report struct {
	CLI    string  `json:"cli"`
	Fields []Field `json:"fields"`
	Layers []Layer `json:"layers"`
	// Roles is present only for a CLI that keeps a model-role matrix. It is
	// what the pane needs to offer a row that does not exist yet: the key
	// prefixes to build, and the vocabulary the CLI accepts.
	Roles *RolesReport `json:"roles,omitempty"`
}

// RolesReport carries the parts of a role matrix that are not rows.
type RolesReport struct {
	Group       string    `json:"group"`
	ChainGroup  string    `json:"chainGroup"`
	ChainHelp   string    `json:"chainHelp,omitempty"`
	ChainHint   string    `json:"chainHint,omitempty"`
	RolePrefix  string    `json:"rolePrefix"`
	TagPrefix   string    `json:"tagPrefix,omitempty"`
	ChainPrefix string    `json:"chainPrefix,omitempty"`
	CyclePrefix string    `json:"cycleKey,omitempty"`
	Catalog     []RoleDef `json:"catalog"`
	// Levels are the thinking suffixes the CLI accepts after a selector.
	Levels []string `json:"levels,omitempty"`
}

// Patch is one save: values to set and keys to hand back to the parent layer.
// Revision is the layer's revision from the Report the editor was built from;
// a file that changed underneath is refused rather than overwritten.
type Patch struct {
	Scope    string         `json:"scope"`
	Set      map[string]any `json:"set,omitempty"`
	Reset    []string       `json:"reset,omitempty"`
	Revision string         `json:"revision,omitempty"`
	Force    bool           `json:"force,omitempty"`
}

// ErrStale is returned when the file changed since the report the editor holds.
var ErrStale = errors.New("this file changed on disk since it was read")

type layerSpec struct {
	scope  string
	label  string
	format Format
	// file resolves the path, or returns "" when this layer does not apply
	// (a CLI with no project file, or a request without a workspace).
	file func(Paths) string
}

type spec struct {
	id     string
	layers []layerSpec
	fields []Field
	// roles is the model-role matrix this CLI keeps, when it keeps one
	// (ADR-0181). Its rows are computed per request from the vendor's catalog
	// and the files, because a user's own role is a row too.
	roles *rolesSpec
}

// fieldFor resolves one key to the field that owns it: a declared row, or a row
// the roles declaration can create. A key nobody declares is refused by name.
func (s *spec) fieldFor(key string) (Field, error) {
	for _, f := range s.fields {
		if f.Key == key {
			return f, nil
		}
	}
	if s.roles != nil {
		if f, owned, err := s.roles.synth(key); owned {
			return f, err
		}
	}
	return Field{}, fmt.Errorf("%s is not a setting PiCode manages for %s", key, s.id)
}

// For returns the declaration for cli, or nil when PiCode has none. Pi is not
// here: it keeps its own editor, API and trust rules (ADR-0101).
func For(cli string) *spec {
	for i := range catalog {
		if catalog[i].id == cli {
			return &catalog[i]
		}
	}
	return nil
}

// Supported lists the CLIs with a declaration, in catalog order.
func Supported() []string {
	out := make([]string, 0, len(catalog))
	for _, s := range catalog {
		out = append(out, s.id)
	}
	return out
}

// Read reports every layer of one CLI. A file that does not exist is a layer
// with no values, not an error: the CLI is running on its own defaults.
func Read(cli string, p Paths) (Report, error) {
	s := For(cli)
	if s == nil {
		return Report{}, fmt.Errorf("PiCode has no settings schema for %q", cli)
	}
	rep := Report{CLI: cli}
	// Every layer is decoded first, because the rows a role matrix has are
	// decided by the files as much as by the vendor's catalog: a role only the
	// workspace assigns is still a row on the machine layer, showing what that
	// layer would inherit.
	docs := make([]map[string]any, 0, len(s.layers))
	for _, ls := range s.layers {
		path := ls.file(p)
		if path == "" {
			rep.Layers = append(rep.Layers, Layer{
				Scope: ls.scope, Label: ls.label, Format: ls.format,
				Values: map[string]any{}, Writable: false,
				Note: "Open this CLI from a workspace to edit its workspace settings.",
			})
			docs = append(docs, nil)
			continue
		}
		layer := Layer{Scope: ls.scope, Label: ls.label, Path: path, Format: ls.format, Values: map[string]any{}, Writable: true}
		raw, err := os.ReadFile(path)
		switch {
		case os.IsNotExist(err):
			rep.Layers = append(rep.Layers, layer)
			docs = append(docs, nil)
			continue
		case err != nil:
			layer.Error = err.Error()
			layer.Writable = false
			rep.Layers = append(rep.Layers, layer)
			docs = append(docs, nil)
			continue
		}
		layer.Exists = true
		layer.Revision = revision(raw)
		doc, err := decode(raw, ls.format, path)
		if err != nil {
			// A document the parser rejects is reported, never silently
			// replaced: the pane offers an explicit Replace (ADR-0099 §4).
			layer.Error = err.Error()
			layer.Writable = false
			rep.Layers = append(rep.Layers, layer)
			docs = append(docs, nil)
			continue
		}
		rep.Layers = append(rep.Layers, layer)
		docs = append(docs, doc)
	}

	rep.Fields = s.fields
	if s.roles != nil {
		rep.Fields = append(append([]Field{}, s.fields...), s.roles.roleFields(docs)...)
		rep.Roles = s.roles.report()
	}
	for i := range rep.Layers {
		doc := docs[i]
		if doc == nil {
			continue
		}
		for _, f := range rep.Fields {
			if !f.allows(rep.Layers[i].Scope) {
				continue
			}
			v, ok := lookup(doc, f.path())
			if !ok {
				continue
			}
			// A value the writer would not rewrite is reported as unreadable
			// and left out of the layer, so the pane never offers a control
			// over a shape PiCode refuses to touch (ADR-0174's rule, applied
			// here for the first time to a settings row).
			if f.Kind == KindList {
				if _, err := asStrings(v); err != nil {
					rep.Layers[i].Unreadable = append(rep.Layers[i].Unreadable, f.Key)
					continue
				}
			}
			rep.Layers[i].Values[f.Key] = redact(f, v)
		}
		sort.Strings(rep.Layers[i].Unreadable)
	}
	return rep, nil
}

// Apply writes one layer. It re-reads the file immediately before the write,
// refuses when it changed since the caller's report, and renames a complete
// temporary file into place so a reader never sees half a document.
func Apply(cli string, p Paths, patch Patch) error {
	s := For(cli)
	if s == nil {
		return fmt.Errorf("PiCode has no settings schema for %q", cli)
	}
	var ls *layerSpec
	for i := range s.layers {
		if s.layers[i].scope == patch.Scope {
			ls = &s.layers[i]
			break
		}
	}
	if ls == nil {
		return fmt.Errorf("%s has no %q settings file", cli, patch.Scope)
	}
	path := ls.file(p)
	if path == "" {
		return fmt.Errorf("%s has no %q settings file here", cli, patch.Scope)
	}
	byKey := map[string]Field{}
	for _, key := range append(keysOf(patch.Set), patch.Reset...) {
		f, err := s.fieldFor(key)
		if err != nil {
			return err
		}
		if !f.allows(ls.scope) {
			return fmt.Errorf("%s cannot be set in the %s layer", key, ls.scope)
		}
		byKey[key] = f
	}

	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		raw = nil
	}
	if patch.Revision != "" && !patch.Force && revision(raw) != patch.Revision {
		return ErrStale
	}
	if len(raw) > 0 {
		if _, err := decode(raw, ls.format, path); err != nil && !patch.Force {
			return err
		}
	}
	if ls.format == FormatTOML {
		// An array of tables cannot hold a declared key: writing into an
		// element would edit arbitrary user data, and creating a `[name]`
		// table beside `[[name]]` is a duplicate TOML refuses to parse.
		for _, key := range append(keysOf(patch.Set), patch.Reset...) {
			p := byKey[key].path()
			if len(p) > 1 && tomlArrayTable(raw, strings.Join(p[:len(p)-1], ".")) {
				return fmt.Errorf("%s is a list of %s entries in this file, which PiCode does not edit; change it in %s", strings.Join(p[:len(p)-1], "."), strings.Join(p[:len(p)-1], "."), path)
			}
		}
	}

	text := raw
	// Deterministic order so a multi-key save is reproducible in tests and in
	// a diff the owner reads.
	keys := make([]string, 0, len(patch.Set))
	for k := range patch.Set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		f := byKey[key]
		if f.Kind == KindList {
			// A list is the one value whose existing span can be a block, and a
			// block is not something one splice can address. The key-map engine
			// solved this in 2026-09-21; this is the same code path, so both
			// writers keep the same guarantees about a file with no final
			// newline and a value the user broke across lines.
			values, err := asStrings(patch.Set[key])
			if err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			if text, err = spliceList(text, ls.format, f.path(), values, path); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			continue
		}
		// A scalar this writer splices is one line by construction. A string
		// carrying a newline is not: YAML would render it as a block scalar,
		// which `valueSpan` then reports as absent, and the next save would
		// insert a second copy of the key beside it. Refused by name instead.
		if s, ok := patch.Set[key].(string); ok && strings.ContainsAny(s, "\r\n") {
			return fmt.Errorf("%s: a value PiCode writes here is one line; this one has a line break", key)
		}
		lit, err := literal(ls.format, patch.Set[key])
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if start, end, ok := valueSpan(text, ls.format, f.path()); ok {
			text = append(append(append([]byte{}, text[:start]...), lit...), text[end:]...)
			continue
		}
		if text, err = insertValue(text, ls.format, f.path(), lit); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	resets := append([]string{}, patch.Reset...)
	sort.Strings(resets)
	for _, key := range resets {
		if text, err = removeValue(text, ls.format, byKey[key].path()); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	if len(text) == len(raw) && string(text) == string(raw) {
		return nil
	}
	// The result must still parse. A splice that produced an unreadable file
	// is a bug in PiCode, and the file the CLI reads is not where it surfaces.
	if _, err := decode(text, ls.format, path); err != nil {
		return fmt.Errorf("refusing to write %s: the result would not parse (%w)", path, err)
	}
	return writeAtomic(path, text)
}

func writeAtomic(path string, text []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".picode-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(text); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func revision(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

// redact keeps secret-shaped values out of the API. A field a CLI uses for a
// token is declared Secret; its value is reported as set without reporting
// what it is.
func redact(f Field, v any) any {
	if !f.Secret {
		return v
	}
	if s, ok := v.(string); ok && s != "" {
		return "••••••"
	}
	return v
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
