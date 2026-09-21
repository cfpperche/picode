package clicreds

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

// RenderLogin merges one vault credential into the CLI's own credential file
// and returns the bytes to write. existing is that file's current content
// (nil when it does not exist). It returns false when this CLI/kind cannot be
// activated this way — nothing is invented.
//
// This is ADR-0166's write, and it is parse.go's mirror: it takes the same
// format and provider, and every shape it writes is the shape the matching
// parser reads, so parseLogin(format, provider, RenderLogin(…)) answers with
// the credential that went in. What it does not write it keeps: the CLI's
// other providers, the keys inside the object it merges, and the file's
// unknown metadata come back unchanged at the JSON value level — a field the
// credential omits (an id_token, a subscription type) is left as the file had
// it, because PiCode cannot restore what it drops.
//
// A file that does not decode is refused rather than treated as empty —
// there may be a login in it PiCode cannot read — while an absent or blank
// file is the empty document, which is what a CLI leaves after a logout. The
// bytes come back as two-space indented JSON with a trailing newline, the
// shape PiCode writes its own files in; the CLI re-reads the file, so that is
// a stable diff, not a contract. Credential material appears in no error and
// no log: a refusal is a bool.
func RenderLogin(format, provider string, cred json.RawMessage, existing []byte) ([]byte, bool) {
	doc, ok := fileDoc(existing)
	if !ok {
		return nil, false
	}
	c, ok := readVaultCred(cred)
	if !ok {
		return nil, false
	}
	switch format {
	case "pi":
		ok = renderProviderMap(doc, provider, nil, KindAPIKey, c)
	case "opencode":
		// OpenCode spells an api key "api"; parseProviderMap accepts both, and
		// the file is OpenCode's own, so it gets OpenCode's spelling.
		ok = renderProviderMap(doc, provider, opencodeIDs, "api", c)
	case "claude":
		ok = renderClaude(doc, c)
	case "codex":
		ok = renderCodex(doc, c)
	case "grok":
		ok = renderGrok(doc, c)
	case "hermes":
		ok = renderHermes(doc, provider, c)
	case "muse":
		ok = renderMuse(doc, c)
	case "agy":
		ok = renderAgy(doc, c)
	default:
		// omp among them: a CLI with no credential file declaration has
		// nothing to render into.
		return nil, false
	}
	if !ok {
		return nil, false
	}
	return marshalDoc(doc)
}

// vaultCred is the credential RenderLogin accepts, decoded: pi's two vault
// shapes in one struct, since a stored row is one or the other.
type vaultCred struct {
	kind      string
	key       string
	access    string
	refresh   string
	expires   int64
	accountID string
}

// readVaultCred decodes and checks one vault credential, making the same
// refusals loginAPIKey and loginOAuth make: a key is the whole api-key fact, an
// oauth row needs both tokens because an access token alone cannot be renewed,
// and a type that is neither is not a credential. A half-credential is never
// written into a vendor's file.
func readVaultCred(raw json.RawMessage) (vaultCred, bool) {
	var wire struct {
		Type      string `json:"type"`
		Key       string `json:"key"`
		Access    string `json:"access"`
		Refresh   string `json:"refresh"`
		Expires   int64  `json:"expires"`
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return vaultCred{}, false
	}
	switch strings.ToLower(strings.TrimSpace(wire.Type)) {
	case KindAPIKey:
		c := vaultCred{kind: KindAPIKey, key: strings.TrimSpace(wire.Key)}
		return c, c.key != ""
	case KindOAuth:
		c := vaultCred{
			kind:      KindOAuth,
			access:    strings.TrimSpace(wire.Access),
			refresh:   strings.TrimSpace(wire.Refresh),
			accountID: strings.TrimSpace(wire.AccountID),
		}
		if wire.Expires > 0 {
			c.expires = wire.Expires
		}
		return c, c.access != "" && c.refresh != ""
	}
	return vaultCred{}, false
}

// renderProviderMap writes the one shape pi and OpenCode share, under the key
// the parser would read:
//
//	{"<provider>": {"type":apiKind,"key":…}} or
//	{"<provider>": {"type":"oauth","access":…,"refresh":…,"expires"?,"accountId"?}}
//
// The entry is replaced whole — it is the one object PiCode owns — while every
// other provider stays. expires is the vault's milliseconds, which is what the
// parser reads back.
func renderProviderMap(doc object, provider string, aliases map[string][]string, apiKind string, c vaultCred) bool {
	var entry object
	switch c.kind {
	case KindAPIKey:
		entry = object{"type": apiKind, "key": c.key}
	case KindOAuth:
		entry = object{"type": KindOAuth, "access": c.access, "refresh": c.refresh}
		if c.expires > 0 {
			entry["expires"] = c.expires
		}
		if c.accountID != "" {
			entry["accountId"] = c.accountID
		}
	default:
		return false
	}
	doc[entryKey(doc, provider, aliases)] = entry
	return true
}

// renderClaude writes Claude Code's own login — the claudeAiOauth object:
//
//	{"claudeAiOauth":{"accessToken":…,"refreshToken":…,"expiresAt":(ms)?}}
//
// An api key is refused: Claude Code reads that from ANTHROPIC_API_KEY, not
// from this file. The object is merged, so the file's subscription type, rate
// limit tier and scopes, and its vendor plugin OAuth (mcpOAuth) survive.
func renderClaude(doc object, c vaultCred) bool {
	if c.kind != KindOAuth {
		return false
	}
	oauth, ok := child(doc, "claudeAiOauth")
	if !ok {
		oauth = object{}
	}
	oauth["accessToken"] = c.access
	oauth["refreshToken"] = c.refresh
	if c.expires > 0 {
		oauth["expiresAt"] = c.expires
	}
	doc["claudeAiOauth"] = oauth
	return true
}

// Codex's own auth_mode values: a key is "apikey", a ChatGPT login "chatgpt".
// Both were read from the installed Codex's auth.json (parse.go's fixtures).
const (
	codexAPIKeyMode  = "apikey"
	codexChatGPTMode = "chatgpt"
)

// renderCodex writes Codex's `$CODEX_HOME/auth.json`:
//
//	{"OPENAI_API_KEY":…,"auth_mode":"apikey"}                     — api key
//	{"tokens":{…},"last_refresh":…,"auth_mode":"chatgpt"}         — ChatGPT login
//
// auth_mode follows the credential written, because that is the mode Codex
// reads: leaving it saying "apikey" next to a token login would make the write
// do nothing. A key left in the file is nulled for the same reason — Codex
// prefers a key over its tokens — and the field stays, spelling "no key" the
// way Codex itself does. Everything else, the id_token and the rest of the
// tokens, is merged.
//
// last_refresh is the one value read from the clock: Codex schedules its token
// refresh from it, and an activation is a fresh login.
func renderCodex(doc object, c vaultCred) bool {
	switch c.kind {
	case KindAPIKey:
		doc["OPENAI_API_KEY"] = c.key
		doc["auth_mode"] = codexAPIKeyMode
	case KindOAuth:
		tokens, ok := child(doc, "tokens")
		if !ok {
			tokens = object{}
		}
		tokens["access_token"] = c.access
		tokens["refresh_token"] = c.refresh
		if c.accountID != "" {
			tokens["account_id"] = c.accountID
		}
		doc["tokens"] = tokens
		doc["auth_mode"] = codexChatGPTMode
		doc["last_refresh"] = time.Now().UTC().Format(time.RFC3339)
		if text(doc, "OPENAI_API_KEY") != "" {
			doc["OPENAI_API_KEY"] = nil
		}
	default:
		return false
	}
	return true
}

// renderGrok writes Grok's `$GROK_HOME/auth.json`: the entry keyed
// "<oidc_issuer>::<oidc_client_id>" gets `key` (the access token),
// `refresh_token` and the RFC 3339 `expires_at`.
//
// The map key is the vendor's session key, so the entry is updated in place and
// its oidc_issuer, oidc_client_id, user_id and email are kept. PiCode will not
// invent that key: a file with no session yet is refused. An api key has no
// entry here at all — Grok's session needs the refresh token that renews it.
func renderGrok(doc object, c vaultCred) bool {
	if c.kind != KindOAuth {
		return false
	}
	for _, id := range sortedKeys(doc) {
		if !strings.Contains(id, "::") {
			continue
		}
		entry, ok := child(doc, id)
		if !ok {
			continue
		}
		entry["key"] = c.access
		entry["refresh_token"] = c.refresh
		if c.expires > 0 {
			entry["expires_at"] = isoFromMillis(c.expires)
		}
		doc[id] = entry
		return true
	}
	return false
}

// renderHermes writes Hermes' per-provider document:
//
//	{"providers":{"<id>":{"tokens":{"access_token":…,"refresh_token":…,"account_id"?}}}}
//
// The provider object is merged, not replaced: Hermes keeps its auth_mode and
// its own last_refresh there. An api key is refused — hermes reads those from
// .env, not from this file.
func renderHermes(doc object, provider string, c vaultCred) bool {
	if c.kind != KindOAuth {
		return false
	}
	providers, ok := child(doc, "providers")
	if !ok {
		providers = object{}
	}
	key := entryKey(providers, provider, hermesIDs)
	entry, ok := child(providers, key)
	if !ok {
		entry = object{}
	}
	tokens, ok := child(entry, "tokens")
	if !ok {
		tokens = object{}
	}
	tokens["access_token"] = c.access
	tokens["refresh_token"] = c.refresh
	if c.accountID != "" {
		tokens["account_id"] = c.accountID
	}
	entry["tokens"] = tokens
	providers[key] = entry
	doc["providers"] = providers
	return true
}

// renderMuse writes Muse Code's `providers.meta` — one object in the two shapes
// the CLI's own file holds, and the two parseMuse reads:
//
//	{"mechanism":"oauth","access_token":…,"refresh_token":…,"expires_at":(RFC 3339)?}
//	{"api_key":…}
//
// Each write clears the other shape's fields, and that is the point of the
// function: Muse prefers the key (its own launcher reads the account login only
// under `mechanism: "oauth"`, and META_API_KEY wins over both), so an api key
// left beside an account login would silently defeat the activation, and a
// mechanism/access_token/refresh_token/expires_at left beside a key would be a
// session the file no longer has. Everything else the object holds — its
// api_base_url, its user_email — is kept.
func renderMuse(doc object, c vaultCred) bool {
	providers, ok := child(doc, "providers")
	if !ok {
		providers = object{}
	}
	meta, ok := child(providers, "meta")
	if !ok {
		meta = object{}
	}
	switch c.kind {
	case KindAPIKey:
		meta["api_key"] = c.key
		for _, field := range []string{"mechanism", "access_token", "refresh_token", "expires_at"} {
			delete(meta, field)
		}
	case KindOAuth:
		meta["mechanism"] = KindOAuth
		meta["access_token"] = c.access
		// Muse's own reader (the launcher inside the binary) reads mechanism,
		// access_token and expires_at and nothing else — no refresh token
		// anywhere — and it takes expires_at only as a number of unix seconds
		// (`json_type == number`, digits, or it is ignored). Writing a string
		// there, or a refresh token Muse never asked for, would be PiCode
		// guessing at a shape the vendor already published.
		if c.refresh != "" {
			meta["refresh_token"] = c.refresh
		} else {
			delete(meta, "refresh_token")
		}
		if c.expires > 0 {
			meta["expires_at"] = c.expires / 1000
		} else {
			delete(meta, "expires_at")
		}
		delete(meta, "api_key")
	default:
		return false
	}
	providers["meta"] = meta
	doc["providers"] = providers
	return true
}

// renderAgy writes Antigravity's antigravity-oauth-token: the token object's
// two tokens and its RFC 3339 expiry. The file's auth_method, id_token and
// token_type are its own and are kept. An api key is refused — that file is a
// Google session and nothing else.
func renderAgy(doc object, c vaultCred) bool {
	if c.kind != KindOAuth {
		return false
	}
	token, ok := child(doc, "token")
	if !ok {
		token = object{}
	}
	token["access_token"] = c.access
	token["refresh_token"] = c.refresh
	if c.expires > 0 {
		token["expiry"] = isoFromMillis(c.expires)
	}
	doc["token"] = token
	return true
}

// entryKey is the key a write lands on: the first name the matching parser
// would look up that the file already carries, else the declared provider id.
// It mirrors candidateIDs so a file keyed with the CLI's own name for a
// provider — OpenCode's "zai-coding-plan", Hermes' "xai-oauth" — is updated in
// place instead of gaining a second entry for the same account.
func entryKey(doc object, provider string, aliases map[string][]string) string {
	for _, id := range candidateIDs(provider, aliases) {
		if _, ok := child(doc, id); ok {
			return id
		}
	}
	return provider
}

// fileDoc is the document a render starts from: the file's own content, or the
// empty document when it is absent or blank. Numbers stay literals (UseNumber)
// rather than float64, so a value PiCode does not own — a large id, a
// nanosecond timestamp — is written back exactly as it was read; parse.go's
// readers coerce instead, because they only read what they understand.
//
// A file that does not decode — malformed bytes, an array, a scalar — is
// refused, not treated as empty: PiCode does not overwrite bytes it cannot
// read.
func fileDoc(existing []byte) (object, bool) {
	if len(bytes.TrimSpace(existing)) == 0 {
		return object{}, true
	}
	dec := json.NewDecoder(bytes.NewReader(existing))
	dec.UseNumber()
	var doc object
	if err := dec.Decode(&doc); err != nil || doc == nil {
		return nil, false
	}
	// One document only: bytes after it are bytes PiCode cannot account for.
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, false
	}
	return doc, true
}

// marshalDoc is the bytes a write lands: two-space indented JSON with a
// trailing newline, the shape PiCode writes its own files in. Map keys are
// sorted, so the same inputs render the same file.
func marshalDoc(doc object) ([]byte, bool) {
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, false
	}
	return append(out, '\n'), true
}

// isoFromMillis is isoMillis' inverse: the RFC 3339 timestamp an ISO-expiry
// file gets. The vault counts milliseconds and RFC 3339 carries them, so the
// value survives the round trip ("…:05.123Z") while a whole second stays clean.
func isoFromMillis(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339Nano)
}
