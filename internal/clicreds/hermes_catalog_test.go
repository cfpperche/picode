package clicreds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const hermesCatalogFixture = `[
 {"id":"nous","name":"Nous Portal","auth_type":"oauth_device_code","env":[]},
 {"id":"openai-codex","name":"OpenAI Codex","auth_type":"oauth_external","env":[]},
 {"id":"qwen-oauth","name":"Qwen OAuth","auth_type":"oauth_external","env":[]},
 {"id":"deepseek","name":"DeepSeek","auth_type":"api_key","env":["DEEPSEEK_API_KEY"]},
 {"id":"alibaba-cn","name":"Alibaba Cloud DashScope (China)","auth_type":"api_key","env":["DASHSCOPE_API_KEY"]},
 {"id":"dashscope-cn","name":"Alibaba Cloud DashScope (China)","auth_type":"api_key","env":["DASHSCOPE_API_KEY"]},
 {"id":"bedrock","name":"AWS Bedrock","auth_type":"aws_sdk","env":[]},
 {"id":"gemini","name":"Google AI Studio","auth_type":"api_key","env":["GOOGLE_API_KEY"]}
]`

func TestParseHermesCatalog(t *testing.T) {
	got := parseHermesCatalog([]byte(hermesCatalogFixture), []Provider{{Provider: "openai-codex"}})
	var ids []string
	for _, p := range got {
		ids = append(ids, p.Provider)
	}
	// Declared ids and their Hermes forms (openai-codex, gemini), an alias
	// (dashscope-cn), a door PiCode cannot drive (bedrock) and Qwen's login
	// that needs its own CLI are left out.
	if strings.Join(ids, ",") != "nous,deepseek,alibaba-cn" {
		t.Fatalf("ids = %v", ids)
	}
	if got[0].Kinds[0] != KindOAuth || got[0].Note == "" {
		t.Fatalf("nous = %+v", got[0])
	}
	if got[1].Kinds[0] != KindAPIKey || got[1].Env[KindAPIKey] != "DEEPSEEK_API_KEY" || got[1].Name != "DeepSeek" || got[1].Native == nil {
		t.Fatalf("deepseek = %+v", got[1])
	}
	if parseHermesCatalog([]byte("nope"), nil) != nil {
		t.Fatal("unreadable catalog must add nothing")
	}
}

func TestForHermesAppendsItsCatalog(t *testing.T) {
	t.Cleanup(resetHermesCatalog)
	path := filepath.Join(t.TempDir(), "hermes.json")
	if err := os.WriteFile(path, []byte(hermesCatalogFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(hermesCatalogEnv, path)
	resetHermesCatalog()
	spec, _ := For("hermes")
	if name, ok := EnvVar("hermes", "deepseek", KindAPIKey); !ok || name != "DEEPSEEK_API_KEY" {
		t.Fatalf("deepseek env = %q %v (providers %d)", name, ok, len(spec.Providers))
	}
	if _, ok := CanUse("omp", "nous"); ok {
		t.Fatal("omp must not gain Hermes's providers")
	}
}

func TestHermesAddID(t *testing.T) {
	cases := []struct{ provider, kind, want string }{
		{"google", KindAPIKey, "gemini"},
		{"github-copilot", KindAPIKey, "copilot"},
		{"opencode", KindAPIKey, "opencode-zen"},
		{"xai", KindAPIKey, "xai"},
		{"xai", KindOAuth, "xai-oauth"},
		{"deepseek", KindAPIKey, "deepseek"},
	}
	for _, tc := range cases {
		if got := HermesAddID(tc.provider, tc.kind); got != tc.want {
			t.Fatalf("HermesAddID(%s,%s) = %s, want %s", tc.provider, tc.kind, got, tc.want)
		}
	}
}

// Hermes 0.21 keeps a pooled key in access_token (measured): the reader
// finds it there, and in key for older pools.
func TestParseHermesPoolKeyField(t *testing.T) {
	for _, field := range []string{"access_token", "key"} {
		raw := `{"credential_pool":{"deepseek":[{"auth_type":"api_key","` + field + `":"sk-pool-key"}]}}`
		login, ok := parseHermes([]byte(raw), "deepseek")
		if !ok || !strings.Contains(string(login.Cred), "sk-pool-key") {
			t.Fatalf("%s: login = %+v, %v", field, login, ok)
		}
	}
}

// TestHermesInterpreterFromWrapper: the `hermes` on PATH is the installer's
// wrapper; its exec line leads to the venv, whose python is the interpreter.
func TestHermesInterpreterFromWrapper(t *testing.T) {
	root := t.TempDir()
	venvBin := filepath.Join(root, "install", "hermes-agent", "venv", "bin")
	if err := os.MkdirAll(venvBin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"python", "hermes"} {
		if err := os.WriteFile(filepath.Join(venvBin, f), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	pathDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(pathDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wrapper := "#!/usr/bin/env bash\nunset PYTHONPATH\nexec \"" + filepath.Join(venvBin, "hermes") + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(pathDir, "hermes"), []byte(wrapper), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pathDir)
	t.Setenv("HERMES_HOME", filepath.Join(root, "elsewhere"))
	py, dir := hermesInterpreter()
	if py != filepath.Join(venvBin, "python") || dir != filepath.Join(root, "install", "hermes-agent") {
		t.Fatalf("interpreter = %q in %q", py, dir)
	}
}
