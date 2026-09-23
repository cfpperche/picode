package store

import "testing"

// A delivery row wants to say which objective it serves. The mission owns that
// reference — it cites the delivery as evidence — so this reads it backwards.
func TestMissionsByDeliveryInvertsTheEvidenceLink(t *testing.T) {
	s := openTest(t)
	ws, err := s.AddWorkspace("project", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := s.ApplyMission(MissionOwner, MissionMutation{
		Action: "create", RequestID: "create", WorkspaceID: ws.ID, FilesReady: true,
		Title: "Ship the queue", Objective: "Deliver the queue end to end",
		Criteria: []MissionCriterion{{ID: "c1", Text: "the queue runs"}},
	}, MissionObservation{})
	if err != nil {
		t.Fatal(err)
	}
	// Two criteria pointing at the same change are still one objective.
	for i, criterion := range []string{"c1", "c1"} {
		if _, err := s.ApplyMission(MissionOwner, MissionMutation{
			Action: "evidence", RequestID: "e" + string(rune('0'+i)), ID: m.ID, ExpectedVersion: m.Version + i,
			Evidence: &MissionEvidence{CriterionID: criterion, Kind: "delivery", Value: "delivery_abc", Outcome: "pass"},
		}, MissionObservation{}); err != nil {
			t.Fatalf("evidence %d: %v", i, err)
		}
	}
	links, truncated, err := s.MissionsByDelivery(ws.ID)
	if err != nil || truncated {
		t.Fatalf("links = %v (truncated %v, err %v)", links, truncated, err)
	}
	got := links["delivery_abc"]
	if len(got) != 1 || got[0].ID != m.ID || got[0].Title != "Ship the queue" || got[0].State == "" {
		t.Fatalf("delivery_abc = %+v", got)
	}
	if other := links["delivery_other"]; len(other) != 0 {
		t.Fatalf("an uncited delivery has links: %+v", other)
	}
	// A workspace with no missions answers with an empty map, not an error.
	empty, truncated, err := s.MissionsByDelivery("ws_missing")
	if err != nil || truncated || len(empty) != 0 {
		t.Fatalf("empty workspace = %v (%v, %v)", empty, truncated, err)
	}
}
