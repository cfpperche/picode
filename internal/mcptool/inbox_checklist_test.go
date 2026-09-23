package mcptool

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// packages/pi-inbox/test/logic.test.ts and packages/pi-checklist/test,
// ported: the MCP wire files the same items and says the same words.

func TestInboxIdentityAndPayloads(t *testing.T) {
	if k, s := InboxIdentity(Identity{Agent: "helper-1"}); k != "agent" || s != "helper-1" {
		t.Fatal(k, s)
	}
	if k, s := InboxIdentity(Identity{Term: "term-9"}); k != "terminal" || s != "term-9" {
		t.Fatal(k, s)
	}
	if k, s := InboxIdentity(Identity{Agent: "a1", Term: "term-9"}); k != "agent" || s != "a1" {
		t.Fatal("agent wins", k, s)
	}
	if k, s := InboxIdentity(Identity{}); k != "system" || s != "mcp (unmanaged)" {
		t.Fatal(k, s)
	}
	p, err := NotifyPayload(Identity{Agent: "a1"}, "deploy done", "all green", "")
	if err != nil || p["kind"] != "fyi" || p["blocking"] != false || p["sourceKind"] != "agent" || p["sourceId"] != "a1" || p["reason"] != "agent notification" {
		t.Fatalf("notify = %v %v", p, err)
	}
	if _, err := NotifyPayload(Identity{}, "  ", "", ""); err == nil {
		t.Fatal("a blank title is refused")
	}
	long, _ := NotifyPayload(Identity{}, strings.Repeat("x", inboxMaxTitle+50), strings.Repeat("y", inboxMaxBody+50), "")
	if len([]rune(long["title"].(string))) != inboxMaxTitle || len([]rune(long["body"].(string))) != inboxMaxBody {
		t.Fatal("clip")
	}
	q, err := AskPayload(Identity{Agent: "a1"}, "Which port?", "8080 vs 8445")
	if err != nil || q["kind"] != "question" || q["blocking"] != true || q["reason"] != "agent needs your input" || q["title"] != "Which port?" || !strings.Contains(q["body"].(string), "8080 vs 8445") {
		t.Fatalf("ask = %v %v", q, err)
	}
	if _, err := AskPayload(Identity{}, "", ""); err == nil {
		t.Fatal("a blank question is refused")
	}
	bare, _ := AskPayload(Identity{}, "Go?", "")
	if bare["body"] != "Go?" || bare["sourceKind"] != "system" {
		t.Fatalf("bare = %v", bare)
	}
}

func TestNotifyAndAskThroughTheWire(t *testing.T) {
	d := &fakeDaemon{status: 201, body: `{"id":"in_1","state":"open"}`}
	c := &Caller{Daemon: d, Identity: Identity{Term: "t1"}}
	res := notifyCall(t.Context(), c, json.RawMessage(`{"title":"hi","body":"b"}`))
	if res.IsError || res.Content[0].Text != "Filed to the human's inbox." || d.path != "/api/inbox" || d.got["sourceKind"] != "terminal" {
		t.Fatalf("notify = %+v wire %s %v", res, d.path, d.got)
	}
	// A refusal keeps the daemon's words; unreachable is soft, never an error.
	d.status, d.body = 400, `{"error":"title is required"}`
	if res := notifyCall(t.Context(), c, json.RawMessage(`{"title":"x"}`)); res.IsError || !strings.Contains(res.Content[0].Text, "title is required") {
		t.Fatalf("refusal = %+v", res)
	}
	if res := notifyCall(t.Context(), &Caller{Unreachable: "no server.json"}, json.RawMessage(`{"title":"x"}`)); res.IsError || !strings.Contains(res.Content[0].Text, "not reachable") {
		t.Fatalf("unreachable = %+v", res)
	}

	// ask_human: files, then polls; the answer comes back as the result.
	d = &fakeDaemon{status: 201, body: `{"id":"in_2","state":"open"}`, gets: []string{`{"id":"in_2","state":"open"}`, `{"id":"in_2","state":"done","response":"respond: use 8445"}`}}
	c = &Caller{Daemon: d, Identity: Identity{Term: "t1"}}
	var slept time.Duration
	res = askCall(t.Context(), c, json.RawMessage(`{"question":"Which port?"}`), time.Minute, func(d time.Duration) { slept += d })
	if res.IsError || res.Content[0].Text != "The human answered: use 8445" || d.getCalls != 2 || slept != askPoll || d.got["blocking"] != true {
		t.Fatalf("ask = %+v gets %d slept %v", res, d.getCalls, slept)
	}
	// Still open at the deadline: the model is told the item id to resume with.
	d = &fakeDaemon{status: 201, body: `{"id":"in_3","state":"open"}`}
	d.gets = nil
	c = &Caller{Daemon: d, Identity: Identity{Term: "t1"}}
	d.status, d.body = 200, `{"id":"in_3","state":"open"}`
	res = askCall(t.Context(), c, json.RawMessage(`{"item":"in_3"}`), 0, func(time.Duration) {})
	if res.IsError || !strings.Contains(res.Content[0].Text, `item="in_3"`) || d.path != "/api/inbox/in_3?wait=1" {
		t.Fatalf("still open = %+v path %s", res, d.path)
	}
	// Closed without an answer.
	d = &fakeDaemon{status: 200, body: `{"id":"in_4","state":"done"}`}
	res = askCall(t.Context(), &Caller{Daemon: d, Identity: Identity{Agent: "a"}}, json.RawMessage(`{"item":"in_4"}`), time.Minute, func(time.Duration) {})
	if res.Content[0].Text != "The human closed the question without an answer." {
		t.Fatalf("closed = %+v", res)
	}
	ignored := "ignore"
	accepted := "accept"
	edited := "edit: 8080 then"
	if AnswerLine(&ignored) != "The human closed the question without an answer." || AnswerLine(&accepted) != "The human accepted." || AnswerLine(&edited) != "The human answered: 8080 then" {
		t.Fatal("answer lines")
	}
	// An unknown item is an error the model reads.
	d = &fakeDaemon{status: 404, body: `{"error":"not found"}`}
	if res := askCall(t.Context(), &Caller{Daemon: d, Identity: Identity{Agent: "a"}}, json.RawMessage(`{"item":"zz"}`), time.Minute, func(time.Duration) {}); !res.IsError {
		t.Fatalf("unknown = %+v", res)
	}
}

func TestChecklistNormalizeSummarizeAndPublish(t *testing.T) {
	items, err := NormalizeChecklist([]map[string]any{{"text": "  write   tests "}, {"text": "ship", "status": "in-progress"}})
	if err != nil || items[0].Text != "write tests" || items[0].Status != "pending" || items[1].Status != "in-progress" {
		t.Fatalf("normalize = %v %v", items, err)
	}
	if _, err := NormalizeChecklist(nil); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatal(err)
	}
	if _, err := NormalizeChecklist([]map[string]any{{"text": ""}}); err == nil || !strings.Contains(err.Error(), "items[0].text") {
		t.Fatal(err)
	}
	if _, err := NormalizeChecklist([]map[string]any{{"text": "x", "status": "done"}}); err == nil || !strings.Contains(err.Error(), "must be one of") {
		t.Fatal(err)
	}
	many := make([]map[string]any, checklistMaxItems+1)
	for i := range many {
		many[i] = map[string]any{"text": "s"}
	}
	if _, err := NormalizeChecklist(many); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatal(err)
	}
	long, _ := NormalizeChecklist([]map[string]any{{"text": strings.Repeat("é", checklistMaxText+5)}})
	if len([]rune(long[0].Text)) != checklistMaxText {
		t.Fatal("clip by code points")
	}
	if got := SummarizeChecklist(items); got != "Checklist saved: 0/2 completed. Current step (2/2): ship" {
		t.Fatal(got)
	}
	done := []ChecklistItem{{"a", "completed"}, {"b", "completed"}}
	if got := SummarizeChecklist(done); got != "Checklist saved: 2/2 completed. All steps completed." {
		t.Fatal(got)
	}
	pending := []ChecklistItem{{"a", "completed"}, {"b", "pending"}}
	if got := SummarizeChecklist(pending); got != "Checklist saved: 1/2 completed. Current step (2/2): b" {
		t.Fatal(got)
	}
	if ChecklistPath(Identity{Agent: "a1"}) != "/api/agents/a1/checklist" || ChecklistPath(Identity{Term: "t1"}) != "/api/terminals/t1/checklist" || ChecklistPath(Identity{}) != "" {
		t.Fatal("publish target")
	}

	d := &fakeDaemon{status: 200, body: `{"checklist":{}}`}
	res := checklistCall(t.Context(), &Caller{Daemon: d, Identity: Identity{Term: "t1"}}, json.RawMessage(`{"items":[{"text":"a","status":"in-progress"},{"text":"b"}]}`))
	if res.IsError || res.Content[0].Text != "Checklist saved: 0/2 completed. Current step (1/2): a" || d.path != "/api/terminals/t1/checklist" {
		t.Fatalf("publish = %+v path %s", res, d.path)
	}
	if list, ok := d.got["items"].([]any); !ok || len(list) != 2 || list[1].(map[string]any)["status"] != "pending" {
		t.Fatalf("wire items = %v", d.got["items"])
	}
	if res := checklistCall(t.Context(), &Caller{Daemon: d}, json.RawMessage(`{"items":[{"text":"a"}]}`)); res.IsError || !strings.Contains(res.Content[0].Text, "no PiCode identity") {
		t.Fatalf("no identity = %+v", res)
	}
	if res := checklistCall(t.Context(), &Caller{Daemon: d, Identity: Identity{Term: "t1"}}, json.RawMessage(`{"items":[]}`)); !res.IsError || !strings.Contains(res.Content[0].Text, "checklist refused") {
		t.Fatalf("empty = %+v", res)
	}
}

func TestAllFamiliesListTheirTools(t *testing.T) {
	s := NewServer("picode", "x", Families(), &Caller{Daemon: &fakeDaemon{}, Identity: Identity{Term: "t"}})
	names := []string{}
	for _, tool := range s.Tools() {
		names = append(names, tool.Name)
	}
	if strings.Join(names, ",") != "ask_human,browser,checklist,computer,delivery,mission,notify_human" {
		t.Fatalf("tools = %v", names)
	}
	if strings.Join(FamilyNames(), ",") != "computer,browser,inbox,checklist,delivery,mission" {
		t.Fatalf("families = %v", FamilyNames())
	}
}
