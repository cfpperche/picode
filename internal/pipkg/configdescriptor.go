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
	"strings"
)

// ConfigField is one typed input of a descriptor-driven config.
type ConfigField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // string | enum | boolean | number | secret
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"` // enum choices
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
	}}
}

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
// machine: catalog entries and manifest-declared ones seen while listing.
func DescriptorByID(id string) *ConfigDescriptor {
	if id == "" {
		return nil
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
// path. Unknown scopes are refused: a descriptor may only use the scopes
// the server knows how to place.
func DescriptorFileAbs(d *ConfigDescriptor, scope string) (string, error) {
	for _, f := range d.Files {
		if f.Scope == scope {
			switch f.Scope {
			case "agent":
				return filepath.Join(UserDir(), filepath.FromSlash(f.Path)), nil
			case "workspace":
				return "", fmt.Errorf("workspace-scope config needs the workspace folder")
			}
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
			if _, ok := v.(float64); !ok {
				return fmt.Errorf("%s must be a number", f.Label)
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
