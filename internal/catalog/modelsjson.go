package catalog

// Custom provider definitions: PiCode's GUI for pi's ~/.pi/agent/models.json
// (ADR-0129). Definitions live in pi's own file; API keys never do — they go
// to auth.json (PutAPIKey) so pi's resolution order keeps one source of truth
// per concern. Merge semantics: upsert by provider id; untouched providers
// and unknown fields inside the touched provider (headers, samplingParams,
// cost) survive byte-for-byte at the JSON value level.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The four APIs pi accepts on a provider (pi docs/models.md).
const (
	APIOpenAICompletions  = "openai-completions"
	APIOpenAIResponses    = "openai-responses"
	APIAnthropicMessages  = "anthropic-messages"
	APIGoogleGenerativeAI = "google-generative-ai"
)

// CustomAPIs is the form's closed list, in display order.
var CustomAPIs = []string{APIOpenAICompletions, APIOpenAIResponses, APIAnthropicMessages, APIGoogleGenerativeAI}

// customIDPattern: lowercase slug. Dots are excluded, which also rules out
// "llama.cpp"; collisions with built-ins are refused by name below.
var customIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// compat keys the GUI manages; anything else in a hand-edited compat object
// is preserved, never invented here.
var customCompatKeys = []string{"supportsDeveloperRole", "supportsReasoningEffort"}

// CustomModel is one model row from the form. Pointers distinguish "leave
// pi's default" from zero; unknown model fields are preserved on merge.
type CustomModel struct {
	ID            string `json:"id"`
	Reasoning     *bool  `json:"reasoning,omitempty"`
	ContextWindow *int   `json:"contextWindow,omitempty"`
	MaxTokens     *int   `json:"maxTokens,omitempty"`
}

// CustomDefinition is the editable shape of one models.json provider entry.
type CustomDefinition struct {
	BaseURL string          `json:"baseUrl"`
	API     string          `json:"api"`
	Compat  map[string]bool `json:"compat,omitempty"`
	Models  []CustomModel   `json:"models"`
}

// ModelsPath is pi's provider-definition file. PiCode merges into it; it is
// never deleted or replaced wholesale.
func ModelsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "models.json")
}

// ValidateCustomID refuses empty/ill-formed ids and every built-in provider
// id — a custom definition never shadows a native provider. Override of a
// built-in's baseUrl is a separate, deliberately unsupported action.
func ValidateCustomID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("name required")
	}
	if !customIDPattern.MatchString(id) {
		return fmt.Errorf("name must be lowercase letters, digits and dashes")
	}
	if _, builtin := LoginMethods[id]; builtin {
		return fmt.Errorf("%s is a built-in provider; pick another name", id)
	}
	return nil
}

// LoadCustomDefinitions returns the custom definitions found in models.json:
// ids that parse, are not built-ins, and carry a usable baseUrl. Missing or
// unreadable file means none — never an error, the catalog still loads.
// pi's schema wraps every provider under the top-level "providers" key; the
// wrapper is read here and written by every mutation below.
func LoadCustomDefinitions() map[string]CustomDefinition {
	out := map[string]CustomDefinition{}
	path := ModelsPath()
	if path == "" {
		return out
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for id, entryRaw := range readProviders(raw) {
		if _, builtin := LoginMethods[id]; builtin {
			continue
		}
		var entry struct {
			BaseURL string          `json:"baseUrl"`
			API     string          `json:"api"`
			Compat  map[string]bool `json:"compat"`
			Models  []CustomModel   `json:"models"`
		}
		if json.Unmarshal(entryRaw, &entry) != nil {
			continue
		}
		if entry.BaseURL == "" {
			continue
		}
		if entry.API == "" {
			entry.API = APIOpenAICompletions
		}
		out[id] = CustomDefinition{BaseURL: entry.BaseURL, API: entry.API, Compat: entry.Compat, Models: entry.Models}
	}
	return out
}

// readProviders extracts the providers object pi's schema defines. It
// returns an empty map — never an error — for a missing wrapper, an invalid
// file, or a wrapper that is not an object.
func readProviders(raw []byte) map[string]json.RawMessage {
	var top struct {
		Providers map[string]json.RawMessage `json:"providers"`
	}
	if json.Unmarshal(raw, &top) != nil {
		return map[string]json.RawMessage{}
	}
	if top.Providers == nil {
		return map[string]json.RawMessage{}
	}
	return top.Providers
}

// UpsertCustomProvider validates and merges one definition into models.json.
// The key is NOT this function's concern (auth.json is).
func UpsertCustomProvider(id string, def CustomDefinition) error {
	id = strings.TrimSpace(id)
	if err := ValidateCustomID(id); err != nil {
		return err
	}
	if err := validateCustomDef(def); err != nil {
		return err
	}
	path := ModelsPath()
	if path == "" {
		return fmt.Errorf("no home directory")
	}
	obj, err := readModelsTop(path)
	if err != nil {
		return err
	}
	providers := readProviders(mustRaw(obj))
	entry := map[string]json.RawMessage{}
	if old, ok := providers[id]; ok {
		// A broken hand-edited entry starts clean; a valid one keeps its
		// unknown fields (headers, oauth blocks, anything pi adds later).
		_ = json.Unmarshal(old, &entry)
	}

	entry["baseUrl"] = mustRaw(def.BaseURL)
	entry["api"] = mustRaw(def.API)

	// Models: the form's list is the array; per-model unknown fields
	// (samplingParams, cost, thinkingLevelMap) survive for surviving ids.
	existing := map[string]map[string]json.RawMessage{}
	if rawModels, ok := entry["models"]; ok {
		var arr []json.RawMessage
		if json.Unmarshal(rawModels, &arr) == nil {
			for _, m := range arr {
				var fields map[string]json.RawMessage
				if json.Unmarshal(m, &fields) != nil {
					continue
				}
				var mid string
				_ = json.Unmarshal(fields["id"], &mid)
				if mid != "" {
					existing[mid] = fields
				}
			}
		}
	}
	models := make([]map[string]json.RawMessage, 0, len(def.Models))
	for _, m := range def.Models {
		fields := map[string]json.RawMessage{}
		if prev, ok := existing[m.ID]; ok {
			fields = prev
		}
		fields["id"] = mustRaw(m.ID)
		if m.Reasoning != nil {
			fields["reasoning"] = mustRaw(*m.Reasoning)
		}
		if m.ContextWindow != nil {
			fields["contextWindow"] = mustRaw(*m.ContextWindow)
		}
		if m.MaxTokens != nil {
			fields["maxTokens"] = mustRaw(*m.MaxTokens)
		}
		models = append(models, fields)
	}
	modelsRaw, err := json.Marshal(models)
	if err != nil {
		return err
	}
	entry["models"] = modelsRaw

	// Compat: only the keys the form manages are written; unknown hand-set
	// keys survive. Absent from the form means absent from the entry only
	// when the form sent no compat at all (Edit always sends both keys).
	if def.Compat != nil {
		compat := map[string]json.RawMessage{}
		if rawCompat, ok := entry["compat"]; ok {
			_ = json.Unmarshal(rawCompat, &compat)
		}
		for _, k := range customCompatKeys {
			v, managed := def.Compat[k]
			if managed {
				compat[k] = mustRaw(v)
			}
		}
		compatRaw, err := json.Marshal(compat)
		if err != nil {
			return err
		}
		entry["compat"] = compatRaw
	}

	entryRaw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if providers == nil {
		providers = map[string]json.RawMessage{}
	}
	providers[id] = entryRaw
	obj["providers"] = mustRaw(providers)
	return writeModelsJSON(path, obj)
}

// RemoveCustomProvider deletes one definition. It refuses ids it does not
// own (built-ins, absent entries) so the GUI cannot erase a hand-written
// override by guessing a name.
func RemoveCustomProvider(id string) error {
	id = strings.TrimSpace(id)
	if err := ValidateCustomID(id); err != nil {
		return err
	}
	path := ModelsPath()
	if path == "" {
		return fmt.Errorf("no home directory")
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("%s is not a custom provider", id)
	} else if err != nil {
		return err
	}
	providers := readProviders(raw)
	entry, ok := providers[id]
	if !ok {
		return fmt.Errorf("%s is not a custom provider", id)
	}
	var probe struct {
		BaseURL string `json:"baseUrl"`
	}
	if json.Unmarshal(entry, &probe) != nil || probe.BaseURL == "" {
		// Built-in override or unparseable entry: not ours to delete.
		return fmt.Errorf("%s is not a custom provider", id)
	}
	delete(providers, id)
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}
	obj["providers"] = mustRaw(providers)
	return writeModelsJSON(path, obj)
}

// readModelsTop reads the whole file as a raw JSON object, preserving every
// top-level key pi defines now or adds later.
func readModelsTop(path string) (map[string]json.RawMessage, error) {
	obj := map[string]json.RawMessage{}
	raw, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("models.json is not valid JSON: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return obj, nil
}

func validateCustomDef(def CustomDefinition) error {
	u := strings.TrimSpace(def.BaseURL)
	if u == "" {
		return fmt.Errorf("base URL required")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return fmt.Errorf("base URL must start with http:// or https://")
	}
	if len(u) > 2048 {
		return fmt.Errorf("base URL is too long")
	}
	apiOK := false
	for _, api := range CustomAPIs {
		if def.API == api {
			apiOK = true
			break
		}
	}
	if !apiOK {
		return fmt.Errorf("unsupported API type")
	}
	if len(def.Models) == 0 {
		return fmt.Errorf("at least one model id is required")
	}
	seen := map[string]bool{}
	for _, m := range def.Models {
		id := strings.TrimSpace(m.ID)
		if id == "" || strings.ContainsAny(id, " \t\r\n") {
			return fmt.Errorf("model ids cannot be empty or contain spaces")
		}
		if len(id) > 256 {
			return fmt.Errorf("model id is too long")
		}
		if seen[id] {
			return fmt.Errorf("model %s is listed twice", id)
		}
		seen[id] = true
		if m.ContextWindow != nil && *m.ContextWindow <= 0 {
			return fmt.Errorf("context window must be a positive number")
		}
		if m.MaxTokens != nil && *m.MaxTokens <= 0 {
			return fmt.Errorf("max output must be a positive number")
		}
	}
	return nil
}

func writeModelsJSON(path string, obj map[string]json.RawMessage) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}

func mustRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("catalog: marshal constant: " + err.Error())
	}
	return b
}
