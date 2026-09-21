package clicreds

// The declarations: one Spec per CLI PiCode launches, in the order the Agent
// CLIs surface lists them (`internal/clilaunch.Catalog`), so a pane can render
// the rows without sorting them again.
//
// Every provider id is pi's vocabulary (the same id means the same account in
// every pane), and every kind, env var name, path and home variable below was
// read on 2026-09-20 from the CLI installed on this machine or from its own
// docs — pi's `docs/providers.md` and the env map mirrored in
// `internal/catalog/env.go`, the omp docs bundle (`providers.md`), `muse login`
// / `muse auth set --help`, `hermes_cli/providers.py` + `status_auth.py`,
// OpenCode's models.dev cache (`~/.cache/opencode/models.json`) and its
// auth.json, and `docs/benchmarks/2026-09-20-agent-cli-credentials.md` for the
// rest. A row PiCode could not support is not here — an unsupported fact is an
// omission — and the two facts neither the study nor the installed CLIs settled
// are marked "unverified:" inline rather than asserted.
//
// Notes exist because a control that does nothing is worse than no control:
// where a provider row's subscription lives only in the CLI's own store —
// OpenCode, Hermes' credential pool, Muse, Antigravity — the Note says so, and
// where a key loses to a signed-in session (Grok) it says that too.
//
// A Native's CredFile names the credential file inside the CLI's home. The two
// per-account-directory fields — DirEnv (the variable that moves that home) and
// VendorDir (the home relative to $HOME) — are declared only where the CLI's
// own docs name such a variable, because that is what they are for: a CLI
// without one has a store PiCode can read, never a directory PiCode can build.
// Seed and DirEnv go together for the same reason.

// Note texts, one per fact, so a row never paraphrases another.
const (
	noteOwnStore   = "OpenCode keeps this login in its own store; PiCode can read and import it, never inject it."
	noteHermesPool = "Hermes keeps this subscription in its own credential pool; PiCode can read and import it, never inject it."
	noteOmpStore   = "Omp keeps its credentials in a SQLite database PiCode does not read; an import means signing in again."
	noteMuseLogin  = "Muse's Meta account login works only inside Muse; PiCode can read and import it, never inject it."
	noteAgyLogin   = "Antigravity signs in with Google inside the CLI; PiCode can read and import that token, never inject it."
	noteGrokKey    = "A signed-in Grok session wins over this key: sign out in Grok for the key to take effect."
)

// Native stores. The path is the *default* home's credential file; a
// per-account directory holds the same CredFile under the CLI's own DirEnv.
//
// pi, OpenCode and Hermes declare one provider per row but one shared store, so
// the same *Native is referenced by each of their rows: Detect walks the rows
// in declaration order and answers with the first provider the file carries.
// The value is read-only after init.
var (
	nativePi = &Native{
		Path:      "$HOME/.pi/agent/auth.json",
		Format:    "pi",
		DirEnv:    "PI_CODING_AGENT_DIR",
		VendorDir: ".pi/agent",
		CredFile:  "auth.json",
	}
	nativeClaude = &Native{
		Path:      "$HOME/.claude/.credentials.json",
		Format:    "claude",
		DirEnv:    "CLAUDE_CONFIG_DIR",
		VendorDir: ".claude",
		CredFile:  ".credentials.json",
		Seed:      []string{".claude.json", "settings.json", "mcp.json", "plugins", "skills", "projects"},
	}
	nativeCodex = &Native{
		Path:      "$HOME/.codex/auth.json",
		Format:    "codex",
		DirEnv:    "CODEX_HOME",
		VendorDir: ".codex",
		CredFile:  "auth.json",
		Seed:      []string{"config.toml", "skills", "memories"},
	}
	nativeGrok = &Native{
		Path:      "$HOME/.grok/auth.json",
		Format:    "grok",
		DirEnv:    "GROK_HOME",
		VendorDir: ".grok",
		CredFile:  "auth.json",
	}
	nativeHermes = &Native{
		Path:      "$HOME/.hermes/auth.json",
		Format:    "hermes",
		DirEnv:    "HERMES_HOME",
		VendorDir: ".hermes",
		CredFile:  "auth.json",
	}
	// Muse and Antigravity publish no home variable, so there is no per-account
	// directory to declare: read-only rows.
	nativeMuse = &Native{
		Path:     "$HOME/.config/muse/auth.json",
		Format:   "muse",
		CredFile: "auth.json",
	}
	nativeAgy = &Native{
		Path:     "$HOME/.gemini/antigravity-cli/antigravity-oauth-token",
		Format:   "agy",
		CredFile: "antigravity-oauth-token",
	}
	// unverified: OpenCode's credential file moves with the XDG data dir, but
	// whether OPENCODE_CONFIG_DIR relocates it (or only the config files) was
	// not established on this machine — so no DirEnv is declared for it and
	// step 2 has no per-account directory to build here.
	nativeOpencode = &Native{
		Path:     "$XDG_DATA_HOME/opencode/auth.json",
		Format:   "opencode",
		CredFile: "auth.json",
	}
)

// envs pairs kind → variable name; the names are the CLI's own, which is why
// the same provider carries a different variable per CLI (pi's llama.cpp reads
// LLAMA_API_KEY, omp's reads LLAMA_CPP_API_KEY).
func envs(pairs ...string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}

var catalog = []Spec{
	{
		CLI: "pi", Name: "Pi",
		Providers: []Provider{
			// pi's own slot is auth.json (ADR-0013); both env names are the
			// ones its bundle reads for anthropic, in pi's order.
			{Provider: "anthropic", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "ANTHROPIC_API_KEY", KindOAuth, "ANTHROPIC_OAUTH_TOKEN"), Native: nativePi},
			// ChatGPT Plus/Pro: `/login` stores tokens in auth.json; pi has no
			// environment variable for it (catalog/env.go has no entry).
			{Provider: "openai-codex", Kinds: []string{KindOAuth}, Native: nativePi},
			{Provider: "xai", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "XAI_API_KEY"), Native: nativePi},
			{Provider: "google", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "GEMINI_API_KEY"), Native: nativePi},
			// Meta's Muse subscription mints a Model API key; the docs' env
			// name is META_API_KEY while the installed auth.json keys it
			// "meta-ai", which is the id declared here.
			{Provider: "meta-ai", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "META_API_KEY"), Native: nativePi},
			{Provider: "openrouter", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "OPENROUTER_API_KEY"), Native: nativePi},
			{Provider: "github-copilot", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "COPILOT_GITHUB_TOKEN"), Native: nativePi},
			{Provider: "opencode", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "OPENCODE_API_KEY"), Native: nativePi},
			{Provider: "zai", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "ZAI_API_KEY"), Native: nativePi},
			{Provider: "kimi-coding", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "KIMI_API_KEY"), Native: nativePi},
			{Provider: "llama.cpp", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "LLAMA_API_KEY"), Native: nativePi},
		},
		// pi publishes no login subcommand — its `pi auth` only prints stored
		// credentials and checks provider readiness — so its sign-in is
		// `/login` inside its TUI, and the guided flow opens pi itself.
		Login: &LoginCommand{Hint: "In the terminal: finish what it asks, type /login, then come back here."},
	},
	{
		CLI: "claude-code", Name: "Claude Code",
		Providers: []Provider{
			// The one CLI where both kinds are pure environment: an API key or
			// `claude setup-token`'s CLAUDE_CODE_OAUTH_TOKEN, so a launch needs
			// no file and no keychain.
			{Provider: "anthropic", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "ANTHROPIC_API_KEY", KindOAuth, "CLAUDE_CODE_OAUTH_TOKEN"), Native: nativeClaude},
		},
		// Claude Code signs in from its own TUI (`/login`), which is why the
		// command is empty and the hint names the command inside it. The CLI
		// also ships `claude auth login`; the TUI is the sign-in this declares.
		// A fresh install opens on its theme picker, not on a prompt — the person
		// who clicked Sign in and found no place to type /login was looking at
		// exactly that (2026-09-21). The hint names the step.
		Login: &LoginCommand{Hint: "In the terminal: finish its setup if it asks (a fresh install offers a theme), then type /login."},
	},
	{
		CLI: "codex", Name: "Codex",
		Providers: []Provider{
			// A ChatGPT login cannot be expressed by a variable — it needs the
			// CODEX_HOME directory, seeded so settings, skills and memories
			// survive the isolation while sessions and logs regenerate.
			{Provider: "openai-codex", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "OPENAI_API_KEY"), Native: nativeCodex},
		},
		// `codex login` opens a browser and writes $CODEX_HOME/auth.json.
		Login: &LoginCommand{Args: []string{"login"}, Hint: "Finish the sign-in in the browser it opens, then come back here."},
	},
	{
		CLI: "grok", Name: "Grok",
		Providers: []Provider{
			{Provider: "xai", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "XAI_API_KEY"), Native: nativeGrok, Note: noteGrokKey},
		},
		// `grok login` is Grok's own OAuth sign-in; it also takes --device-auth
		// for headless machines, but the plain command is the one that works
		// here and is what the person is told to finish in the browser.
		Login: &LoginCommand{Args: []string{"login"}, Hint: "Finish the sign-in in the browser it opens, then come back here."},
	},
	{
		CLI: "hermes", Name: "Hermes Agent",
		Providers: []Provider{
			// `hermes auth` owns the Codex login; its overlay declares no
			// environment variable for it.
			{Provider: "openai-codex", Kinds: []string{KindOAuth}, Native: nativeHermes, Note: noteHermesPool},
			// Hermes resolves ANTHROPIC_API_KEY, or an OAuth token in
			// ANTHROPIC_TOKEN / CLAUDE_CODE_OAUTH_TOKEN.
			{Provider: "anthropic", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "ANTHROPIC_API_KEY", KindOAuth, "CLAUDE_CODE_OAUTH_TOKEN"), Native: nativeHermes},
			// xai-oauth is the pool login; the key path is XAI_API_KEY.
			{Provider: "xai", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "XAI_API_KEY"), Native: nativeHermes, Note: noteHermesPool},
			{Provider: "google", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "GEMINI_API_KEY"), Native: nativeHermes},
			{Provider: "openrouter", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "OPENROUTER_API_KEY"), Native: nativeHermes},
			// copilot-acp is an external process login, not a token Hermes
			// writes; COPILOT_GITHUB_TOKEN (and GH_TOKEN) is the key path.
			{Provider: "github-copilot", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "COPILOT_GITHUB_TOKEN"), Native: nativeHermes, Note: noteHermesPool},
			{Provider: "opencode", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "OPENCODE_API_KEY"), Native: nativeHermes},
			// Z.AI / GLM also answers to GLM_API_KEY and Z_AI_API_KEY.
			{Provider: "zai", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "ZAI_API_KEY"), Native: nativeHermes},
			{Provider: "kimi-coding", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "KIMI_API_KEY"), Native: nativeHermes},
		},
		// `hermes auth add` asks for the provider on its own prompt, so no
		// provider id is passed: answering that prompt is part of the sign-in.
		Login: &LoginCommand{Args: []string{"auth", "add"}, Hint: "Answer the provider Hermes asks for, then finish the sign-in here."},
	},
	{
		CLI: "opencode", Name: "OpenCode",
		Providers: []Provider{
			// The env names are OpenCode's own models.dev list (its login
			// flows write its own store, which is why those rows are
			// import-only). Google accepts GEMINI_API_KEY first among
			// GOOGLE_API_KEY, GOOGLE_GENERATIVE_AI_API_KEY and GEMINI_API_KEY.
			{Provider: "anthropic", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "ANTHROPIC_API_KEY"), Native: nativeOpencode, Note: noteOwnStore},
			{Provider: "openai-codex", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "OPENAI_API_KEY"), Native: nativeOpencode, Note: noteOwnStore},
			{Provider: "xai", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "XAI_API_KEY"), Native: nativeOpencode, Note: noteOwnStore},
			{Provider: "google", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "GEMINI_API_KEY"), Native: nativeOpencode},
			{Provider: "openrouter", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "OPENROUTER_API_KEY"), Native: nativeOpencode},
			// GITHUB_TOKEN only: unverified — models.dev lists that variable for
			// Copilot, but whether the installed OpenCode also has a Copilot
			// login flow of its own was not established here.
			{Provider: "github-copilot", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "GITHUB_TOKEN"), Native: nativeOpencode},
			{Provider: "opencode", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "OPENCODE_API_KEY"), Native: nativeOpencode},
			// models.dev's id is zai-coding-plan; the variable is ZHIPU_API_KEY.
			{Provider: "zai", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "ZHIPU_API_KEY"), Native: nativeOpencode},
			// models.dev's ids are kimi-code-plan-global / -cn.
			{Provider: "kimi-coding", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "KIMI_API_KEY"), Native: nativeOpencode},
			// models.dev's Meta entry (api.meta.ai) takes META_MODEL_API_KEY;
			// this is the API-key path, not the Muse subscription.
			{Provider: "meta-ai", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "META_MODEL_API_KEY"), Native: nativeOpencode},
		},
		// `opencode auth login` asks which provider and which method first.
		Login: &LoginCommand{Args: []string{"auth", "login"}, Hint: "Pick the provider OpenCode asks for, sign in, then come back here."},
	},
	{
		CLI: "muse", Name: "Muse Code",
		Providers: []Provider{
			// `muse login` says META_API_KEY wins over the account login, so
			// the key row is injectable; the Meta session itself is not.
			{Provider: "meta-ai", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "META_API_KEY"), Native: nativeMuse, Note: noteMuseLogin},
		},
		// `muse login` is Meta's device-code flow: it prints a code, the person
		// approves it in a browser, and the launcher stores the account login
		// in providers.meta beside the api key.
		Login: &LoginCommand{Args: []string{"login"}, Hint: "Approve the code in your browser, then come back here."},
	},
	{
		CLI: "agy", Name: "Antigravity",
		Providers: []Provider{
			// Muse and Antigravity publish no credential variable PiCode could
			// confirm, so their rows are read-and-import only.
			{Provider: "google", Kinds: []string{KindOAuth}, Native: nativeAgy, Note: noteAgyLogin},
		},
		// No LoginCommand: `agy`'s subcommands on this machine are agent,
		// changelog, install, mcp, mic-serve, models, plugin, remote-control and
		// update — there is no sign-in command to open. Its Google login
		// happens inside the CLI itself (see the row's Note), so the pane
		// offers Import and not Sign in, and this comment is the reason.
	},
	{
		CLI: "omp", Name: "Omp",
		Providers: []Provider{
			// No Native anywhere on this Spec: Omp's credentials live in a
			// live SQLite database (agent.db) PiCode will not read, so the
			// env rows below are the only channel and the Notes say what an
			// import would cost.
			{Provider: "anthropic", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "ANTHROPIC_API_KEY", KindOAuth, "ANTHROPIC_OAUTH_TOKEN"), Note: noteOmpStore},
			{Provider: "openai-codex", Kinds: []string{KindOAuth}, Env: envs(KindOAuth, "OPENAI_CODEX_OAUTH_TOKEN"), Note: noteOmpStore},
			{Provider: "xai", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "XAI_API_KEY", KindOAuth, "XAI_OAUTH_TOKEN"), Note: noteOmpStore},
			{Provider: "google", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "GEMINI_API_KEY")},
			// Omp's docs' `meta` row: MODEL_API_KEY, then META_API_KEY.
			{Provider: "meta-ai", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "META_API_KEY")},
			{Provider: "openrouter", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "OPENROUTER_API_KEY")},
			{Provider: "github-copilot", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "COPILOT_GITHUB_TOKEN"), Note: noteOmpStore},
			// Omp's ids are opencode-zen / opencode-go, both OPENCODE_API_KEY.
			{Provider: "opencode", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "OPENCODE_API_KEY")},
			{Provider: "zai", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "ZAI_API_KEY")},
			// Omp reaches Kimi through `moonshot` (MOONSHOT_API_KEY, then
			// KIMI_API_KEY) and the OAuth provider `kimi-code`.
			{Provider: "kimi-coding", Kinds: []string{KindAPIKey, KindOAuth}, Env: envs(KindAPIKey, "KIMI_API_KEY"), Note: noteOmpStore},
			// Omp spells this variable LLAMA_CPP_API_KEY where pi spells it
			// LLAMA_API_KEY — the same key, one name per CLI.
			{Provider: "llama.cpp", Kinds: []string{KindAPIKey}, Env: envs(KindAPIKey, "LLAMA_CPP_API_KEY")},
		},
		// Omp publishes no login subcommand either — its `auth-broker` and
		// `auth-gateway` commands manage a credential vault and a proxy, they
		// do not sign in — so its sign-in is `/login` inside the TUI, and what
		// it stores goes to the SQLite database this package does not read: the
		// guided flow ends at Import.
		Login: &LoginCommand{Hint: "In the terminal: finish what it asks, type /login, then come back here."},
	},
}
