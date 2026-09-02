package store

import (
	"encoding/json"
	"testing"
	"time"
)

func TestActBatchLifecycle(t *testing.T) {
	s := openTest(t)

	// Empty agent: nothing pending.
	if _, ok, _ := s.PendingActBatch("a1"); ok {
		t.Fatal("empty store had a batch")
	}

	first, err := s.CreateActBatch("a1", "https://x.com", `[{"act":"read","selector":"main"}]`, 1)
	if err != nil {
		t.Fatal(err)
	}

	// A second create drops the first pending batch (one pending per agent).
	second, err := s.CreateActBatch("a1", "https://x.com", `[{"act":"scroll","to":"bottom"}]`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != ActPending { // the in-memory copy is unchanged
		t.Fatal("CreateActBatch mutated its return value")
	}

	claimed, ok, err := s.ClaimActBatch("a1")
	if err != nil || !ok {
		t.Fatalf("claim = %v %v", ok, err)
	}
	if claimed.ID != second.ID || claimed.State != ActClaimed {
		t.Fatalf("claimed %+v", claimed)
	}

	// Claiming again returns the same claimed batch (idempotent for the panel).
	again, ok, err := s.ClaimActBatch("a1")
	if err != nil || !ok || again.ID != second.ID {
		t.Fatalf("re-claim = %+v %v %v", again, ok, err)
	}

	// A stale pending batch expires on claim.
	old, err := s.CreateActBatch("a2", "https://y.io", `[]`, 1)
	if err != nil {
		t.Fatal(err)
	}
	stale := time.Now().UTC().Add(-11 * time.Minute).Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`UPDATE act_batches SET created_at = ? WHERE id = ?`, stale, old.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.ClaimActBatch("a2"); err != nil || ok {
		t.Fatalf("stale claim = %v %v", ok, err)
	}
	if b, _, _ := s.GetActBatchRow(old.ID); b.State != ActExpired {
		t.Fatalf("stale state %q", b.State)
	}

	// Finish.
	if err := s.FinishActBatch(second.ID); err != nil {
		t.Fatal(err)
	}
	if b, _, _ := s.GetActBatchRow(second.ID); b.State != ActDone {
		t.Fatalf("done state %q", b.State)
	}

	// ExpirePendingActBatches clears pending for the agent.
	if _, err := s.CreateActBatch("a3", "https://z.dev", `[]`, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.ExpirePendingActBatches("a3"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.PendingActBatch("a3"); ok {
		t.Fatal("pending survived expire")
	}
}

func TestDecodeActActions(t *testing.T) {
	raw, err := json.Marshal([]map[string]any{{"act": "click", "selector": "#a"}})
	if err != nil {
		t.Fatal(err)
	}
	acts, err := DecodeActActions(string(raw))
	if err != nil || len(acts) != 1 {
		t.Fatalf("%v %v", acts, err)
	}
	if _, err := DecodeActActions("nope"); err == nil {
		t.Fatal("bad json accepted")
	}
}
