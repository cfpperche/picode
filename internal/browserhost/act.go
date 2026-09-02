package browserhost

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ActMaxRounds caps the send → act → result loop (ADR-0054). One round is
// a turn that ended in a picode-act block; the cap is the boundary
// against MV3-as-Playwright.
const ActMaxRounds = 3

// ActMaxActions is the per-batch cap the parser enforces.
const ActMaxActions = 12

// Act is one step the extension may perform on the granted tab.
type Act struct {
	Act      string `json:"act"`                // click | fill | press | read | scroll
	Selector string `json:"selector,omitempty"` // CSS
	Value    string `json:"value,omitempty"`    // fill
	Key      string `json:"key,omitempty"`      // press, default Enter
	To       string `json:"to,omitempty"`       // scroll: top | bottom
}

// ActBatch is a validated picode-act block.
type ActBatch struct {
	Actions []Act `json:"actions"`
}

// ActBatchWire is what travels to the extension: the batch plus its
// identity and bounds.
type ActBatchWire struct {
	ID      string `json:"id"`
	AgentID string `json:"agentId"`
	Origin  string `json:"origin"`
	Round   int    `json:"round"`
	Rounds  int    `json:"rounds"`
	Actions []Act  `json:"actions"`
}

// ActOutcome is one executed action, reported back to the agent.
type ActOutcome struct {
	Act      string `json:"act"`
	Selector string `json:"selector,omitempty"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Text     string `json:"text,omitempty"` // read
}

// ActIntro is appended to a send that asked for actuation: it tells the
// model the one fenced block it may reply with.
const ActIntro = `[browser-act]
You may act on this page. When actions are useful, end your reply with ONE fenced block:
` + "```picode-act" + `
{"actions":[{"act":"click","selector":"button[type=submit]"},{"act":"fill","selector":"#login","value":"goat"},{"act":"press","selector":"#q","key":"Enter"},{"act":"read","selector":"main"},{"act":"scroll","to":"bottom"}]}
` + "```" + `
Rules: CSS selectors only; at most 12 actions; acts are click, fill, press, read, scroll.
The human approves and runs them; their result returns as [browser-act result].
Omit the block to answer without acting.`

// ParseActBlock extracts and validates the LAST ```picode-act fenced JSON
// block from an assistant reply. No block is not an error — the loop
// simply ends (ok=false, err=nil).
func ParseActBlock(text string) (ActBatch, bool, error) {
	const fence = "```picode-act"
	idx := strings.LastIndex(text, fence)
	if idx < 0 {
		return ActBatch{}, false, nil
	}
	rest := text[idx+len(fence):]
	nl := strings.IndexByte(rest, '\n')
	if nl >= 0 {
		rest = rest[nl+1:]
	}
	end := strings.Index(rest, "```")
	if end < 0 {
		return ActBatch{}, false, fmt.Errorf("picode-act block is not closed")
	}
	var batch ActBatch
	if err := json.Unmarshal([]byte(strings.TrimSpace(rest[:end])), &batch); err != nil {
		return ActBatch{}, false, fmt.Errorf("picode-act block is not valid JSON")
	}
	if len(batch.Actions) == 0 {
		return ActBatch{}, false, fmt.Errorf("picode-act block has no actions")
	}
	if len(batch.Actions) > ActMaxActions {
		return ActBatch{}, false, fmt.Errorf("picode-act block has %d actions (max %d)", len(batch.Actions), ActMaxActions)
	}
	for i, a := range batch.Actions {
		if err := validateAct(a); err != nil {
			return ActBatch{}, false, fmt.Errorf("action %d: %w", i+1, err)
		}
	}
	return batch, true, nil
}

func validateAct(a Act) error {
	switch a.Act {
	case "click", "read":
		if strings.TrimSpace(a.Selector) == "" {
			return fmt.Errorf("%s needs a selector", a.Act)
		}
	case "fill":
		if strings.TrimSpace(a.Selector) == "" {
			return fmt.Errorf("fill needs a selector")
		}
		if len(a.Value) > 2000 {
			return fmt.Errorf("fill value is over 2000 characters")
		}
	case "press":
		if strings.TrimSpace(a.Selector) == "" {
			return fmt.Errorf("press needs a selector")
		}
		if a.Key != "" && len(a.Key) > 20 {
			return fmt.Errorf("press key is over 20 characters")
		}
	case "scroll":
		switch a.To {
		case "top", "bottom":
		default:
			return fmt.Errorf("scroll needs to=top|bottom")
		}
	default:
		return fmt.Errorf("unknown act %q", a.Act)
	}
	if len(a.Selector) > 300 {
		return fmt.Errorf("selector is over 300 characters")
	}
	return nil
}

// ComposeActResult is the follow-up prompt that carries outcomes back to
// the agent. Read text is clipped so a huge page cannot eat the context.
func ComposeActResult(round int, outcomes []ActOutcome, stopped bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[browser-act result] round %d of %d\n", round, ActMaxRounds)
	if stopped {
		b.WriteString("The human stopped the loop.\n")
		return b.String()
	}
	for _, o := range outcomes {
		label := o.Act
		if o.Selector != "" {
			label += " " + o.Selector
		}
		if o.OK {
			b.WriteString(label + ": ok\n")
		} else {
			b.WriteString(label + ": error: " + o.Error + "\n")
		}
		if o.Text != "" {
			b.WriteString("text:\n" + clipActText(o.Text) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func clipActText(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 1500 {
		return s[:1500] + "…"
	}
	return s
}
