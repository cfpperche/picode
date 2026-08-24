package session

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// Event is one conversation beat for the chat surface (view-only).
type Event struct {
	Kind     string         `json:"kind"` // user | assistant | thinking | tool
	Text     string         `json:"text,omitempty"`
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name,omitempty"`
	Args     string         `json:"args,omitempty"`
	Status   string         `json:"status,omitempty"`
	Detail   string         `json:"detail,omitempty"`
	ToolArgs map[string]any `json:"toolArgs,omitempty"`
	Result   map[string]any `json:"result,omitempty"`
}

// Transcript reads a JSONL session into chat events (ADR-0005: read-only).
func Transcript(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Event
	pending := map[string]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		switch raw["type"] {
		case "compaction", "compaction_summary":
			sum, _ := raw["summary"].(string)
			if sum == "" {
				sum = "Session compacted."
			}
			out = append(out, Event{Kind: "assistant", Text: sum})
			continue
		case "message":
		default:
			continue
		}
		msg, _ := raw["message"].(map[string]any)
		if msg == nil {
			continue
		}
		switch msg["role"] {
		case "user":
			if t := textOf(msg["content"]); t != "" {
				out = append(out, Event{Kind: "user", Text: t})
			}
		case "assistant":
			out = appendAssistant(out, msg, pending)
		case "toolResult":
			out = applyToolResult(out, msg, pending)
		}
	}
	return out, sc.Err()
}

func appendAssistant(out []Event, msg map[string]any, pending map[string]int) []Event {
	blocks, _ := msg["content"].([]any)
	if blocks == nil {
		if t := textOf(msg["content"]); t != "" {
			out = append(out, Event{Kind: "assistant", Text: t})
		}
		return out
	}
	for _, b := range blocks {
		blk, _ := b.(map[string]any)
		if blk == nil {
			continue
		}
		switch blk["type"] {
		case "text":
			if t, _ := blk["text"].(string); strings.TrimSpace(t) != "" {
				out = append(out, Event{Kind: "assistant", Text: t})
			}
		case "thinking":
			if t, _ := blk["thinking"].(string); strings.TrimSpace(t) != "" {
				out = append(out, Event{Kind: "thinking", Text: t})
			}
		case "toolCall":
			id, _ := blk["id"].(string)
			name, _ := blk["name"].(string)
			args, _ := blk["arguments"].(map[string]any)
			ev := Event{Kind: "tool", ID: id, Name: name, Args: summarize(args), Status: "···", ToolArgs: args}
			pending[id] = len(out)
			out = append(out, ev)
		}
	}
	return out
}

func applyToolResult(out []Event, msg map[string]any, pending map[string]int) []Event {
	id, _ := msg["toolCallId"].(string)
	detail := textOf(msg["content"])
	status := "ok"
	if err, _ := msg["isError"].(bool); err {
		status = "error"
	}
	res, _ := msg["details"].(map[string]any)
	if i, ok := pending[id]; ok && i >= 0 && i < len(out) {
		out[i].Status = status
		out[i].Detail = detail
		if res != nil {
			out[i].Result = res
		}
		return out
	}
	name, _ := msg["toolName"].(string)
	return append(out, Event{Kind: "tool", ID: id, Name: name, Status: status, Detail: detail, Result: res})
}

func textOf(content any) string {
	switch c := content.(type) {
	case string:
		return strings.TrimSpace(c)
	case []any:
		var b strings.Builder
		for _, part := range c {
			p, _ := part.(map[string]any)
			if p == nil {
				continue
			}
			if p["type"] == "text" {
				if t, _ := p["text"].(string); t != "" {
					if b.Len() > 0 {
						b.WriteByte('\n')
					}
					b.WriteString(t)
				}
			}
		}
		return strings.TrimSpace(b.String())
	}
	return ""
}

func summarize(args map[string]any) string {
	if args == nil {
		return ""
	}
	if c, ok := args["command"].(string); ok {
		return c
	}
	if p, ok := args["path"].(string); ok {
		return p
	}
	b, err := json.Marshal(args)
	if err != nil || string(b) == "{}" {
		return ""
	}
	s := string(b)
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
