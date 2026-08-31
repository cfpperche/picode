package catalog

import (
	"testing"
)

func TestQuotaKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id, auth, want string
	}{
		{"anthropic", "oauth", "oauth"},
		{"anthropic", "api_key", ""},
		{"xai", "oauth", "oauth"},
		{"xai", "api_key", ""},
		{"openai-codex", "oauth", "oauth"},
		{"github-copilot", "oauth", "oauth"},
		{"kimi-coding", "oauth", "oauth"},
		{"kimi-coding", "api_key", "api_key"},
		{"openai", "api_key", ""},
		{"llama.cpp", "api_key", ""},
		{"openrouter", "oauth", ""},
		{"openrouter", "api_key", "api_key"},
		{"minimax", "api_key", "api_key"},
		{"minimax-cn", "api_key", "api_key"},
		{"qwen-token-plan", "api_key", ""},
		{"zai", "api_key", "api_key"},
		{"zai", "oauth", ""},
		{"zai-coding-cn", "api_key", "api_key"},
		{"opencode-go", "api_key", "api_key"},
		{"", "oauth", ""},
	}
	for _, c := range cases {
		if got := QuotaKind(c.id, c.auth); got != c.want {
			t.Errorf("QuotaKind(%q,%q)=%q want %q", c.id, c.auth, got, c.want)
		}
	}
}

func TestActiveOAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, ok := ActiveOAuth("anthropic"); ok {
		t.Fatal("empty")
	}
	if err := PutAPIKey("anthropic", "sk-test"); err != nil {
		t.Fatal(err)
	}
	if _, ok := ActiveOAuth("anthropic"); ok {
		t.Fatal("api key is not oauth")
	}
	if ActiveAuthType("anthropic") != LoginAPIKey {
		t.Fatalf("auth %s", ActiveAuthType("anthropic"))
	}
	if err := PutOAuth("anthropic", map[string]any{
		"type": "oauth", "access": "a1", "refresh": "r1", "expires": float64(9),
	}); err != nil {
		t.Fatal(err)
	}
	cred, ok := ActiveOAuth("anthropic")
	if !ok || cred.Access != "a1" || cred.Refresh != "r1" || cred.Expires != 9 {
		t.Fatalf("cred %+v ok=%v", cred, ok)
	}
	if QuotaKind("anthropic", ActiveAuthType("anthropic")) != LoginOAuth {
		t.Fatal("quota kind")
	}
	if ActiveLabel("anthropic") == "" {
		t.Fatal("label")
	}
}
