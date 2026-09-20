package mcptool

import (
	"context"
	"encoding/json"
	"strings"
)

// The checklist family (ADR-0055 over MCP, ADR-0154 N1): the model writes
// its plan, PiCode shows the current step on the terminal's (or agent's)
// card. Text and payload mirror packages/pi-checklist. What does not carry
// over: pi's gate (mutations refused before a plan) and the end-of-turn
// reminder are extension hooks MCP has no equivalent for; the contract
// travels as the server's instructions instead.

const (
	checklistMaxItems = 30
	checklistMaxText  = 200
)

var checklistStatuses = []string{"pending", "in-progress", "completed"}

var checklistGuidelines = []string{
	"Call checklist before the first edit, write or bash of a task, with 2–8 concrete steps, the first one in-progress.",
	"Send the whole list every time: mark the finished step completed and the next one in-progress in one call.",
	"Keep steps concrete and short (one line each); add steps you discover, never delete history.",
}

// ChecklistItem is one step as PiCode stores it.
type ChecklistItem struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

var checklistFamily = Family{
	Name:         "checklist",
	Instructions: "checklist:\n- " + strings.Join(checklistGuidelines, "\n- "),
	Tools: func(c *Caller) []Tool {
		return []Tool{{
			Name: "checklist",
			Description: "Write or update your internal checklist for the current task: the whole list of concrete steps with their status. " +
				"Call it before your first change, and again whenever a step starts or finishes. The human follows it from the sidebar.",
			InputSchema: map[string]any{
				"type": "object", "required": []string{"items"},
				"properties": map[string]any{
					"items": map[string]any{
						"type": "array", "description": "The whole checklist, in order",
						"items": map[string]any{
							"type": "object", "required": []string{"text"},
							"properties": map[string]any{
								"text":   map[string]any{"type": "string", "description": "One concrete step, one line"},
								"status": map[string]any{"type": "string", "enum": checklistStatuses, "description": "pending (default), in-progress, completed"},
							},
						},
					},
				},
			},
			Call: func(ctx context.Context, args json.RawMessage) Result { return checklistCall(ctx, c, args) },
		}}
	},
}

// NormalizeChecklist is pi-checklist's normalizeItems: the message it
// returns is one the model can act on.
func NormalizeChecklist(raw []map[string]any) ([]ChecklistItem, error) {
	if len(raw) == 0 {
		return nil, errString("items must hold at least one step")
	}
	if len(raw) > checklistMaxItems {
		return nil, errString("items must hold at most " + itoa(checklistMaxItems) + " steps")
	}
	out := make([]ChecklistItem, 0, len(raw))
	for i, it := range raw {
		text := collapseSpace(stringOf(it["text"]))
		if text == "" {
			return nil, errString("items[" + itoa(i) + "].text is required")
		}
		status := "pending"
		if v, ok := it["status"]; ok && v != nil {
			s, _ := v.(string)
			valid := false
			for _, known := range checklistStatuses {
				valid = valid || known == s
			}
			if !valid {
				return nil, errString("items[" + itoa(i) + "].status must be one of " + strings.Join(checklistStatuses, ", "))
			}
			status = s
		}
		out = append(out, ChecklistItem{Text: clip(text, checklistMaxText), Status: status})
	}
	return out, nil
}

// SummarizeChecklist is pi-checklist's summarize.
func SummarizeChecklist(items []ChecklistItem) string {
	done := 0
	for _, it := range items {
		if it.Status == "completed" {
			done++
		}
	}
	head := "Checklist saved: " + itoa(done) + "/" + itoa(len(items)) + " completed."
	if len(items) == 0 || done == len(items) {
		return head + " All steps completed."
	}
	pos := -1
	for i, it := range items {
		if it.Status == "in-progress" {
			pos = i
			break
		}
	}
	if pos < 0 {
		for i, it := range items {
			if it.Status == "pending" {
				pos = i
				break
			}
		}
	}
	if pos < 0 {
		pos = len(items) - 1
	}
	return head + " Current step (" + itoa(pos+1) + "/" + itoa(len(items)) + "): " + items[pos].Text
}

// ChecklistPath is pi-checklist's publishTarget: the agent's route, else
// the terminal's, else nothing to publish to.
func ChecklistPath(id Identity) string {
	if id.Agent != "" {
		return "/api/agents/" + id.Agent + "/checklist"
	}
	if id.Term != "" {
		return "/api/terminals/" + id.Term + "/checklist"
	}
	return ""
}

func checklistCall(ctx context.Context, c *Caller, args json.RawMessage) Result {
	var p struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return Fail("checklist refused: items must be an array of {text, status}")
	}
	items, err := NormalizeChecklist(p.Items)
	if err != nil {
		return Fail("checklist refused: " + err.Error())
	}
	summary := SummarizeChecklist(items)
	path := ChecklistPath(c.Identity)
	if path == "" {
		return Text(summary + " (Not shown in PiCode: this CLI has no PiCode identity — open it in a PiCode terminal.)")
	}
	if c.Unreachable != "" || c.Daemon == nil {
		return Text(summary + " (Not shown in PiCode: it is not reachable.)")
	}
	raw, _ := json.Marshal(map[string]any{"items": items})
	if status, body, err := c.Daemon.Post(ctx, path, raw); err != nil || status < 200 || status >= 300 {
		note := "it is not reachable"
		if err == nil {
			var parsed struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(body, &parsed) == nil && parsed.Error != "" {
				note = parsed.Error
			} else {
				note = "HTTP " + itoa(status)
			}
		}
		return Text(summary + " (Not shown in PiCode: " + note + ".)")
	}
	return Text(summary)
}
