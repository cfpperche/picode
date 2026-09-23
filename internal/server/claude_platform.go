package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Claude Code on a third-party platform — Amazon Bedrock, Google Vertex AI,
// Microsoft Foundry — set up from PiCode's GUI (ADR-0189). Claude Code's own
// /login wizard stores that choice as an `env` block in the user settings
// (~/.claude/settings.json): the platform's switch and its values set, every
// other platform's keys removed (read from 2.1.280's wizard). PiCode writes
// the same block, so the choice is Claude Code's own and holds in a terminal
// PiCode never opened; the file is the truth, as .credentials.json is for the
// subscription. A platform outranks the subscription and a Console key
// (measured: "dispatching to bedrock" with ANTHROPIC_API_KEY also set), so
// choosing one ends the others' turn and choosing either of those removes it.

// claudePlatformKeys are the env keys a platform setup owns — the union of
// what the wizard sets or clears — removed before a platform is written and
// when Claude Code goes back to the subscription or a key.
var claudePlatformKeys = []string{
	"CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY",
	"CLAUDE_CODE_USE_ANTHROPIC_AWS", "CLAUDE_CODE_USE_ANTHROPIC_GOOGLE_CLOUD", "CLAUDE_CODE_USE_MANTLE",
	"AWS_REGION", "AWS_PROFILE", "AWS_BEARER_TOKEN_BEDROCK", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
	"ANTHROPIC_VERTEX_PROJECT_ID", "CLOUD_ML_REGION", "GOOGLE_APPLICATION_CREDENTIALS",
	"ANTHROPIC_FOUNDRY_RESOURCE", "ANTHROPIC_FOUNDRY_BASE_URL", "ANTHROPIC_FOUNDRY_API_KEY",
	// Model pins name one platform's model ids; the wizard clears them too.
	"ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_DEFAULT_FABLE_MODEL", "ANTHROPIC_SMALL_FAST_MODEL",
}

var claudePlatformSwitch = map[string]string{
	"bedrock": "CLAUDE_CODE_USE_BEDROCK",
	"vertex":  "CLAUDE_CODE_USE_VERTEX",
	"foundry": "CLAUDE_CODE_USE_FOUNDRY",
}

var claudePlatformNames = map[string]string{
	"bedrock": "Amazon Bedrock",
	"vertex":  "Google Vertex AI",
	"foundry": "Microsoft Foundry",
	"gateway": "a custom gateway",
}

// claudeGatewayKeys are an Anthropic-compatible gateway's own keys
// (ADR-0190). Unlike a platform's, they are removed only when a gateway is the
// login in use or is being written: an ANTHROPIC_BASE_URL someone set for a
// corporate proxy, with no token beside it, is theirs and stays.
var claudeGatewayKeys = []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_MODEL"}

// claudePlatformRequest is the form the dialog sends. Secrets travel once,
// on save, and are never read back.
type claudePlatformRequest struct {
	Kind            string `json:"kind"`
	Auth            string `json:"auth"`
	Region          string `json:"region"`
	Profile         string `json:"profile"`
	BearerToken     string `json:"bearerToken"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	Project         string `json:"project"`
	KeyFile         string `json:"keyFile"`
	Resource        string `json:"resource"`
	APIKey          string `json:"apiKey"`
	BaseURL         string `json:"baseUrl"`
	AuthToken       string `json:"authToken"`
	Model           string `json:"model"`
	FastModel       string `json:"fastModel"`
}

// claudePlatformView is what the roster says about the platform in use —
// never a secret.
type claudePlatformView struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Auth     string `json:"auth"`
	Region   string `json:"region,omitempty"`
	Profile  string `json:"profile,omitempty"`
	Project  string `json:"project,omitempty"`
	Resource string `json:"resource,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
	// A gateway's address and models; its token is never read back.
	BaseURL   string `json:"baseUrl,omitempty"`
	Model     string `json:"model,omitempty"`
	FastModel string `json:"fastModel,omitempty"`
}

var (
	claudeRegionRe   = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)
	claudeResourceRe = regexp.MustCompile(`^[A-Za-z0-9-]{2,64}$`)
)

// env turns the form into the settings env block, the wizard's shape.
func (r claudePlatformRequest) env() (map[string]string, error) {
	t := func(s string) string { return strings.TrimSpace(s) }
	out := map[string]string{}
	switch r.Kind {
	case "bedrock":
		if !claudeRegionRe.MatchString(t(r.Region)) {
			return nil, errors.New("An AWS region is required, like us-east-1.")
		}
		out["AWS_REGION"] = t(r.Region)
		switch r.Auth {
		case "bearer":
			if t(r.BearerToken) == "" {
				return nil, errors.New("A Bedrock API key is required.")
			}
			out["AWS_BEARER_TOKEN_BEDROCK"] = t(r.BearerToken)
		case "profile":
			if t(r.Profile) == "" {
				return nil, errors.New("An AWS profile name is required.")
			}
			out["AWS_PROFILE"] = t(r.Profile)
		case "accessKey":
			if t(r.AccessKeyID) == "" || t(r.SecretAccessKey) == "" {
				return nil, errors.New("An access key ID and its secret are required.")
			}
			out["AWS_ACCESS_KEY_ID"] = t(r.AccessKeyID)
			out["AWS_SECRET_ACCESS_KEY"] = t(r.SecretAccessKey)
			if t(r.SessionToken) != "" {
				out["AWS_SESSION_TOKEN"] = t(r.SessionToken)
			}
		case "environment":
		default:
			return nil, errors.New("Pick how Claude Code signs in to AWS.")
		}
	case "vertex":
		if t(r.Project) == "" {
			return nil, errors.New("A Google Cloud project ID is required.")
		}
		if !claudeRegionRe.MatchString(t(r.Region)) {
			return nil, errors.New("A region is required, like us-east5 or global.")
		}
		out["ANTHROPIC_VERTEX_PROJECT_ID"] = t(r.Project)
		out["CLOUD_ML_REGION"] = t(r.Region)
		switch r.Auth {
		case "serviceAccount":
			if t(r.KeyFile) == "" || !filepath.IsAbs(t(r.KeyFile)) {
				return nil, errors.New("The service account key file needs its full path.")
			}
			out["GOOGLE_APPLICATION_CREDENTIALS"] = t(r.KeyFile)
		case "adc", "environment":
		default:
			return nil, errors.New("Pick how Claude Code signs in to Google Cloud.")
		}
	case "foundry":
		if !claudeResourceRe.MatchString(t(r.Resource)) {
			return nil, errors.New("The Foundry resource name is required (letters, digits and dashes).")
		}
		out["ANTHROPIC_FOUNDRY_RESOURCE"] = t(r.Resource)
		switch r.Auth {
		case "apiKey":
			if t(r.APIKey) == "" {
				return nil, errors.New("A Foundry API key is required.")
			}
			out["ANTHROPIC_FOUNDRY_API_KEY"] = t(r.APIKey)
		case "environment":
		default:
			return nil, errors.New("Pick how Claude Code signs in to Microsoft Foundry.")
		}
	case "gateway":
		u, err := url.Parse(t(r.BaseURL))
		local := err == nil && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
		if err != nil || u.Host == "" || !(u.Scheme == "https" || (u.Scheme == "http" && local)) {
			return nil, errors.New("The gateway URL must start with https:// (http:// only for this machine).")
		}
		if t(r.AuthToken) == "" {
			return nil, errors.New("The gateway's token is required.")
		}
		out["ANTHROPIC_BASE_URL"] = t(r.BaseURL)
		out["ANTHROPIC_AUTH_TOKEN"] = t(r.AuthToken)
		if t(r.Model) != "" {
			out["ANTHROPIC_MODEL"] = t(r.Model)
		}
		if t(r.FastModel) != "" {
			out["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = t(r.FastModel)
		}
		// A gateway has no switch of its own: its URL and token are it.
		return out, nil
	default:
		return nil, errors.New("Unknown platform.")
	}
	out[claudePlatformSwitch[r.Kind]] = "1"
	return out, nil
}

// claudeUserSettingsPath is Claude Code's user settings file.
func claudeUserSettingsPath() string {
	if dir := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); dir != "" {
		return filepath.Join(dir, "settings.json")
	}
	path, err := claudeSettingsPath()
	if err != nil {
		return ""
	}
	return path
}

// readClaudeSettings decodes the user settings, keeping every field; an
// absent or blank file is the empty document, one that does not decode is
// refused rather than overwritten.
func readClaudeSettings(path string) (map[string]any, os.FileMode, error) {
	doc := map[string]any{}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return doc, 0o600, nil
	}
	if err != nil {
		return nil, 0, err
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if strings.TrimSpace(string(raw)) == "" {
		return doc, mode, nil
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, 0, errors.New("Claude Code's settings file is not JSON PiCode can edit safely")
	}
	return doc, mode, nil
}

func settingsEnv(doc map[string]any) map[string]any {
	env, _ := doc["env"].(map[string]any)
	return env
}

func truthyEnv(v any) bool {
	s, _ := v.(string)
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes"
}

// claudePlatformInUse reads the platform Claude Code is set to, if any.
func claudePlatformInUse() (claudePlatformView, bool) {
	path := claudeUserSettingsPath()
	if path == "" {
		return claudePlatformView{}, false
	}
	doc, _, err := readClaudeSettings(path)
	if err != nil {
		return claudePlatformView{}, false
	}
	env := settingsEnv(doc)
	str := func(k string) string { s, _ := env[k].(string); return strings.TrimSpace(s) }
	for _, kind := range []string{"bedrock", "vertex", "foundry"} {
		if !truthyEnv(env[claudePlatformSwitch[kind]]) {
			continue
		}
		v := claudePlatformView{Kind: kind, Name: claudePlatformNames[kind]}
		switch kind {
		case "bedrock":
			v.Region = str("AWS_REGION")
			switch {
			case str("AWS_BEARER_TOKEN_BEDROCK") != "":
				v.Auth = "bearer"
			case str("AWS_PROFILE") != "":
				v.Auth, v.Profile = "profile", str("AWS_PROFILE")
			case str("AWS_ACCESS_KEY_ID") != "":
				v.Auth = "accessKey"
			default:
				v.Auth = "environment"
			}
		case "vertex":
			v.Project, v.Region = str("ANTHROPIC_VERTEX_PROJECT_ID"), str("CLOUD_ML_REGION")
			v.Auth = "adc"
			if kf := str("GOOGLE_APPLICATION_CREDENTIALS"); kf != "" {
				v.Auth, v.KeyFile = "serviceAccount", kf
			}
		case "foundry":
			v.Resource = str("ANTHROPIC_FOUNDRY_RESOURCE")
			v.Auth = "environment"
			if str("ANTHROPIC_FOUNDRY_API_KEY") != "" {
				v.Auth = "apiKey"
			}
		}
		return v, true
	}
	// A gateway is a URL with a token beside it (Claude Code's own LLM
	// gateway setup); a URL alone is a proxy, not a login.
	if str("ANTHROPIC_BASE_URL") != "" && str("ANTHROPIC_AUTH_TOKEN") != "" {
		v := claudePlatformView{Kind: "gateway", Name: claudePlatformNames["gateway"], Auth: "token",
			BaseURL: str("ANTHROPIC_BASE_URL"), Model: str("ANTHROPIC_MODEL"), FastModel: str("ANTHROPIC_DEFAULT_HAIKU_MODEL")}
		return v, true
	}
	return claudePlatformView{}, false
}

// writeClaudePlatformEnv replaces the platform keys of the settings env with
// set (nil removes the platform). Every other field, and every env key the
// platform does not own, is kept. Nothing is written when no platform key is
// there to remove and none is being set.
func writeClaudePlatformEnv(set map[string]string) error {
	path := claudeUserSettingsPath()
	if path == "" {
		return errors.New("PiCode cannot find Claude Code's settings file")
	}
	doc, mode, err := readClaudeSettings(path)
	if err != nil {
		return err
	}
	env := settingsEnv(doc)
	changed := false
	if env == nil {
		env = map[string]any{}
	}
	owned := claudePlatformKeys
	current, inUse := claudePlatformInUse()
	if (inUse && current.Kind == "gateway") || set["ANTHROPIC_BASE_URL"] != "" {
		owned = append(append([]string{}, claudePlatformKeys...), claudeGatewayKeys...)
	}
	for _, k := range owned {
		if _, ok := env[k]; ok {
			delete(env, k)
			changed = true
		}
	}
	for k, v := range set {
		env[k] = v
		changed = true
	}
	if !changed {
		return nil
	}
	if len(env) > 0 {
		doc["env"] = env
	} else {
		delete(doc, "env")
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeInterceptFile(path, append(out, '\n'), mode)
}

// clearClaudePlatform ends a platform's turn: the subscription or a Console
// key was chosen, and a platform left in the settings would outrank it.
func clearClaudePlatform() error {
	if _, ok := claudePlatformInUse(); !ok {
		return nil
	}
	return writeClaudePlatformEnv(nil)
}

func handleClaudePlatformPut(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req claudePlatformRequest
		if !readCLIJSON(w, r, &req) {
			return
		}
		env, err := req.env()
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := writeClaudePlatformEnv(env); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := setClaudeKeyInUse(deps, "", ""); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		view, _ := claudePlatformInUse()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "platform": view})
	}
}

func handleClaudePlatformDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := writeClaudePlatformEnv(nil); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
