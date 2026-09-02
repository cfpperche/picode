package automate

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NotifyPayload is the message POSTed to an automation's notify URL when
// a run ends (ADR-0045 amendment): the Slack incoming-webhook shape,
// which Discord ("/slack" webhooks), Teams and most relays accept.
// text carries the whole message for clients that ignore blocks.
type NotifyPayload struct {
	Text   string        `json:"text"`
	Blocks []notifyBlock `json:"blocks,omitempty"`
}

type notifyBlock struct {
	Type string      `json:"type"`
	Text *notifyText `json:"text,omitempty"`
}

type notifyText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MaxNotifySummary keeps a chat message a message.
const MaxNotifySummary = 1500

// BuildNotify renders the payload. status is done | failed | skipped;
// reason explains a non-done run; summary is the agent's final message.
func BuildNotify(name, status, reason string, costUSD float64, link, summary string) NotifyPayload {
	icon, verb := "✅", "ran"
	switch status {
	case "failed":
		icon, verb = "❌", "failed"
	case "skipped":
		icon, verb = "⏭️", "was skipped"
	}
	head := fmt.Sprintf("%s *%s* %s", icon, name, verb)
	if costUSD > 0 {
		head += fmt.Sprintf(" · $%.2f", costUSD)
	}
	if reason != "" {
		head += " · " + reason
	}
	if link != "" {
		head += " · <" + link + "|Open in PiCode>"
	}
	body := strings.TrimSpace(summary)
	if len(body) > MaxNotifySummary {
		body = body[:MaxNotifySummary] + "…"
	}
	text := head
	if body != "" {
		text += "\n" + body
	}
	p := NotifyPayload{Text: text, Blocks: []notifyBlock{{Type: "section", Text: &notifyText{Type: "mrkdwn", Text: head}}}}
	if body != "" {
		p.Blocks = append(p.Blocks, notifyBlock{Type: "section", Text: &notifyText{Type: "mrkdwn", Text: body}})
	}
	return p
}

// Marshal is the JSON body.
func (p NotifyPayload) Marshal() []byte {
	b, _ := json.Marshal(p)
	return b
}
