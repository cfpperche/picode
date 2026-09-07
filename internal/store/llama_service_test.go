package store

import (
	"encoding/json"
	"testing"
)

func TestLlamaServiceRevision(t *testing.T) {
	s := openTest(t)
	raw, rev, err := s.LlamaService()
	if err != nil || rev != 0 || string(raw) != "{}" {
		t.Fatal(string(raw), rev, err)
	}
	rev, err = s.SaveLlamaService(json.RawMessage(`{"created":true}`), 0)
	if err != nil || rev != 1 {
		t.Fatal(rev, err)
	}
	if _, err = s.SaveLlamaService(json.RawMessage(`{}`), 0); err == nil {
		t.Fatal("stale creation accepted")
	}
	if _, err = s.SaveLlamaService(json.RawMessage(`invalid`), 1); err == nil {
		t.Fatal("invalid document accepted")
	}
	rev, err = s.SaveLlamaService(json.RawMessage(`{"updated":true}`), 1)
	if err != nil || rev != 2 {
		t.Fatal(rev, err)
	}
	raw, rev, err = s.LlamaService()
	if err != nil || rev != 2 || string(raw) != `{"updated":true}` {
		t.Fatal(string(raw), rev, err)
	}
}
