package catalog

import (
	"os"
	"strings"
)

// Credential sources a provider row can have. Source is what pi would use,
// not where PiCode would like it to be (ADR-0013: auth.json is pi's slot).
const (
	SourceVault = "vault"       // ~/.picode/accounts.json + auth.json
	SourceEnv   = "environment" // an API-key env var in the daemon's environment
)

// APIKeyEnvVars mirrors pi's getApiKeyEnvVars, read from the installed
// bundle (@earendil-works/pi-coding-agent, dist/bundle) on 2026-09-03.
// A provider missing here has no env path in pi. github-copilot and
// anthropic are special-cased below, exactly as pi does.
var APIKeyEnvVars = map[string][]string{
	"anthropic":                  {"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_API_KEY"},
	"github-copilot":             {"COPILOT_GITHUB_TOKEN"},
	"ant-ling":                   {"ANT_LING_API_KEY"},
	"qwen-token-plan":            {"QWEN_TOKEN_PLAN_API_KEY"},
	"qwen-token-plan-cn":         {"QWEN_TOKEN_PLAN_CN_API_KEY"},
	"qwen-token-plan-individual": {"QWEN_TOKEN_PLAN_API_KEY"},
	"openai":                     {"OPENAI_API_KEY"},
	"azure-openai-responses":     {"AZURE_OPENAI_API_KEY"},
	"nvidia":                     {"NVIDIA_API_KEY"},
	"deepseek":                   {"DEEPSEEK_API_KEY"},
	"google":                     {"GEMINI_API_KEY"},
	"google-vertex":              {"GOOGLE_CLOUD_API_KEY"},
	"groq":                       {"GROQ_API_KEY"},
	"cerebras":                   {"CEREBRAS_API_KEY"},
	"xai":                        {"XAI_API_KEY"},
	"radius":                     {"RADIUS_API_KEY"},
	"openrouter":                 {"OPENROUTER_API_KEY"},
	"vercel-ai-gateway":          {"AI_GATEWAY_API_KEY"},
	"zai":                        {"ZAI_API_KEY"},
	"zai-coding-cn":              {"ZAI_CODING_CN_API_KEY"},
	"mistral":                    {"MISTRAL_API_KEY"},
	"minimax":                    {"MINIMAX_API_KEY"},
	"minimax-cn":                 {"MINIMAX_CN_API_KEY"},
	"moonshotai":                 {"MOONSHOT_API_KEY"},
	"moonshotai-cn":              {"MOONSHOT_API_KEY"},
	"huggingface":                {"HF_TOKEN"},
	"fireworks":                  {"FIREWORKS_API_KEY"},
	"together":                   {"TOGETHER_API_KEY"},
	"baseten":                    {"BASETEN_API_KEY"},
	"opencode":                   {"OPENCODE_API_KEY"},
	"opencode-go":                {"OPENCODE_API_KEY"},
	"kimi-coding":                {"KIMI_API_KEY"},
	"cloudflare-workers-ai":      {"CLOUDFLARE_API_KEY"},
	"cloudflare-ai-gateway":      {"CLOUDFLARE_API_KEY"},
	"xiaomi":                     {"XIAOMI_API_KEY"},
	"xiaomi-token-plan-cn":       {"XIAOMI_TOKEN_PLAN_CN_API_KEY"},
	"xiaomi-token-plan-ams":      {"XIAOMI_TOKEN_PLAN_AMS_API_KEY"},
	"xiaomi-token-plan-sgp":      {"XIAOMI_TOKEN_PLAN_SGP_API_KEY"},
}

// LookupEnv is os.LookupEnv; tests replace it.
var LookupEnv = os.LookupEnv

// EnvKeyName returns the env var currently supplying this provider, and its
// value. Measured against pi v0.84.4: with only GROQ_API_KEY set,
// `pi auth check --provider groq --json` answers ready/api_key — so a
// provider with no auth.json entry is still usable by every agent we spawn.
func EnvKeyName(provider string) (string, string, bool) {
	for _, name := range APIKeyEnvVars[strings.ToLower(strings.TrimSpace(provider))] {
		if v, ok := LookupEnv(name); ok && strings.TrimSpace(v) != "" {
			return name, strings.TrimSpace(v), true
		}
	}
	return "", "", false
}

// EnvAPIKey is the key an env var supplies, if any. Never logged.
func EnvAPIKey(provider string) (string, bool) {
	_, v, ok := EnvKeyName(provider)
	return v, ok
}
