package catalog

// Custom provider definitions for omp (oh-my-pi): the same GUI (ADR-0129)
// pointed at omp's own file, ~/.omp/agent/models.yml (the owner's call that
// amends ADR-0169's "custom endpoints stay pi-only"). omp is pi-family but
// keeps its models in YAML and its credentials in a SQLite database PiCode
// does not touch, so two rules differ from modelsjson.go:
//
//   - the merge is node-level YAML: untouched providers, unknown fields
//     inside a touched one (headers, discovery, modelOverrides) and the
//     file's comments survive;
//   - the API key lives INSIDE the definition, as the provider's `apiKey` —
//     omp's own documented channel and the only one a custom id has (no
//     auth.json, no env mapping, no login flow). It is never serialized
//     back: the roster carries `keyed`, the verify/listing paths read it
//     server-side and send it to the gateway the user configured, nowhere
//     else.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// ompBuiltinProviders are the provider ids omp ships (its env-var docs and
// models.yml docs, read 2026-09-21 from the installed 18.2.8). Vendored
// knowledge like LoginMethods is for pi: a custom id never shadows one.
var ompBuiltinProviders = map[string]bool{
	"anthropic": true, "openai": true, "openai-codex": true, "google": true,
	"google-vertex": true, "google-gemini-cli": true, "google-antigravity": true,
	"github-copilot": true, "azure-openai": true, "aws-bedrock": true,
	"xai": true, "groq": true, "cerebras": true, "mistral": true,
	"openrouter": true, "kilo": true, "zai": true, "minimax": true,
	"opencode": true, "cursor": true, "cline": true, "command-code": true,
	"charm-hyper": true, "ai-gateway": true, "wafer-serverless": true,
	"yolo-auto": true, "umans-ai-coding-plan": true, "abliteration": true,
	"kimi-coding": true, "moonshot": true, "meta-ai": true, "deepseek": true,
	"ollama": true, "llama.cpp": true, "lm-studio": true, "litellm": true,
}

// ompThinkingFormats are the compat.thinkingFormat values omp's own compat
// table documents for chat completions. pi's wider union (deepseek, together,
// baseten, …) is pi's wire code, not omp's, so the omp writer refuses the
// rest instead of writing a value omp would send wrong.
var ompThinkingFormats = []string{"openai", "openrouter", "zai", "qwen", "qwen-chat-template"}

// ompCompatKeys are the compat flags the omp form manages: the two omp's
// models.yml doc example carries. pi's supportsUsageInStreaming is pi's
// Responses-usage switch; omp reads usage without it, so it is not written.
var ompCompatKeys = []string{"supportsDeveloperRole", "supportsReasoningEffort"}

// ompTestDir points the path at a fixture (tests); empty means the real home.
var ompTestDir string

// OMPModelsPath is omp's provider-definition file: $PI_CODING_AGENT_DIR (the
// override omp itself resolves first) or ~/.omp/agent, then models.yml.
func OMPModelsPath() string {
	dir := ompTestDir
	if dir == "" {
		if override := strings.TrimSpace(os.Getenv("PI_CODING_AGENT_DIR")); override != "" {
			dir = override
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			dir = filepath.Join(home, ".omp", "agent")
		}
	}
	return filepath.Join(dir, "models.yml")
}

// OMPValidateCustomID applies the shared id rules to omp's built-in set.
func OMPValidateCustomID(id string) error {
	id = strings.TrimSpace(id)
	if err := validateCustomIDShape(id); err != nil {
		return err
	}
	if ompBuiltinProviders[id] {
		return fmt.Errorf("%s is a built-in provider; pick another name", id)
	}
	return nil
}

// CustomRow is one custom provider as the roster serves it: the editable
// shape (identical to what /api/catalog serves for pi, so the form prefills
// the same way) plus whether a key is saved — never the key itself.
type CustomRow struct {
	BaseURL            string          `json:"baseUrl"`
	API                string          `json:"api"`
	Compat             map[string]bool `json:"compat,omitempty"`
	ThinkingFormat     string          `json:"thinkingFormat,omitempty"`
	ChatTemplateKwargs map[string]any  `json:"chatTemplateKwargs,omitempty"`
	ChatTemplateArgs   map[string]any  `json:"chatTemplateArgs,omitempty"`
	Definitions        []CustomModel   `json:"definitions"`
	Keyed              bool            `json:"keyed"`
}

// OMPLoadCustomDefinitions returns the custom definitions in models.yml:
// ids that parse, are not built-ins, and carry a usable baseUrl. A missing
// or unreadable file means none — never an error, the roster still loads.
func OMPLoadCustomDefinitions() map[string]CustomRow {
	out := map[string]CustomRow{}
	path := OMPModelsPath()
	if path == "" {
		return out
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var doc yaml.Node
	if yaml.Unmarshal(raw, &doc) != nil {
		return out
	}
	providers := yamlProvidersMapping(&doc)
	if providers == nil {
		return out
	}
	for i := 0; i+1 < len(providers.Content); i += 2 {
		var id string
		if providers.Content[i].Decode(&id) != nil {
			continue
		}
		if ompBuiltinProviders[id] {
			continue
		}
		entry := providers.Content[i+1]
		if entry == nil || entry.Kind != yaml.MappingNode {
			continue
		}
		var parsed struct {
			BaseURL string `yaml:"baseUrl"`
			API     string `yaml:"api"`
			Models  []struct {
				ID            string         `yaml:"id"`
				Name          string         `yaml:"name"`
				Reasoning     *bool          `yaml:"reasoning"`
				Input         []string       `yaml:"input"`
				Cost          *CustomCost    `yaml:"cost"`
				ContextWindow *int           `yaml:"contextWindow"`
				MaxTokens     *int           `yaml:"maxTokens"`
				LevelMap      map[string]any `yaml:"thinkingLevelMap"`
			} `yaml:"models"`
		}
		if entry.Decode(&parsed) != nil || parsed.BaseURL == "" {
			continue
		}
		if parsed.API == "" {
			parsed.API = APIOpenAICompletions
		}
		row := CustomRow{BaseURL: parsed.BaseURL, API: parsed.API}
		row.ThinkingFormat = ompCompatString(entry, "thinkingFormat")
		for _, k := range ompCompatKeys {
			var v bool
			if compat := mappingValueKind(entry, "compat", yaml.MappingNode); compat != nil {
				if c := mappingGet(compat, k); c != nil && c.Decode(&v) == nil {
					if row.Compat == nil {
						row.Compat = map[string]bool{}
					}
					row.Compat[k] = v
				}
			}
		}
		for _, m := range parsed.Models {
			def := CustomModel{
				ID: m.ID, Name: m.Name, Reasoning: m.Reasoning, Input: m.Input,
				Cost: m.Cost, ContextWindow: m.ContextWindow, MaxTokens: m.MaxTokens,
				ThinkingLevelMap: m.LevelMap,
			}
			if def.ThinkingLevelMap != nil {
				for _, l := range customThinkingLevels {
					if v, ok := def.ThinkingLevelMap[l].(string); ok && v != "" {
						def.ThinkingLevels = append(def.ThinkingLevels, l)
					}
				}
			}
			row.Definitions = append(row.Definitions, def)
		}
		keyed := false
		if n := mappingGet(entry, "apiKey"); n != nil {
			var key string
			if n.Decode(&key) == nil && strings.TrimSpace(key) != "" {
				keyed = true
			}
		}
		row.Keyed = keyed
		out[id] = row
	}
	return out
}

// ompCompatString reads one string compat value (the thinking format). A
// non-string reads as absent.
func ompCompatString(entry *yaml.Node, key string) string {
	compat := mappingValueKind(entry, "compat", yaml.MappingNode)
	if compat == nil {
		return ""
	}
	n := mappingGet(compat, key)
	if n == nil {
		return ""
	}
	var s string
	if n.Decode(&s) != nil {
		return ""
	}
	return s
}

// OMPCustomAPIKey reads one custom provider's saved key, for the server-side
// paths that forward it to the gateway (verify, model listing). It is never
// written to a response.
func OMPCustomAPIKey(id string) (string, bool) {
	id = strings.TrimSpace(id)
	defs := OMPLoadCustomDefinitions()
	if _, ok := defs[id]; !ok {
		return "", false
	}
	var doc yaml.Node
	raw, err := os.ReadFile(OMPModelsPath())
	if err != nil {
		return "", false
	}
	if yaml.Unmarshal(raw, &doc) != nil {
		return "", false
	}
	providers := yamlProvidersMapping(&doc)
	entry := yamlMappingValue(providers, id)
	if entry == nil {
		return "", false
	}
	n := mappingGet(entry, "apiKey")
	if n == nil {
		return "", false
	}
	var key string
	if n.Decode(&key) != nil {
		return "", false
	}
	return key, true
}

// OMPUpsertCustomProvider validates and merges one definition into
// models.yml, creating the file (0600, agent dir 0700) when absent. The key,
// when non-empty, is written as the provider's apiKey; empty leaves the
// stored one alone (Edit with a blank key keeps the credential).
func OMPUpsertCustomProvider(id string, def CustomDefinition, key string) error {
	id = strings.TrimSpace(id)
	if err := OMPValidateCustomID(id); err != nil {
		return err
	}
	if err := validateCustomDef(def); err != nil {
		return err
	}
	if def.ThinkingFormat != "" && !slices.Contains(ompThinkingFormats, def.ThinkingFormat) {
		return fmt.Errorf("omp does not support thinking format %s", def.ThinkingFormat)
	}
	path := OMPModelsPath()
	if path == "" {
		return fmt.Errorf("no home directory")
	}
	var doc yaml.Node
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("models.yml is not valid YAML: %w", err)
		}
	case errors.Is(err, os.ErrNotExist):
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mappingNode()}}
	default:
		return err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("models.yml is not a YAML document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("models.yml is not a mapping")
	}
	// omp's schema allows one root key, providers. A file it would reject
	// stays its rejection even after our merge — writing into it would report
	// success while the gateway never loads. Refuse and name the fix instead.
	for i := 0; i+1 < len(root.Content); i += 2 {
		var k string
		_ = root.Content[i].Decode(&k)
		if k != "providers" {
			return fmt.Errorf("models.yml has entries outside the providers key; omp's schema allows only \"providers\" at the top level")
		}
	}
	providers := yamlMappingValue(root, "providers")
	if providers == nil || providers.Kind != yaml.MappingNode {
		providers = mappingNode()
		mappingSet(root, "providers", providers)
	}
	entry := yamlMappingValue(providers, id)
	if entry == nil || entry.Kind != yaml.MappingNode {
		entry = mappingNode()
		mappingSet(providers, id, entry)
	}

	// Managed scalars: a definition always owns baseUrl and api.
	mappingSet(entry, "baseUrl", scalarNode(def.BaseURL))
	mappingSet(entry, "api", scalarNode(def.API))

	// Models: the form's list is the array; per-model unknown fields survive
	// for surviving ids, matched by id.
	existingModels := mappingValueKind(entry, "models", yaml.SequenceNode)
	models := &yaml.Node{Kind: yaml.SequenceNode}
	for _, m := range def.Models {
		item := ompFindModel(existingModels, m.ID)
		if item == nil {
			item = mappingNode()
		}
		mappingSet(item, "id", scalarNode(m.ID))
		if m.Reasoning != nil {
			mappingSet(item, "reasoning", scalarNode(*m.Reasoning))
		}
		if m.ContextWindow != nil {
			mappingSet(item, "contextWindow", scalarNode(*m.ContextWindow))
		}
		if m.MaxTokens != nil {
			mappingSet(item, "maxTokens", scalarNode(*m.MaxTokens))
		}
		// Name, input and cost are form-managed: a value writes it, a blank
		// deletes the stored key, so the form owns what it shows.
		if name := strings.TrimSpace(m.Name); name != "" {
			mappingSet(item, "name", scalarNode(name))
		} else {
			mappingDelete(item, "name")
		}
		if len(m.Input) > 0 {
			input := &yaml.Node{Kind: yaml.SequenceNode}
			for _, mod := range canonicalInput(m.Input) {
				input.Content = append(input.Content, scalarNode(mod))
			}
			mappingSet(item, "input", input)
		} else {
			mappingDelete(item, "input")
		}
		if m.Cost != nil {
			cost := mappingNode()
			mappingSet(cost, "input", scalarNode(m.Cost.Input))
			mappingSet(cost, "output", scalarNode(m.Cost.Output))
			mappingSet(cost, "cacheRead", scalarNode(m.Cost.CacheRead))
			mappingSet(cost, "cacheWrite", scalarNode(m.Cost.CacheWrite))
			mappingSet(item, "cost", cost)
		} else {
			mappingDelete(item, "cost")
		}
		// omp's models.yml schema has no thinkingLevelMap (its thinking
		// levels are runtime selection, not a per-model map), so the form's
		// level selection writes `reasoning` only; a stale map a hand edit
		// added is left alone rather than deleted behind the user's back.
		models.Content = append(models.Content, item)
	}
	mappingSet(entry, "models", models)

	// Compat: only the keys omp accepts are written; unknown hand-set keys
	// survive. A definition that sends no compat at all (a bare API caller)
	// leaves the block alone.
	if def.Compat != nil || def.ThinkingFormat != "" {
		compat := mappingValueKind(entry, "compat", yaml.MappingNode)
		if compat == nil {
			compat = mappingNode()
			mappingSet(entry, "compat", compat)
		}
		for _, k := range ompCompatKeys {
			if v, managed := def.Compat[k]; managed {
				mappingSet(compat, k, scalarNode(v))
			}
		}
		if def.ThinkingFormat != "" {
			mappingSet(compat, "thinkingFormat", scalarNode(def.ThinkingFormat))
		} else if def.Compat != nil {
			mappingDelete(compat, "thinkingFormat")
		}
	}

	// The key is the one credential channel omp gives a custom id. Empty
	// means "keep the saved one" (an Edit that does not retype the key); an
	// explicit removal is Remove provider, which deletes definition and key.
	if k := strings.TrimSpace(key); k != "" {
		mappingSet(entry, "apiKey", scalarNode(k))
	}
	return writeYAMLFile(path, &doc)
}

// OMPRemoveCustomProvider deletes one definition and its key. It refuses ids
// it does not own (built-ins, absent entries, entries without a baseUrl —
// an override of a built-in is not ours to delete).
func OMPRemoveCustomProvider(id string) error {
	id = strings.TrimSpace(id)
	if err := OMPValidateCustomID(id); err != nil {
		return err
	}
	path := OMPModelsPath()
	if path == "" {
		return fmt.Errorf("no home directory")
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s is not a custom provider", id)
	} else if err != nil {
		return err
	}
	var doc yaml.Node
	if yaml.Unmarshal(raw, &doc) != nil {
		return fmt.Errorf("models.yml is not valid YAML")
	}
	providers := yamlProvidersMapping(&doc)
	if providers == nil || yamlMappingValue(providers, id) == nil {
		return fmt.Errorf("%s is not a custom provider", id)
	}
	entry := yamlMappingValue(providers, id)
	var baseURL string
	if n := mappingGet(entry, "baseUrl"); n == nil || n.Decode(&baseURL) != nil || baseURL == "" {
		return fmt.Errorf("%s is not a custom provider", id)
	}
	mappingDelete(providers, id)
	return writeYAMLFile(path, &doc)
}

// yamlProvidersMapping finds the top-level providers mapping node, or nil.
func yamlProvidersMapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	providers := yamlMappingValue(root, "providers")
	if providers == nil || providers.Kind != yaml.MappingNode {
		return nil
	}
	return providers
}

// yamlMappingValue returns the value node for key in a mapping node.
func yamlMappingValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		var k string
		if m.Content[i].Decode(&k) == nil && k == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// mappingGet is yamlMappingValue without the kind check on the result, for
// scalars the caller decodes itself.
func mappingGet(m *yaml.Node, key string) *yaml.Node {
	return yamlMappingValue(m, key)
}

// mappingSet replaces a key's value in place — order and the key node's
// comments survive — or appends the pair when the key is new.
func mappingSet(m *yaml.Node, key string, val *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		var k string
		if m.Content[i].Decode(&k) == nil && k == key {
			m.Content[i+1] = val
			return
		}
	}
	m.Content = append(m.Content, scalarNode(key), val)
}

// mappingDelete removes a key's pair; a missing key is a no-op.
func mappingDelete(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		var k string
		if m.Content[i].Decode(&k) == nil && k == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

// mappingValueKind returns the value node for key when it has the wanted
// kind, else nil — a hand-written scalar where an object belongs reads as
// absent instead of failing the write.
func mappingValueKind(m *yaml.Node, key string, kind yaml.Kind) *yaml.Node {
	n := yamlMappingValue(m, key)
	if n == nil || n.Kind != kind {
		return nil
	}
	return n
}

func mappingNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode}
}

func scalarNode(v any) *yaml.Node {
	n := yaml.Node{Kind: yaml.ScalarNode}
	if err := n.Encode(v); err != nil {
		panic("catalog: encode yaml scalar: " + err.Error())
	}
	return &n
}

// ompFindModel finds one existing model item by id in a models sequence,
// preserving its unknown fields. A non-mapping item is skipped.
func ompFindModel(seq *yaml.Node, id string) *yaml.Node {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil
	}
	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		var mid string
		if n := mappingGet(item, "id"); n != nil && n.Decode(&mid) == nil && mid == id {
			return item
		}
	}
	return nil
}

// writeYAMLFile writes the document atomically at 0600 (agent dir 0700),
// comments included where yaml.v3 kept them.
func writeYAMLFile(path string, doc *yaml.Node) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".models-*.yml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
