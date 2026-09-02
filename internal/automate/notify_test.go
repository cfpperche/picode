package automate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildNotify(t *testing.T) {
	p := BuildNotify("Nightly tests", "done", "", 0.1234, "https://box/#/automations/a1", "All green.\n12 passed.")
	if !strings.HasPrefix(p.Text, "✅ *Nightly tests* ran · $0.12 · <https://box/#/automations/a1|Open in PiCode>\nAll green.") {
		t.Fatalf("text %q", p.Text)
	}
	if len(p.Blocks) != 2 || p.Blocks[1].Text.Text != "All green.\n12 passed." {
		t.Fatalf("blocks %+v", p.Blocks)
	}
	f := BuildNotify("Docs drift", "failed", "cost cap", 2, "", "")
	if f.Text != "❌ *Docs drift* failed · $2.00 · cost cap" || len(f.Blocks) != 1 {
		t.Fatalf("failed %q %d", f.Text, len(f.Blocks))
	}
	long := BuildNotify("x", "done", "", 0, "", strings.Repeat("a", MaxNotifySummary+50))
	if !strings.HasSuffix(long.Text, "…") || len(long.Blocks[1].Text.Text) != MaxNotifySummary+len("…") {
		t.Fatal("summary not clipped")
	}
	var round map[string]any
	if err := json.Unmarshal(long.Marshal(), &round); err != nil || round["text"] == nil {
		t.Fatal("not JSON")
	}
}
