package catalog

// Login method for /login. From pi docs/providers.md. Unknown ids → api_key.
const (
	LoginAPIKey = "api_key"
	LoginOAuth  = "oauth"
	LoginBoth   = "both"
)

// LoginMethods is the TUI /login set. both = account or API key.
var LoginMethods = map[string]string{
	"openai-codex":               LoginOAuth,
	"github-copilot":             LoginOAuth,
	"anthropic":                  LoginBoth,
	"xai":                        LoginBoth,
	"openrouter":                 LoginBoth,
	"radius":                     LoginBoth,
	"openai":                     LoginAPIKey,
	"google":                     LoginAPIKey,
	"deepseek":                   LoginAPIKey,
	"mistral":                    LoginAPIKey,
	"groq":                       LoginAPIKey,
	"cerebras":                   LoginAPIKey,
	"nvidia":                     LoginAPIKey,
	"amazon-bedrock":             LoginAPIKey,
	"huggingface":                LoginAPIKey,
	"fireworks":                  LoginAPIKey,
	"together":                   LoginAPIKey,
	"zai":                        LoginAPIKey,
	"zai-coding-cn":              LoginAPIKey,
	"opencode":                   LoginAPIKey,
	"opencode-go":                LoginAPIKey,
	"vercel-ai-gateway":          LoginAPIKey,
	"ant-ling":                   LoginAPIKey,
	"azure-openai-responses":     LoginAPIKey,
	"cloudflare-ai-gateway":      LoginAPIKey,
	"cloudflare-workers-ai":      LoginAPIKey,
	"baseten":                    LoginAPIKey,
	"kimi-coding":                LoginBoth,
	"minimax":                    LoginAPIKey,
	"minimax-cn":                 LoginAPIKey,
	"qwen-token-plan":            LoginAPIKey,
	"qwen-token-plan-individual": LoginAPIKey,
	"qwen-token-plan-cn":         LoginAPIKey,
	"xiaomi":                     LoginAPIKey,
	"xiaomi-token-plan-cn":       LoginAPIKey,
	"xiaomi-token-plan-ams":      LoginAPIKey,
	"xiaomi-token-plan-sgp":      LoginAPIKey,
	"llama.cpp":                  LoginAPIKey,
}

func loginMethod(id string) string {
	if m, ok := LoginMethods[id]; ok {
		return m
	}
	return LoginAPIKey
}
