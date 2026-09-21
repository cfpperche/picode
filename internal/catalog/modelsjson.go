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
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
var customCompatKeys = []string{"supportsDeveloperRole", "supportsReasoningEffort", "supportsUsageInStreaming"}

// customThinkingLevels are the pi thinking levels the form manages. "off" is
// deliberately absent: pi's default map already covers it and its provider
// value is not always the level name, so the form never invents one.
var customThinkingLevels = []string{"minimal", "low", "medium", "high", "xhigh", "max"}

// CustomModel is one model row from the form. Pointers distinguish "leave
// pi's default" from zero; unknown model fields are preserved on merge.
type CustomModel struct {
	ID            string `json:"id"`
	Reasoning     *bool  `json:"reasoning,omitempty"`
	ContextWindow *int   `json:"contextWindow,omitempty"`
	MaxTokens     *int   `json:"maxTokens,omitempty"`
	// Name, Input and Cost are form-managed row fields: a value writes it, an
	// empty one deletes the stored key, so clearing a hand-set value in the
	// form removes it instead of hiding it. The form always sends the full
	// row, so blank means delete; a row missing from Models drops the whole
	// model. Cost is USD per 1M tokens (pi-ai models.js divides rates by
	// 1e6).
	Name  string      `json:"name,omitempty"`
	Input []string    `json:"input,omitempty"`
	Cost  *CustomCost `json:"cost,omitempty"`
	// ThinkingLevels is the form's level selection; upsert turns it into a
	// thinkingLevelMap on the entry (selected levels keep their name, the rest
	// become null = hidden). ThinkingLevelMap is read back for the form's
	// prefill and is never written from the request.
	ThinkingLevels   []string       `json:"thinkingLevels,omitempty"`
	ThinkingLevelMap map[string]any `json:"thinkingLevelMap,omitempty"`
	// ThinkingLevelValues overrides the provider value per selected level: a
	// blank value keeps the level's own name (xhigh), a filled one writes it
	// (xhigh -> "high"). Keys outside the managed levels are refused.
	ThinkingLevelValues map[string]string `json:"thinkingLevelValues,omitempty"`
}

// CustomCost is pi's ModelCostRates: USD per 1M tokens.
type CustomCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// customThinkingFormats are the compat.thinkingFormat values the form offers:
// pi's own union (pi-ai types.d.ts), minus the formats that only make sense on
// a built-in provider. The form writes the field itself, so an unknown value
// is refused here rather than handed to pi.
var customThinkingFormats = []string{
	"openai", "openrouter", "deepseek", "together", "baseten", "zai", "qwen",
	"chat-template", "qwen-chat-template", "string-thinking", "ant-ling",
}

// chatTemplateVars are the pi-controlled references a template object value
// may use instead of a literal.
var chatTemplateVars = []string{"thinking.enabled", "thinking.effort", "thinking.budget"}

// validateTemplateObject mirrors the browser's check for chatTemplateKwargs /
// chatTemplateArgs: values are a string, number, boolean or null, or a
// {"$var": …} reference with an optional omitWhenOff. The GUI is not the only
// guard, and a bad object would make pi send a field the provider rejects.
func validateTemplateObject(key string, obj map[string]any) error {
	for name, value := range obj {
		switch v := value.(type) {
		case nil, string, float64, bool:
			continue
		case map[string]any:
			for other := range v {
				if other != "$var" && other != "omitWhenOff" {
					return fmt.Errorf("%s.%s: only $var and omitWhenOff may sit beside each other", key, name)
				}
			}
			ref, ok := v["$var"].(string)
			if !ok || !slices.Contains(chatTemplateVars, ref) {
				return fmt.Errorf("%s.%s: $var must be one of %s", key, name, strings.Join(chatTemplateVars, ", "))
			}
			if raw, ok := v["omitWhenOff"]; ok {
				if _, isBool := raw.(bool); !isBool {
					return fmt.Errorf("%s.%s: omitWhenOff is true or false", key, name)
				}
			}
		default:
			return fmt.Errorf("%s.%s: use a string, number, true/false, null or a {\"$var\": …} reference", key, name)
		}
	}
	return nil
}

// CustomDefinition is the editable shape of one models.json provider entry.
// ThinkingFormat is compat.thinkingFormat: a string among the others there,
// which is why compat is not a map[string]bool on this struct. The two chat
// template objects are compat's only nested values (pi's
// chatTemplateKwargs / chatTemplateArgs).
type CustomDefinition struct {
	BaseURL            string          `json:"baseUrl"`
	API                string          `json:"api"`
	Compat             map[string]bool `json:"compat,omitempty"`
	ThinkingFormat     string          `json:"thinkingFormat,omitempty"`
	ChatTemplateKwargs map[string]any  `json:"chatTemplateKwargs,omitempty"`
	ChatTemplateArgs   map[string]any  `json:"chatTemplateArgs,omitempty"`
	Models             []CustomModel   `json:"models"`
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
	if err := validateCustomIDShape(id); err != nil {
		return err
	}
	if _, builtin := LoginMethods[strings.TrimSpace(id)]; builtin {
		return fmt.Errorf("%s is a built-in provider; pick another name", strings.TrimSpace(id))
	}
	return nil
}

// validateCustomIDShape is the id grammar both writers share: non-empty,
// lowercase slug (customIDPattern). The built-in sets differ per CLI.
func validateCustomIDShape(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("name required")
	}
	if !customIDPattern.MatchString(id) {
		return fmt.Errorf("name must be lowercase letters, digits and dashes")
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
		// compat holds bools (the two the form manages) and strings
		// (thinkingFormat), so it is decoded key by key: typing the whole
		// object as map[string]bool made one string key fail the entry and
		// the provider vanish from the catalog.
		var entry struct {
			BaseURL string                     `json:"baseUrl"`
			API     string                     `json:"api"`
			Compat  map[string]json.RawMessage `json:"compat"`
			Models  []CustomModel              `json:"models"`
		}
		if json.Unmarshal(entryRaw, &entry) != nil {
			continue
		}
		compat := map[string]bool{}
		for _, key := range customCompatKeys {
			var v bool
			if raw, ok := entry.Compat[key]; ok && json.Unmarshal(raw, &v) == nil {
				compat[key] = v
			}
		}
		thinkingFormat := ""
		if raw, ok := entry.Compat["thinkingFormat"]; ok {
			_ = json.Unmarshal(raw, &thinkingFormat)
		}
		kwargs := compatObject(entry.Compat, "chatTemplateKwargs")
		args := compatObject(entry.Compat, "chatTemplateArgs")
		if entry.BaseURL == "" {
			continue
		}
		if entry.API == "" {
			entry.API = APIOpenAICompletions
		}
		out[id] = CustomDefinition{
			BaseURL: entry.BaseURL, API: entry.API, Compat: compat,
			ThinkingFormat: thinkingFormat, ChatTemplateKwargs: kwargs, ChatTemplateArgs: args,
			Models: entry.Models,
		}
	}
	return out
}

// compatObject reads one nested compat value (the chat template objects). A
// value that is not an object reads as absent, so a hand-written string there
// cannot break the row.
func compatObject(compat map[string]json.RawMessage, key string) map[string]any {
	raw, ok := compat[key]
	if !ok {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		return nil
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
	// A file pi rejects (entries outside the providers wrapper) stays pi's
	// rejection even after our merge — writing into it would report success
	// while the gateway never loads. Refuse and name the fix instead.
	for key := range obj {
		if key != "providers" {
			return fmt.Errorf("models.json has hand-written entries outside the providers wrapper; move them under \"providers\" first")
		}
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
		// Name, input and cost are form-managed: a value writes it, a blank
		// deletes the stored key, so the form owns what it shows.
		if name := strings.TrimSpace(m.Name); name != "" {
			fields["name"] = mustRaw(name)
		} else {
			delete(fields, "name")
		}
		if len(m.Input) > 0 {
			fields["input"] = mustRaw(canonicalInput(m.Input))
		} else {
			delete(fields, "input")
		}
		if m.Cost != nil {
			fields["cost"] = mustRaw(*m.Cost)
		} else {
			delete(fields, "cost")
		}
		// Levels are written as a map so pi can hide what the model lacks.
		// A model the form marks as non-reasoning drops the managed levels
		// again; unknown map keys (a hand-set "off") always survive.
		if err := mergeThinkingLevels(fields, m); err != nil {
			return err
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
	// when the form sent no compat at all (Edit always sends both keys). The
	// block also runs for a lone thinkingFormat, so an API caller can set the
	// format without sending the bool map.
	if def.Compat != nil || def.ThinkingFormat != "" || def.ChatTemplateKwargs != nil || def.ChatTemplateArgs != nil {
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
		// thinkingFormat is a string among the bools: empty means pi's own
		// default for the API type, so the key is removed instead of written
		// as an empty string. A caller who sent no compat at all keeps a
		// hand-set format.
		if def.ThinkingFormat != "" {
			compat["thinkingFormat"] = mustRaw(def.ThinkingFormat)
		} else if def.Compat != nil {
			delete(compat, "thinkingFormat")
		}
		// The chat template objects are the form's to own like the bools: sent
		// with a value they are written, sent empty they clear (a stale inert
		// object is the drift that confuses the next reader). A caller who sent
		// no compat at all leaves them alone.
		for _, obj := range []struct {
			key string
			val map[string]any
		}{{"chatTemplateKwargs", def.ChatTemplateKwargs}, {"chatTemplateArgs", def.ChatTemplateArgs}} {
			switch {
			case len(obj.val) > 0:
				compat[obj.key] = mustRaw(obj.val)
			case def.Compat != nil:
				delete(compat, obj.key)
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

// mergeThinkingLevels turns the form's level selection into a
// thinkingLevelMap. Selected levels keep their own name as the provider
// value unless ThinkingLevelValues overrides it (xhigh -> "high");
// unselected managed levels become null, which pi reads as unsupported and
// hides. Keys the form does not manage (a hand-set "off", a future pi
// level) are preserved untouched.
func mergeThinkingLevels(fields map[string]json.RawMessage, m CustomModel) error {
	if len(m.ThinkingLevels) == 0 {
		if m.Reasoning != nil && !*m.Reasoning {
			return dropThinkingLevels(fields)
		}
		return nil
	}
	selected := map[string]bool{}
	for _, l := range m.ThinkingLevels {
		if !slices.Contains(customThinkingLevels, l) {
			return fmt.Errorf("unsupported thinking level: %s", l)
		}
		selected[l] = true
	}
	levelMap := map[string]json.RawMessage{}
	if raw, ok := fields["thinkingLevelMap"]; ok {
		_ = json.Unmarshal(raw, &levelMap)
	}
	for _, l := range customThinkingLevels {
		if selected[l] {
			if v := strings.TrimSpace(m.ThinkingLevelValues[l]); v != "" {
				levelMap[l] = mustRaw(v)
			} else {
				levelMap[l] = mustRaw(l)
			}
			continue
		}
		levelMap[l] = mustRaw(nil)
	}
	raw, err := json.Marshal(levelMap)
	if err != nil {
		return err
	}
	fields["thinkingLevelMap"] = raw
	return nil
}

// canonicalInput writes input modalities in pi's order (text, image),
// deduplicated. Membership is checked by validateCustomDef.
func canonicalInput(in []string) []string {
	var out []string
	for _, want := range []string{"text", "image"} {
		if slices.Contains(in, want) {
			out = append(out, want)
		}
	}
	return out
}

// dropThinkingLevels removes the levels the form manages, and the whole map
// when nothing else is left, so a model switched back to non-reasoning does
// not keep a stale level list.
func dropThinkingLevels(fields map[string]json.RawMessage) error {
	raw, ok := fields["thinkingLevelMap"]
	if !ok {
		return nil
	}
	levelMap := map[string]json.RawMessage{}
	if json.Unmarshal(raw, &levelMap) != nil {
		return nil
	}
	for _, l := range customThinkingLevels {
		delete(levelMap, l)
	}
	if len(levelMap) == 0 {
		delete(fields, "thinkingLevelMap")
		return nil
	}
	rest, err := json.Marshal(levelMap)
	if err != nil {
		return err
	}
	fields["thinkingLevelMap"] = rest
	return nil
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
	if def.ThinkingFormat != "" && !slices.Contains(customThinkingFormats, def.ThinkingFormat) {
		return fmt.Errorf("unsupported thinking format: %s", def.ThinkingFormat)
	}
	for _, obj := range []struct {
		key string
		val map[string]any
	}{{"chatTemplateKwargs", def.ChatTemplateKwargs}, {"chatTemplateArgs", def.ChatTemplateArgs}} {
		if e := validateTemplateObject(obj.key, obj.val); e != nil {
			return e
		}
	}
	seen := map[string]bool{}
	for _, m := range def.Models {
		id := strings.TrimSpace(m.ID)
		if id == "" || strings.ContainsAny(id, " \t\r\n") {
			return fmt.Errorf("model ids cannot be empty or contain spaces")
		}
		for _, l := range m.ThinkingLevels {
			if !slices.Contains(customThinkingLevels, l) {
				return fmt.Errorf("unsupported thinking level: %s", l)
			}
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
		if len(m.Name) > 120 {
			return fmt.Errorf("model name is too long")
		}
		for _, modality := range m.Input {
			if modality != "text" && modality != "image" {
				return fmt.Errorf("unsupported input modality: %s", modality)
			}
		}
		if m.Cost != nil {
			for _, rate := range []float64{m.Cost.Input, m.Cost.Output, m.Cost.CacheRead, m.Cost.CacheWrite} {
				if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 {
					return fmt.Errorf("cost rates must be zero or above")
				}
			}
		}
		for k, v := range m.ThinkingLevelValues {
			if !slices.Contains(customThinkingLevels, k) {
				return fmt.Errorf("unsupported thinking level: %s", k)
			}
			if vv := strings.TrimSpace(v); vv == "" || len(vv) > 64 || strings.ContainsAny(vv, " \t\r\n") {
				return fmt.Errorf("provider value for %s must be one word", k)
			}
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
