package modellist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Probe proves a key with one minimal real request.
//
// This is the one place PiCode spends a provider's money (the owner's call;
// open topic P4): everything else PiCode sends is a listing. The prompt is one
// word and the output ceiling is the smallest the API accepts, so the cost is a
// fraction of a cent and the answer is real — a key that pi would fail on is
// caught before the first turn, which a presence check cannot do.
//
// The response body is discarded: nothing but timing, the model that answered
// and (when the endpoint reports them) the token counts leaves this function.
type ProbeResult struct {
	Model        string `json:"model"`
	MS           int    `json:"ms"`
	InputTokens  int    `json:"inputTokens,omitempty"`
	OutputTokens int    `json:"outputTokens,omitempty"`
}

// probePrompt is the whole payload; one word in, at most a token or two out.
const probePrompt = "hi"

// probeTimeout is this call's own budget: a probe holds a dialog open, so it
// answers faster than a listing would.
const probeTimeout = 20 * time.Second

// KindModel and KindQuota sharpen the kinds for a probe: a 404 here means the
// model (or the route) is wrong, and a 429 is the account's quota, not a
// generic upstream error.
const (
	KindModel Kind = "model"
	KindQuota Kind = "quota"
)

// Probe sends one minimal completion to model and reports what came back.
func Probe(ctx context.Context, baseURL, api, key, model string) (ProbeResult, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return ProbeResult{}, &Error{Kind: KindInput, Detail: "no model to verify — the endpoint has none listed"}
	}
	base, err := normalizeBase(baseURL)
	if err != nil {
		return ProbeResult{}, err
	}
	if api == "" {
		api = APIOpenAICompletions
	}
	switch api {
	case APIOpenAICompletions, APIOpenAIResponses, APIAnthropic, APIGoogle:
	default:
		return ProbeResult{}, &Error{Kind: KindInput, Detail: "unknown API type " + api, Hint: "Verify covers " + SupportedAPIs}
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	start := time.Now()
	res, err := probeOnce(ctx, base, api, key, model, false)
	if err != nil {
		// A gateway that only speaks the newer field says so in its own words;
		// one retry with max_completion_tokens is cheaper than making the user
		// read that message and guess.
		var typed *Error
		if api == APIOpenAICompletions && errors.As(err, &typed) && typed.Kind == KindInput &&
			strings.Contains(strings.ToLower(typed.Detail), "max_tokens") {
			res, err = probeOnce(ctx, base, api, key, model, true)
		}
		if err != nil {
			return ProbeResult{}, err
		}
	}
	res.MS = int(time.Since(start).Milliseconds())
	res.Model = model
	return res, nil
}

// probeOnce performs one attempt. completionField switches the OpenAI-style
// output ceiling between max_tokens and max_completion_tokens.
func probeOnce(ctx context.Context, base url.URL, api, key, model string, completionField bool) (ProbeResult, error) {
	path, payload := probePayload(api, model, completionField)
	u := base
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	if api == APIGoogle {
		// Google puts the model in the route rather than the body.
		u.Path = strings.TrimSuffix(u.Path, "/models") + "/models/" + url.PathEscape(model) + ":generateContent"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ProbeResult{}, &Error{Kind: KindInput, Host: u.Host, Detail: err.Error()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return ProbeResult{}, &Error{Kind: KindInput, Host: u.Host, Detail: err.Error()}
	}
	req.Header.Set("User-Agent", "picode (https://github.com/cfpperche/picode)")
	req.Header.Set("Content-Type", "application/json")
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
		return ProbeResult{}, &Error{Kind: KindTransport, Host: u.Host, Detail: reachError(err)}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return ProbeResult{}, &Error{Kind: KindTransport, Host: u.Host, Detail: "the answer was cut short"}
	}
	if resp.StatusCode != http.StatusOK {
		return ProbeResult{}, probeFailure(resp.StatusCode, u.Host, raw, key, model)
	}
	return usageOf(raw), nil
}

// probePayload builds the smallest request each API type accepts.
func probePayload(api, model string, completionField bool) (string, map[string]any) {
	switch api {
	case APIAnthropic:
		return "/messages", map[string]any{
			"model":      model,
			"max_tokens": 1,
			"messages":   []map[string]any{{"role": "user", "content": probePrompt}},
		}
	case APIGoogle:
		return "/models", map[string]any{
			"contents":         []map[string]any{{"role": "user", "parts": []map[string]any{{"text": probePrompt}}}},
			"generationConfig": map[string]any{"maxOutputTokens": 1},
		}
	case APIOpenAIResponses:
		// The Responses API wants its own field names and a floor of 16 output
		// tokens, so this probe costs a few tokens more than the others.
		return "/responses", map[string]any{
			"model":             model,
			"input":             probePrompt,
			"max_output_tokens": 16,
			"stream":            false,
		}
	}
	limitKey := "max_tokens"
	if completionField {
		limitKey = "max_completion_tokens"
	}
	return "/chat/completions", map[string]any{
		"model":    model,
		"messages": []map[string]any{{"role": "user", "content": probePrompt}},
		limitKey:   1,
		"stream":   false,
	}
}

// probeFailure classifies a failed probe, naming the model where that is the
// answer and never repeating the key.
func probeFailure(status int, host string, body []byte, key, model string) *Error {
	detail := upstreamDetail(body, key)
	e := &Error{Status: status, Host: host, Detail: detail}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		e.Kind = KindAuth
	case status == http.StatusPaymentRequired || status == http.StatusTooManyRequests:
		e.Kind = KindQuota
	case status == http.StatusNotFound:
		e.Kind = KindModel
		e.Detail = model
		if detail != "no details" {
			e.Detail = model + " — " + detail
		}
	case status == http.StatusBadRequest:
		e.Kind = KindInput
	default:
		e.Kind = KindUpstream
	}
	return e
}

// usageOf reads the token counts an endpoint reports, in the three shapes the
// supported APIs use. Reporting them is how a person sees what the probe cost.
func usageOf(body []byte) ProbeResult {
	var obj struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			InputTokens      int `json:"input_tokens"`
			OutputTokens     int `json:"output_tokens"`
		} `json:"usage"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if json.Unmarshal(body, &obj) != nil {
		return ProbeResult{}
	}
	in := obj.Usage.PromptTokens + obj.Usage.InputTokens + obj.UsageMetadata.PromptTokenCount
	out := obj.Usage.CompletionTokens + obj.Usage.OutputTokens + obj.UsageMetadata.CandidatesTokenCount
	return ProbeResult{InputTokens: in, OutputTokens: out}
}
