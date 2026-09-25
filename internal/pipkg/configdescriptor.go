package pipkg

// Config descriptors (docs/plans/package-config-manifest.md; pulls forward
// the declarative end state ADR-0099 deferred). A descriptor says which file
// a package configures and which typed fields it holds, so the Packages view
// can offer a Configure form without a bespoke editor per package. The
// descriptor is data, not code: the files stay the only source of truth, and
// every ADR-0099 guarantee (unknown keys preserved, atomic write, 409 on an
// unparseable file, scoped reset) is enforced by the server on top of it.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ConfigField is one typed input of a descriptor-driven config.
type ConfigField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // string | enum | boolean | number | secret
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"` // enum choices
	Min      *float64 `json:"min,omitempty"`     // number bounds
	Max      *float64 `json:"max,omitempty"`
	Help     string   `json:"help,omitempty"`
}

// ConfigFile is one file a descriptor writes. Agent paths are relative to
// UserDir (~/.pi/agent); workspace paths to the workspace folder.
type ConfigFile struct {
	Scope  string `json:"scope"`  // agent | workspace
	Path   string `json:"path"`   // relative to the scope's base
	Format string `json:"format"` // json
}

// ConfigDescriptor describes one package's configuration file.
type ConfigDescriptor struct {
	ID          string        `json:"id"`
	Match       string        `json:"match"` // substring of the package name or source
	Title       string        `json:"title"`
	Application string        `json:"application"` // the honest-apply sentence the page renders
	Files       []ConfigFile  `json:"files"`
	Fields      []ConfigField `json:"fields"`
}

// catalogDescriptors are the descriptors PiCode knows without any help from
// the package. The provider list is the tool's own supported set — a value
// outside it is rejected here before the tool ever sees it.
func catalogDescriptors() []ConfigDescriptor {
	return []ConfigDescriptor{{
		ID:    "web-search",
		Match: "pi-web-search",
		Title: "Web search",
		Application: "Applies on the next search — the tool reads this file each " +
			"run; the running conversation keeps its own model.",
		Files: []ConfigFile{{Scope: "agent", Path: "web-search.json", Format: "json"}},
		Fields: []ConfigField{
			{
				Key: "provider", Label: "Provider", Type: "enum", Required: true,
				Options: []string{
					"google-generative-ai", "xai", "openai-responses",
					"azure-openai-responses", "openai-codex-responses",
					"anthropic-messages",
				},
				Help: "Must back native web search.",
			},
			{
				Key: "model", Label: "Model", Type: "string", Required: true,
				Help: "Model id at that provider, e.g. gemini-3.5-flash.",
			},
		},
	},
		{
			ID:    "compact",
			Match: "pi-compact",
			Title: "Compaction",
			Application: "Applies from the next turn on — the extension re-reads this " +
				"file per input. Empty fields stay unset: the chain decides.",
			Files: []ConfigFile{{Scope: "workspace", Path: ".pi/compact.json", Format: "json"}},
			Fields: []ConfigField{
				{Key: "enabled", Label: "Enabled", Type: "boolean", Help: "Unchecked = off. Leave untouched to keep unset."},
				{Key: "atTokens", Label: "At tokens", Type: "number", Min: f64(1), Help: "Absolute token trigger. Empty = knob off."},
				{Key: "atPercent", Label: "At fraction", Type: "number", Min: f64(0), Max: f64(1), Help: "Fraction of the context window, 0–1 (e.g. 0.5)."},
				{Key: "floorTokens", Label: "Floor tokens", Type: "number", Min: f64(1), Help: "Never compact below this many tokens."},
				{Key: "cooldownTurns", Label: "Cooldown turns", Type: "number", Min: f64(0), Help: "Turns to wait after a compaction."},
				{Key: "model", Label: "Summarizer model", Type: "string", Help: "provider/id, e.g. zai/glm-5.3. Empty = auto chain."},
				{Key: "thinking", Label: "Thinking", Type: "enum", Options: []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}},
				{Key: "instructions", Label: "Instructions", Type: "string", Help: "Extra instructions for the summarizer."},
			},
		}}
}

// f64 keeps the catalog table reading as numbers, not pointers.
func f64(v float64) *float64 { return &v }

// remembered holds manifest-declared descriptors found while listing
// installed packages, so a later GET/PUT/DELETE that only knows the id can
// resolve them. Catalog entries need no remembering: they are static.
var remembered = map[string]*ConfigDescriptor{}

// DescriptorFor resolves the descriptor for one package: the catalog by
// name/source substring, then a picode.config manifest beside the installed
// package. Nil when nothing describes it — the honest absence ADR-0099 §5
// keeps.
func DescriptorFor(nameOrSource, installedPath string) *ConfigDescriptor {
	for i := range catalogDescriptors() {
		d := &catalogDescriptors()[i]
		if d.matches(nameOrSource) {
			return d
		}
	}
	if installedPath != "" {
		if d := descriptorFromManifest(installedPath); d != nil {
			remembered[d.ID] = d
			return d
		}
	}
	for _, d := range remembered {
		if d.matches(nameOrSource) {
			return d
		}
	}
	return nil
}

// DescriptorByID fetches a descriptor previously resolvable on this
// machine: user-described first, then catalog entries, then
// manifest-declared ones seen while listing.
func DescriptorByID(id string) *ConfigDescriptor {
	if id == "" {
		return nil
	}
	if d := UserDescriptorByID(id); d != nil {
		return d
	}
	for i := range catalogDescriptors() {
		if catalogDescriptors()[i].ID == id {
			return &catalogDescriptors()[i]
		}
	}
	return remembered[id]
}

func (d *ConfigDescriptor) matches(nameOrSource string) bool {
	return d.Match != "" && nameOrSource != "" &&
		strings.Contains(strings.ToLower(nameOrSource), strings.ToLower(d.Match))
}

// descriptorFromManifest reads picode.config from an installed package's
// package.json — the upstream-friendly path: a package carries its own
// description. Match defaults to the package name. A manifest that omits
// the essentials (id, title, files, at least one field) is ignored, not an
// error: half a description is still no description.
func descriptorFromManifest(installedPath string) *ConfigDescriptor {
	raw, err := os.ReadFile(filepath.Join(installedPath, "package.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		Name   string `json:"name"`
		Picode struct {
			Config *ConfigDescriptor `json:"config"`
		} `json:"picode"`
	}
	if json.Unmarshal(raw, &doc) != nil || doc.Picode.Config == nil {
		return nil
	}
	d := doc.Picode.Config
	if d.ID == "" || d.Title == "" || len(d.Files) == 0 || len(d.Fields) == 0 {
		return nil
	}
	if d.Match == "" {
		d.Match = doc.Name
	}
	return d
}

// DescriptorFileAbs resolves one of the descriptor's files to an absolute
// path. Agent files hang off UserDir; workspace files off the workspace
// folder the caller resolves. Unknown scopes are refused: a descriptor may
// only use scopes the server knows how to place.
func DescriptorFileAbs(d *ConfigDescriptor, scope, workspacePath string) (string, error) {
	for _, f := range d.Files {
		if f.Scope != scope {
			continue
		}
		switch f.Scope {
		case "agent":
			return filepath.Join(UserDir(), filepath.FromSlash(f.Path)), nil
		case "workspace":
			if workspacePath == "" {
				return "", fmt.Errorf("workspace-scope config needs the workspace folder")
			}
			return filepath.Join(workspacePath, filepath.FromSlash(f.Path)), nil
		}
	}
	return "", fmt.Errorf("descriptor %q declares no %q file", d.ID, scope)
}

// ValidateDescriptorValues checks a values map against the descriptor's
// fields: required means present and non-empty; enum means one of the
// options. Unknown keys are none of the descriptor's business — they are
// preserved on write, never validated.
func ValidateDescriptorValues(d *ConfigDescriptor, values map[string]any) error {
	if values == nil {
		values = map[string]any{}
	}
	for _, f := range d.Fields {
		v, present := values[f.Key]
		empty := !present || v == nil || v == ""
		if f.Required && empty {
			return fmt.Errorf("%s is required", f.Label)
		}
		if empty {
			continue
		}
		switch f.Type {
		case "enum":
			s, ok := v.(string)
			if !ok || !containsString(f.Options, s) {
				return fmt.Errorf("%s must be one of: %s", f.Label, strings.Join(f.Options, ", "))
			}
		case "string", "secret":
			if _, ok := v.(string); !ok {
				return fmt.Errorf("%s must be text", f.Label)
			}
		case "boolean":
			if _, ok := v.(bool); !ok {
				return fmt.Errorf("%s must be true or false", f.Label)
			}
		case "number":
			n, ok := v.(float64)
			if !ok {
				return fmt.Errorf("%s must be a number", f.Label)
			}
			if f.Min != nil && n < *f.Min {
				return fmt.Errorf("%s must be at least %s", f.Label, trimNum(*f.Min))
			}
			if f.Max != nil && n > *f.Max {
				return fmt.Errorf("%s must be at most %s", f.Label, trimNum(*f.Max))
			}
		}
	}
	return nil
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// trimNum prints a bound the short way ("1", "0.5" — never "1.0").
func trimNum(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// DescriptorLayer is one config file as the GET view returns it.
type DescriptorLayer struct {
	Path    string         `json:"path"`
	Exists  bool           `json:"exists"`
	Invalid string         `json:"invalid,omitempty"`
	Values  map[string]any `json:"values"`
}

// ReadDescriptorLayer reads and parses one descriptor file. A file the
// parser refuses is reported, never silently emptied.
func ReadDescriptorLayer(abs string) DescriptorLayer {
	layer := DescriptorLayer{Path: abs, Values: map[string]any{}}
	b, err := os.ReadFile(abs)
	if err != nil {
		return layer
	}
	layer.Exists = true
	var values map[string]any
	if err := json.Unmarshal(b, &values); err != nil {
		layer.Invalid = err.Error()
		layer.Values = map[string]any{}
		return layer
	}
	if values == nil {
		values = map[string]any{}
	}
	layer.Values = values
	return layer
}

// WriteDescriptorFile merges the typed values onto the raw document —
// unknown keys survive — and writes atomically (tmp + rename), the same
// contract WriteRolesFile gives the roles editor.
func WriteDescriptorFile(abs string, values, raw map[string]any) error {
	if raw == nil {
		raw = map[string]any{}
	}
	for k, v := range values {
		raw[k] = v
	}
	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".desc-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, abs)
}

// --- user-described configs (C5) ---
//
// A user descriptor is written by the owner through the Packages view
// ("Describe config…") for a package whose configuration the catalog and
// the package itself do not describe. It resolves BEFORE the catalog: the
// explicit act of describing wins. Deleting it falls back honestly.

var (
	userMu       sync.RWMutex
	userRegistry = map[string]*ConfigDescriptor{} // id -> descriptor
)

var fieldKeyRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.-]*$`)

// SanitizeDescriptorID makes an id safe as a file name (scoped npm names
// contain "/").
func SanitizeDescriptorID(id string) string {
	s := strings.ReplaceAll(strings.TrimSpace(id), "/", "__")
	return strings.ReplaceAll(s, "\\", "__")
}

// ValidateUserDescriptor checks a user-authored descriptor before it is
// persisted. Field keys must be unique and well-formed; enum fields need
// options; number bounds cannot invert.
func ValidateUserDescriptor(d *ConfigDescriptor) error {
	if strings.TrimSpace(d.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(d.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if len(d.Files) != 1 {
		return fmt.Errorf("exactly one file must be declared")
	}
	f := d.Files[0]
	if f.Scope != "agent" && f.Scope != "workspace" {
		return fmt.Errorf("file scope must be agent or workspace")
	}
	if f.Path == "" || strings.HasPrefix(f.Path, "/") || strings.Contains(f.Path, "..") {
		return fmt.Errorf("file path must be relative and cannot climb (no ..)")
	}
	if f.Format != "json" {
		return fmt.Errorf("only json files are supported")
	}
	if len(d.Fields) == 0 {
		return fmt.Errorf("describe at least one field")
	}
	seen := map[string]bool{}
	for i := range d.Fields {
		fl := &d.Fields[i]
		if fl.Key == "" || !fieldKeyRe.MatchString(fl.Key) {
			return fmt.Errorf("field %d: key must be letters, digits, _ - or .", i+1)
		}
		if seen[fl.Key] {
			return fmt.Errorf("field %q is duplicated", fl.Key)
		}
		seen[fl.Key] = true
		if strings.TrimSpace(fl.Label) == "" {
			return fmt.Errorf("field %q needs a label", fl.Key)
		}
		switch fl.Type {
		case "string", "secret", "boolean", "number", "enum":
		default:
			return fmt.Errorf("field %q has unknown type %q", fl.Key, fl.Type)
		}
		if fl.Type == "enum" && len(fl.Options) == 0 {
			return fmt.Errorf("field %q needs options", fl.Key)
		}
		if fl.Type == "number" && fl.Min != nil && fl.Max != nil && *fl.Min > *fl.Max {
			return fmt.Errorf("field %q: min is above max", fl.Key)
		}
	}
	return nil
}

// RegisterUserDescriptor makes a user descriptor resolvable immediately.
func RegisterUserDescriptor(d *ConfigDescriptor) {
	userMu.Lock()
	defer userMu.Unlock()
	cp := *d
	userRegistry[d.ID] = &cp
}

// UserDescriptorByID returns the user descriptor with this exact id.
func UserDescriptorByID(id string) *ConfigDescriptor {
	userMu.RLock()
	defer userMu.RUnlock()
	if d, ok := userRegistry[id]; ok {
		cp := *d
		return &cp
	}
	return nil
}

// UserDescriptorFor matches a user descriptor against a package name or
// source string.
func UserDescriptorFor(nameOrSource string) *ConfigDescriptor {
	userMu.RLock()
	defer userMu.RUnlock()
	for _, d := range userRegistry {
		if d.matches(nameOrSource) {
			cp := *d
			return &cp
		}
	}
	return nil
}

// LoadUserDescriptors registers every descriptor stored under dir. A file
// that no longer validates is skipped, not fatal: one bad description must
// not take the others down.
func LoadUserDescriptors(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var d ConfigDescriptor
		if json.Unmarshal(raw, &d) != nil || d.ID == "" {
			continue
		}
		RegisterUserDescriptor(&d)
	}
}

// SaveUserDescriptor validates and persists a user descriptor atomically,
// then registers it.
func SaveUserDescriptor(dir string, d *ConfigDescriptor) error {
	if err := ValidateUserDescriptor(d); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	abs := filepath.Join(dir, SanitizeDescriptorID(d.ID)+".json")
	tmp, err := os.CreateTemp(dir, ".desc-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return err
	}
	RegisterUserDescriptor(d)
	return nil
}

// DeleteUserDescriptor removes a stored user descriptor and unregisters it.
// Removing something that is not there succeeds.
func DeleteUserDescriptor(dir, id string) error {
	userMu.Lock()
	delete(userRegistry, id)
	userMu.Unlock()
	err := os.Remove(filepath.Join(dir, SanitizeDescriptorID(id)+".json"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DescriptorOrigin names where the descriptor with this id came from:
// "user", "catalog" or "manifest". Empty when nothing resolves — the same
// answer DescriptorByID would give.
func DescriptorOrigin(id string) string {
	if id == "" {
		return ""
	}
	if UserDescriptorByID(id) != nil {
		return "user"
	}
	for i := range catalogDescriptors() {
		if catalogDescriptors()[i].ID == id {
			return "catalog"
		}
	}
	if _, ok := remembered[id]; ok {
		return "manifest"
	}
	return ""
}
