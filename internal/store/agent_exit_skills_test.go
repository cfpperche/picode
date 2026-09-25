package store

import (
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// ADR-0196 slice 6: an exit keeps the skill set its run loaded (read at the
// exit when no launch recorded one), and Outcomes compares each skill's exits
// with the recorded exits that did not load it. An exit with no record is on
// neither side.
func TestExitSkillStats(t *testing.T) {
	s := openTest(t)
	use := func(names ...string) func(Agent) []clilaunch.SkillUse {
		return func(Agent) []clilaunch.SkillUse {
			out := []clilaunch.SkillUse{}
			for _, n := range names {
				out = append(out, clilaunch.SkillUse{Name: n, Digest: "d-" + n, Scope: "machine", Via: "~/.agents/skills"})
			}
			return out
		}
	}
	rows := []struct {
		skills  func(Agent) []clilaunch.SkillUse
		outcome string
	}{
		{use("pdf", "review"), ExitResolved},
		{use("pdf"), ExitResolved},
		{use("pdf"), ExitUnresolved},
		{use("review"), ExitPartial},
		{use(), ""},
		{nil, ExitResolved}, // no record: counts nowhere
	}
	for i, r := range rows {
		a, err := s.AddAgent(FreeWorkspaceID, "a", "")
		if err != nil {
			t.Fatal(err)
		}
		ex, err := s.RemoveAgentWithExit(a.ID, ExitInput{Skills: r.skills, Label: ExitLabel{Outcome: r.outcome}})
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if (ex.Config.Loaded != nil) != (r.skills != nil) || (ex.Config.Loaded != nil && ex.Config.Loaded.Source != "exit") {
			t.Fatalf("%d: loaded %+v", i, ex.Config.Loaded)
		}
	}
	st, err := s.AgentExitSkillStats(ExitFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Recorded != 5 || st.Unrecorded != 1 || len(st.Rows) != 2 {
		t.Fatalf("%+v", st)
	}
	pdf := st.Rows[0]
	if pdf.Name != "pdf" || pdf.Versions != 1 ||
		pdf.With != (ExitSkillSide{Total: 3, Resolved: 2, Unresolved: 1}) ||
		pdf.Without != (ExitSkillSide{Total: 2, Partial: 1, Unanswered: 1}) {
		t.Fatalf("pdf %+v", pdf)
	}
	review := st.Rows[1]
	if review.With.Total != 2 || review.Without.Total != 3 || review.Without.Resolved != 1 {
		t.Fatalf("review %+v", review)
	}
}
