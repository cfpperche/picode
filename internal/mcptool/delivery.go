package mcptool

import (
	"context"
	"encoding/json"
	"fmt"
)

var deliveryFamily = Family{
	Name:         "delivery",
	Instructions: "Register a delivery and request review through PiCode. Keep the same requestId and exact payload when retrying. Registration and review requests are declarations, not passed checks, human approval, integration or deployment. Identity is the PiCode launch, not an attested native conversation. An integration queue request is a declaration too: it names your own delivery at the revision and target it declares, the owner decides whether and when it runs, and PiCode runs it through the project's own declaration of how integration runs.",
	Tools: func(c *Caller) []Tool {
		return []Tool{{
			Name: "delivery", Description: "Register, update, request/withdraw review, request or withdraw a place in the integration queue, list or inspect delivery declarations for your PiCode launch repository. No merge or deploy execution.",
			InputSchema: map[string]any{"type": "object", "required": []string{"action"}, "additionalProperties": false, "properties": map[string]any{
				"action":    map[string]any{"type": "string", "enum": []string{"capabilities", "register", "update", "request-review", "withdraw-review", "request-integration", "withdraw-integration", "show", "list"}},
				"requestId": map[string]any{"type": "string"}, "id": map[string]any{"type": "string"}, "title": map[string]any{"type": "string"},
				"branch": map[string]any{"type": "string"}, "revision": map[string]any{"type": "string"}, "target": map[string]any{"type": "string"},
				"expectedVersion": map[string]any{"type": "integer", "minimum": 1}, "before": map[string]any{"type": "integer", "minimum": 0},
			}},
			Call: func(ctx context.Context, args json.RawMessage) Result {
				raw, err := CallDelivery(ctx, c, args)
				if err != nil {
					return Fail(err.Error())
				}
				return Text(string(raw))
			},
		}}
	},
}

// CallDelivery is shared by the native shell command and optional MCP family.
// It always replaces caller-supplied identity with the inherited PiCode identity.
func CallDelivery(ctx context.Context, c *Caller, args json.RawMessage) (json.RawMessage, error) {
	if c.Identity.Empty() {
		return nil, fmt.Errorf("%s", NoIdentity)
	}
	if c.Daemon == nil || c.Unreachable != "" {
		return nil, fmt.Errorf("PiCode is unreachable; inspect connectivity before retrying with the same request ID")
	}
	var payload map[string]any
	if err := json.Unmarshal(args, &payload); err != nil || payload == nil {
		return nil, fmt.Errorf("expected a delivery request object")
	}
	delete(payload, "agent")
	delete(payload, "term")
	if c.Identity.Agent != "" {
		payload["agent"] = c.Identity.Agent
	}
	if c.Identity.Term != "" {
		payload["term"] = c.Identity.Term
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	status, raw, err := c.Daemon.Post(ctx, "/api/delivery/tool", body)
	if err != nil {
		return nil, fmt.Errorf("delivery response unavailable; retry only with the same request ID and content")
	}
	if status < 200 || status >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		if e.Error == "" {
			e.Error = fmt.Sprintf("delivery refused (HTTP %d)", status)
		}
		return nil, fmt.Errorf("%s", e.Error)
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid delivery response; outcome unknown")
	}
	return raw, nil
}
