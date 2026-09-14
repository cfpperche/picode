// Package modellist asks a provider endpoint what models it serves.
//
// This is the one outbound call PiCode makes on the provider's behalf, and it
// is a metadata listing only: no prompt, no completion and no model traffic
// passes through PiCode (ADR-0003, ADR-0129). The key is used for this request
// and never returned to the caller (the handler tests pin that).
package modellist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The API types a custom endpoint may declare (mirrors catalog's constants;
// kept here as strings so this package stays free of the catalog).
const (
	APIOpenAICompletions = "openai-completions"
	APIOpenAIResponses   = "openai-responses"
	APIAnthropic         = "anthropic-messages"
	APIGoogle            = "google-generative-ai"
)

// Model is one listed id, with the limits the endpoint volunteered.
type Model struct {
	ID            string `json:"id"`
	ContextWindow int    `json:"contextWindow,omitempty"`
	MaxTokens     int    `json:"maxTokens,omitempty"`
}

// Result is a successful listing. URL names the address that answered, so a
// form can tell the user which one it used (a gateway that needs /v1 is a
// common surprise).
type Result struct {
	Models []Model `json:"models"`
	URL    string  `json:"url"`
}

// Kind classifies a failure into the fix a person can act on.
type Kind string

const (
	KindAuth      Kind = "auth"      // the key was refused
	KindMissing   Kind = "missing"   // no model list at that URL
	KindBilling   Kind = "billing"   // the account cannot list
	KindUpstream  Kind = "upstream"  // the endpoint answered with an error
	KindTransport Kind = "transport" // never reached the endpoint
	KindInput     Kind = "input"     // the request itself cannot be built
)

// Error is the classified failure. The handler turns it into the one line the
// dialog shows; nothing here includes the key.
type Error struct {
	Kind   Kind
	Status int // upstream status when KindUpstream/KindAuth/KindMissing; 0 otherwise
	Host   string
	Detail string
	// Hint is optional guidance for the caller to append, set only where it
	// answers the failure the person is looking at (an unknown API type). It is
	// a field and not a suffix the handler guesses at by matching Detail.
	Hint string
}

// SupportedAPIs is what this package can address, in the words a person reads.
const SupportedAPIs = "OpenAI Chat Completions, OpenAI Responses, Anthropic Messages and Google Generative AI"

func (e *Error) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("%s: %s (%d)", e.Kind, e.Detail, e.Status)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Detail)
}

// maxBody caps what an endpoint may send back. A model list is small; the cap
// only exists so a wrong URL answering with a huge page cannot fill memory.
const maxBody = 2 << 20

var httpClient = &http.Client{Timeout: 12 * time.Second}

// List asks baseURL for its models. key may be empty (some local endpoints
// need none). api picks the shape and the auth header.
func List(ctx context.Context, baseURL, api, key string) (Result, error) {
	base, err := normalizeBase(baseURL)
	if err != nil {
		return Result{}, err
	}
	if api == "" {
		api = APIOpenAICompletions
	}
	switch api {
	case APIOpenAICompletions, APIOpenAIResponses, APIAnthropic, APIGoogle:
	default:
		return Result{}, &Error{Kind: KindInput, Detail: "unknown API type " + api, Hint: "It covers " + SupportedAPIs}
	}

	// The first path is the one pi itself uses; a gateway whose base URL has
	// no /v1 is offered the other before giving up.
	paths := []string{"/models"}
	if !strings.HasSuffix(base.Path, "/v1") && !strings.HasSuffix(base.Path, "/v1beta") {
		paths = append(paths, "/v1/models")
	}
	var last *Error
	for _, p := range paths {
		u := base
		u.Path = strings.TrimSuffix(u.Path, "/") + p
		res, err := fetch(ctx, u, api, key)
		if err == nil {
			res.URL = u.String()
			return res, nil
		}
		var typed *Error
		if !errors.As(err, &typed) {
			return Result{}, err
		}
		last = typed
		// Only a missing list justifies trying the next address: a refused
		// key or an unreachable host answers the same way on both.
		if typed.Kind != KindMissing {
			return Result{}, typed
		}
	}
	return Result{}, last
}

// fetch performs one request and parses the answer.
func fetch(ctx context.Context, u url.URL, api, key string) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, &Error{Kind: KindInput, Host: u.Host, Detail: err.Error()}
	}
	req.Header.Set("User-Agent", "picode (https://github.com/cfpperche/picode)")
	req.Header.Set("Accept", "application/json")
	switch {
	case key == "":
	case api == APIAnthropic:
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	case api == APIGoogle:
		req.Header.Set("x-goog-api-key", key)
	default:
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, &Error{Kind: KindTransport, Host: u.Host, Detail: reachError(err)}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Result{}, &Error{Kind: KindTransport, Host: u.Host, Detail: "the answer was cut short"}
	}
	if resp.StatusCode != http.StatusOK {
		return Result{}, classify(resp.StatusCode, u.Host, body, key)
	}
	models, err := parse(api, body)
	if err != nil {
		return Result{}, &Error{Kind: KindInput, Status: resp.StatusCode, Host: u.Host, Detail: err.Error()}
	}
	return Result{Models: models}, nil
}

// classify turns an upstream status into the kind the dialog explains.
func classify(status int, host string, body []byte, key string) *Error {
	detail := upstreamDetail(body, key)
	e := &Error{Status: status, Host: host, Detail: detail}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		e.Kind = KindAuth
	case status == http.StatusPaymentRequired:
		e.Kind = KindBilling
	case status == http.StatusNotFound || status == http.StatusMethodNotAllowed:
		e.Kind = KindMissing
		e.Detail = "no model list"
	default:
		e.Kind = KindUpstream
	}
	return e
}

// upstreamDetail pulls the provider's own message out of an error body, so a
// person sees "invalid api key" instead of an empty status. A provider that
// echoes the key back has it redacted: the detail travels to the browser, and
// key material never does (ADR-0129).
func upstreamDetail(body []byte, key string) string {
	var obj struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}
	if json.Unmarshal(body, &obj) == nil {
		for _, s := range []string{obj.Error.Message, obj.Message, obj.Detail, obj.Error.Type} {
			if t := strings.TrimSpace(s); t != "" {
				return truncate(redact(t, key), 160)
			}
		}
	}
	if t := strings.TrimSpace(string(body)); t != "" && !strings.HasPrefix(t, "<") {
		return truncate(redact(t, key), 160)
	}
	return "no details"
}

// redact removes the key from a message that came back from the endpoint.
//
// Only a key long enough to be a credential is redacted: a real key is a long
// random string, while a one- or two-character value would match letters of
// ordinary words and turn "max_tokens" into "max_to***ens" — over-redaction is
// safe but it destroys the message a person needs to read. Nothing shorter than
// credentialMinLen can be a working provider key.
func redact(s, key string) string {
	if len(key) < credentialMinLen {
		return s
	}
	return strings.ReplaceAll(s, key, "***")
}

// credentialMinLen is the shortest string treated as key material.
const credentialMinLen = 8

// parse reads the two list shapes the supported API types answer with: the
// OpenAI/Anthropic {"data":[{...}]} envelope and Google's {"models":[{...}]}
// with "models/" prefixed names.
func parse(api string, body []byte) ([]Model, error) {
	var envelope struct {
		Data   *[]json.RawMessage `json:"data"`
		Models *[]json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("the answer is not a model list")
	}
	// Presence, not length: an endpoint that answers with an empty list has
	// told us something real ("no models here"), while a body with neither key
	// is not a listing at all.
	var raw []json.RawMessage
	switch {
	case api == APIGoogle && envelope.Models != nil:
		raw = *envelope.Models
	case envelope.Data != nil:
		raw = *envelope.Data
	case envelope.Models != nil:
		raw = *envelope.Models
	default:
		return nil, fmt.Errorf("the answer carries no model list")
	}
	var out []Model
	seen := map[string]bool{}
	for _, item := range raw {
		m, ok := parseItem(item)
		if !ok || seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		out = append(out, m)
	}
	// The dialog shows them in the order the endpoint gave: a gateway's own
	// order is the one a user hunting for a name recognises.
	return out, nil
}

// parseItem accepts both an object and a bare string, because gateways send
// both.
func parseItem(item json.RawMessage) (Model, bool) {
	var s string
	if json.Unmarshal(item, &s) == nil {
		id := strings.TrimSpace(s)
		return Model{ID: id}, id != ""
	}
	var obj struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		// Context window spellings seen in the wild.
		ContextLength    int `json:"context_length"`
		ContextWindow    int `json:"context_window"`
		MaxContextLength int `json:"max_context_length"`
		InputTokenLimit  int `json:"inputTokenLimit"`
		// Max output spellings.
		MaxOutputTokens     int `json:"max_output_tokens"`
		MaxCompletionTokens int `json:"max_completion_tokens"`
		MaxTokens           int `json:"max_tokens"`
		OutputTokenLimit    int `json:"outputTokenLimit"`
		TopProvider         struct {
			MaxCompletionTokens int `json:"max_completion_tokens"`
		} `json:"top_provider"`
	}
	if json.Unmarshal(item, &obj) != nil {
		return Model{}, false
	}
	id := strings.TrimSpace(obj.ID)
	if id == "" {
		// Google names entries "models/gemini-x"; pi wants the bare id.
		id = strings.TrimPrefix(strings.TrimSpace(obj.Name), "models/")
	}
	if id == "" {
		return Model{}, false
	}
	m := Model{ID: id}
	for _, v := range []int{obj.ContextLength, obj.ContextWindow, obj.MaxContextLength, obj.InputTokenLimit} {
		if v > 0 {
			m.ContextWindow = v
			break
		}
	}
	for _, v := range []int{obj.MaxOutputTokens, obj.MaxCompletionTokens, obj.MaxTokens, obj.OutputTokenLimit, obj.TopProvider.MaxCompletionTokens} {
		if v > 0 {
			m.MaxTokens = v
			break
		}
	}
	return m, true
}

// normalizeBase checks the address before anything is dialled.
func normalizeBase(baseURL string) (url.URL, error) {
	raw := strings.TrimSpace(baseURL)
	if raw == "" {
		return url.URL{}, &Error{Kind: KindInput, Detail: "a base URL is required"}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return url.URL{}, &Error{Kind: KindInput, Detail: "that base URL is not a URL"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return url.URL{}, &Error{Kind: KindInput, Host: u.Host, Detail: "the base URL must start with http:// or https://"}
	}
	if u.Host == "" {
		return url.URL{}, &Error{Kind: KindInput, Detail: "that base URL has no host"}
	}
	return *u, nil
}

// reachError turns a transport failure into something a person can act on.
func reachError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "context deadline exceeded"), strings.Contains(msg, "timeout"):
		return "the endpoint did not answer in 12s"
	case strings.Contains(msg, "no such host"):
		return "that host name does not resolve"
	case strings.Contains(msg, "connection refused"):
		return "nothing is listening on that address"
	case strings.Contains(msg, "certificate"):
		return "the TLS certificate was refused"
	}
	return truncate(msg, 160)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Never split a rune.
	for n > 0 && !utf8Start(s[n]) {
		n--
	}
	return strings.TrimSpace(s[:n]) + "…"
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }
