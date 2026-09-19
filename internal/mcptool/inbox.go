package mcptool

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

// The inbox family (ADR-0037 over MCP, ADR-0154 N1): notify_human files an
// FYI, ask_human files a blocking question. Text and payloads mirror
// packages/pi-inbox. One difference by necessity: pi ends its turn and the
// human's reply rides pi's receiver back; a guest CLI has no receiver, so
// ask_human waits on the item — the daemon records the answer on a
// question from a terminal that is not running pi — and hands the answer
// back as the tool result, or says how to keep waiting.

const (
	inboxMaxTitle = 200
	inboxMaxBody  = 100_000
	// askWaitDefault is how long one ask_human call waits for the human
	// (PICODE_ASK_WAIT seconds overrides). A client's tool timeout is
	// wall-clock (Claude Code: ~28 h by default) plus an idle window that
	// progress notifications keep open, so the wait is long: the human may
	// be away from the desk. At the deadline, or when the client cancels,
	// the call names the item so the model can resume the wait.
	askWaitDefault = 8 * time.Hour
	askPoll        = 2 * time.Second
	// askProgressEvery is how often the wait reports progress to the client.
	askProgressEvery = 15 * time.Second
)

const inboxUnreachable = "PiCode is not reachable (no server.json or connection refused) — could not file to the inbox. " +
	"Proceed without human input or surface this in your final message."

var notifyGuidelines = []string{
	"Use notify_human only for things the human must know; silence is a valid outcome.",
	"Send one consolidated notify_human per state change, never one per event.",
	"Your final answer belongs in your reply to the user, not in notify_human.",
}

var askGuidelines = []string{
	"Use ask_human when you are genuinely blocked on a decision only the human can make.",
	"ask_human waits for the answer — hours if needed — and returns it; if it ever comes back unanswered, keep waiting with the item id it names instead of asking again.",
	"Ask one consolidated question; do not file several ask_human items in a row.",
}

var inboxFamily = Family{
	Name:         "inbox",
	Instructions: "notify_human:\n- " + strings.Join(notifyGuidelines, "\n- ") + "\n\nask_human:\n- " + strings.Join(askGuidelines, "\n- "),
	Tools: func(c *Caller) []Tool {
		return []Tool{
			{
				Name:        "notify_human",
				Description: "File a non-blocking note into the human's PiCode inbox. Use it only for things the human must know about; it never interrupts them.",
				InputSchema: map[string]any{
					"type": "object", "required": []string{"title"},
					"properties": map[string]any{
						"title":  map[string]any{"type": "string", "description": "Short headline (what happened)"},
						"body":   map[string]any{"type": "string", "description": "Details, markdown"},
						"reason": map[string]any{"type": "string", "description": "Why the human is seeing this"},
					},
				},
				Call: func(ctx context.Context, args json.RawMessage) Result { return notifyCall(ctx, c, args) },
			},
			{
				Name:        "ask_human",
				Description: "File a blocking question into the human's PiCode inbox and wait for the answer. Returns the human's reply, or says the question is still open and how to keep waiting.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{"type": "string", "description": "The question, one sentence if possible"},
						"context":  map[string]any{"type": "string", "description": "What the human needs to answer well, markdown"},
						"item":     map[string]any{"type": "string", "description": "Keep waiting on a question already filed: the item id a previous call named"},
					},
				},
				Call: func(ctx context.Context, args json.RawMessage) Result {
					return askCall(ctx, c, args, askWait(), time.Sleep)
				},
			},
		}
	},
}

func askWait() time.Duration {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("PICODE_ASK_WAIT"))); err == nil && v > 0 {
		return time.Duration(v) * time.Second
	}
	return askWaitDefault
}

func clip(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}

// InboxIdentity is pi-inbox's agentIdentity: agent, else terminal, else
// honest provenance for an unmanaged caller.
func InboxIdentity(id Identity) (kind, source string) {
	if id.Agent != "" {
		return "agent", id.Agent
	}
	if id.Term != "" {
		return "terminal", id.Term
	}
	return "system", "mcp (unmanaged)"
}

// NotifyPayload is pi-inbox's buildNotifyPayload.
func NotifyPayload(id Identity, title, body, reason string) (map[string]any, error) {
	title = clip(strings.TrimSpace(title), inboxMaxTitle)
	if title == "" {
		return nil, errString("notify_human needs a title")
	}
	if reason = strings.TrimSpace(reason); reason == "" {
		reason = "agent notification"
	}
	kind, source := InboxIdentity(id)
	return map[string]any{
		"kind": "fyi", "sourceKind": kind, "sourceId": source,
		"reason": clip(reason, inboxMaxTitle), "title": title, "body": clip(strings.TrimSpace(body), inboxMaxBody), "blocking": false,
	}, nil
}

// AskPayload is pi-inbox's buildAskPayload (no session path: a guest CLI
// has no pi session to resume).
func AskPayload(id Identity, question, context string) (map[string]any, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errString("ask_human needs a question")
	}
	body := question
	if context = strings.TrimSpace(context); context != "" {
		body = question + "\n\n" + context
	}
	kind, source := InboxIdentity(id)
	return map[string]any{
		"kind": "question", "sourceKind": kind, "sourceId": source,
		"reason": "agent needs your input", "title": clip(question, inboxMaxTitle), "body": clip(body, inboxMaxBody), "blocking": true,
	}, nil
}

// AnswerLine reads the response the store recorded ("<verb>: <text>", or
// the bare verb) the way the model should hear it.
func AnswerLine(response *string) string {
	if response == nil {
		return "The human closed the question without an answer."
	}
	raw := strings.TrimSpace(*response)
	for _, verb := range []string{"respond", "edit"} {
		if text, ok := strings.CutPrefix(raw, verb+": "); ok && strings.TrimSpace(text) != "" {
			return "The human answered: " + strings.TrimSpace(text)
		}
	}
	switch raw {
	case "", "ignore":
		return "The human closed the question without an answer."
	case "accept":
		return "The human accepted."
	}
	return "The human answered: " + raw
}

type errString string

func (e errString) Error() string { return string(e) }

func inboxPost(ctx context.Context, c *Caller, payload map[string]any) (item map[string]any, soft string) {
	if c.Unreachable != "" || c.Daemon == nil {
		return nil, inboxUnreachable
	}
	raw, _ := json.Marshal(payload)
	status, body, err := c.Daemon.Post(ctx, "/api/inbox", raw)
	if err != nil {
		return nil, inboxUnreachable
	}
	if status < 200 || status >= 300 {
		msg := "HTTP " + itoa(status)
		var parsed struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &parsed) == nil && parsed.Error != "" {
			msg = parsed.Error
		}
		return nil, "Inbox refused the item: " + msg
	}
	_ = json.Unmarshal(body, &item)
	return item, ""
}

func notifyCall(ctx context.Context, c *Caller, args json.RawMessage) Result {
	var p struct{ Title, Body, Reason string }
	if err := json.Unmarshal(args, &p); err != nil {
		return Fail("notify_human: arguments must be an object")
	}
	payload, err := NotifyPayload(c.Identity, p.Title, p.Body, p.Reason)
	if err != nil {
		return Fail(err.Error())
	}
	if _, soft := inboxPost(ctx, c, payload); soft != "" {
		return Text(soft)
	}
	return Text("Filed to the human's inbox.")
}

// askCall files (or resumes) a question and waits up to wait for the
// answer, polling the item. sleep is injected for tests.
func askCall(ctx context.Context, c *Caller, args json.RawMessage, wait time.Duration, sleep func(time.Duration)) Result {
	var p struct{ Question, Context, Item string }
	if err := json.Unmarshal(args, &p); err != nil {
		return Fail("ask_human: arguments must be an object")
	}
	id := strings.TrimSpace(p.Item)
	if id == "" {
		payload, err := AskPayload(c.Identity, p.Question, p.Context)
		if err != nil {
			return Fail(err.Error())
		}
		item, soft := inboxPost(ctx, c, payload)
		if soft != "" {
			return Text(soft)
		}
		id, _ = item["id"].(string)
		if id == "" {
			return Text("Question filed to the human's inbox, but PiCode did not name the item — answer will arrive in the terminal, not here.")
		}
	}
	deadline := time.Now().Add(wait)
	started := time.Now()
	progress := ProgressFrom(ctx)
	progress("waiting for the human's answer in the PiCode inbox (item " + id + ")")
	lastProgress := time.Now()
	for {
		if time.Since(lastProgress) >= askProgressEvery {
			progress("still waiting for the human (" + elapsed(time.Since(started)) + ", item " + id + ")")
			lastProgress = time.Now()
		}
		status, body, err := c.Daemon.Get(ctx, "/api/inbox/"+id)
		if err == nil && status == 200 {
			var it struct {
				State    string  `json:"state"`
				Response *string `json:"response"`
			}
			if json.Unmarshal(body, &it) == nil && it.State == "done" {
				return Text(AnswerLine(it.Response))
			}
		} else if err == nil && status == 404 {
			return Fail("ask_human: item " + id + " does not exist")
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			return Text("The question is still open in the human's inbox (item " + id + "). Call ask_human again with item=\"" + id + "\" to keep waiting, or proceed without the answer.")
		}
		sleep(askPoll)
	}
}

func elapsed(d time.Duration) string {
	d = d.Round(time.Minute)
	if d < time.Hour {
		return itoa(int(d.Minutes())) + " min"
	}
	return itoa(int(d.Hours())) + " h " + itoa(int(d.Minutes())%60) + " min"
}
