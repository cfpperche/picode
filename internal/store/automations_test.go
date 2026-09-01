package store

import (
	"testing"
	"time"
)

func TestCreateAutomationDefaults(t *testing.T) {
	s := openTest(t)
	a, secret, err := s.CreateAutomation(AutomationParams{Name: " Nightly ", Action: AutomationStart, Prompt: "hi", Cron: "0  9 * * 1-5", Webhook: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Name != "Nightly" || !a.Enabled || a.WorkspaceID != FreeWorkspaceID || *a.Cron != "0 9 * * 1-5" || !a.Webhook {
		t.Fatalf("defaults: %+v", a)
	}
	if len(secret) != 64 {
		t.Fatalf("secret length %d", len(secret))
	}
	got, err := s.GetAutomation(a.ID)
	if err != nil || !got.Webhook || got.MaxCostUSD != nil || got.MaxRuns != nil {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, ok, _ := s.VerifyWebhookSecret(a.ID, secret); !ok {
		t.Fatal("secret must verify")
	}
	if _, ok, _ := s.VerifyWebhookSecret(a.ID, "nope"); ok {
		t.Fatal("wrong secret verified")
	}
	if _, ok, _ := s.VerifyWebhookSecret(a.ID, ""); ok {
		t.Fatal("empty secret verified")
	}
}

func TestCreateAutomationRejects(t *testing.T) {
	s := openTest(t)
	ok := AutomationParams{Name: "x", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"}
	bad := []struct {
		name string
		mut  func(*AutomationParams)
	}{
		{"no name", func(p *AutomationParams) { p.Name = " " }},
		{"long name", func(p *AutomationParams) { p.Name = string(make([]byte, 61)) }},
		{"bad action", func(p *AutomationParams) { p.Action = "run" }},
		{"message without target", func(p *AutomationParams) { p.Action = AutomationMessage }},
		{"no prompt", func(p *AutomationParams) { p.Prompt = "" }},
		{"bad cron", func(p *AutomationParams) { p.Cron = "* * *" }},
		{"no trigger", func(p *AutomationParams) { p.Cron = ""; p.Webhook = false }},
		{"negative cost", func(p *AutomationParams) { p.MaxCostUSD = -1 }},
		{"runs without window", func(p *AutomationParams) { p.MaxRuns = 3 }},
	}
	for _, b := range bad {
		p := ok
		b.mut(&p)
		if _, _, err := s.CreateAutomation(p); err == nil {
			t.Errorf("%s: accepted", b.name)
		}
	}
	if _, _, err := s.CreateAutomation(ok); err != nil {
		t.Fatalf("baseline rejected: %v", err)
	}
}

func TestUpdateAutomationAndWebhookToggle(t *testing.T) {
	s := openTest(t)
	a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
	off, cost, runs, win := false, 0.5, 3, 60
	a, err := s.UpdateAutomation(a.ID, AutomationPatch{Enabled: &off, MaxCostUSD: &cost, MaxRuns: &runs, MaxRunsWindowMin: &win})
	if err != nil || a.Enabled || *a.MaxCostUSD != 0.5 || *a.MaxRuns != 3 {
		t.Fatalf("patch: %+v %v", a, err)
	}
	zero := 0.0
	a, _ = s.UpdateAutomation(a.ID, AutomationPatch{MaxCostUSD: &zero})
	if a.MaxCostUSD != nil {
		t.Fatal("zero must clear the cap")
	}
	empty := ""
	if _, err := s.UpdateAutomation(a.ID, AutomationPatch{Cron: &empty}); err == nil {
		t.Fatal("clearing the only trigger must fail")
	}
	secret, err := s.SetAutomationWebhook(a.ID, true)
	if err != nil || secret == "" {
		t.Fatalf("webhook on: %v", err)
	}
	if _, err := s.UpdateAutomation(a.ID, AutomationPatch{Cron: &empty}); err != nil {
		t.Fatalf("clearing cron with webhook on: %v", err)
	}
	if _, err := s.SetAutomationWebhook(a.ID, false); err == nil {
		t.Fatal("removing the last trigger must fail")
	}
	second, _ := s.SetAutomationWebhook(a.ID, true)
	if _, ok, _ := s.VerifyWebhookSecret(a.ID, secret); ok {
		t.Fatal("rotated secret must not verify")
	}
	if _, ok, _ := s.VerifyWebhookSecret(a.ID, second); !ok {
		t.Fatal("new secret must verify")
	}
}

func TestRunsLifecycle(t *testing.T) {
	s := openTest(t)
	a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
	if r, _ := s.RunningRun(a.ID); r != nil {
		t.Fatal("nothing running yet")
	}
	r1, err := s.CreateRun(a.ID, TriggerSchedule, RunRunning, "")
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if r, _ := s.RunningRun(a.ID); r == nil || r.ID != r1.ID {
		t.Fatal("running run not found")
	}
	if _, err := s.CreateRun(a.ID, TriggerManual, RunSkipped, "busy"); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountRunsSince(a.ID, time.Now().Add(-time.Hour)); n != 1 {
		t.Fatalf("skipped must not count: %d", n)
	}
	_ = s.SetRunSession(r1.ID, "/tmp/s.jsonl")
	if err := s.FinishRun(r1.ID, RunDone, "", 0.42); err != nil {
		t.Fatal(err)
	}
	_ = s.FinishRun(r1.ID, RunFailed, "late", 9) // second finish is a no-op
	got, _ := s.GetRun(r1.ID)
	if got.Status != RunDone || got.CostUSD != 0.42 || got.FinishedAt == nil || *got.SessionPath != "/tmp/s.jsonl" {
		t.Fatalf("finished run: %+v", got)
	}
	runs, _ := s.ListRuns(a.ID, 0)
	if len(runs) != 2 || runs[0].Trigger != TriggerManual {
		t.Fatalf("list order: %+v", runs)
	}
	last, _ := s.LastRun(a.ID)
	if last == nil || last.Reason != "busy" {
		t.Fatalf("last run: %+v", last)
	}
	counts, _ := s.RunCountsByDay(a.ID, 7, time.Now())
	if len(counts) != 7 || counts[6] != 1 {
		t.Fatalf("day counts: %v", counts)
	}
	if _, err := s.CreateRun(a.ID, "cosmic", RunRunning, ""); err == nil {
		t.Fatal("bad trigger accepted")
	}
	if err := s.DeleteAutomation(a.ID); err != nil {
		t.Fatal(err)
	}
	if runs, _ := s.ListRuns(a.ID, 0); len(runs) != 0 {
		t.Fatal("runs must cascade")
	}
	if err := s.DeleteAutomation(a.ID); err != ErrNotFound {
		t.Fatalf("second delete = %v", err)
	}
}

func TestFailStaleRuns(t *testing.T) {
	s := openTest(t)
	a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
	r, _ := s.CreateRun(a.ID, TriggerSchedule, RunRunning, "")
	_, _ = s.CreateRun(a.ID, TriggerSchedule, RunDone, "")
	_ = s.SetRunSession(r.ID, "/tmp/run.jsonl")
	stale, err := s.FailStaleRuns("daemon restarted", func(p string) float64 {
		if p == "/tmp/run.jsonl" {
			return 0.25
		}
		return 0
	})
	if err != nil || len(stale) != 1 || stale[0].ID != r.ID {
		t.Fatalf("stale: %+v %v", stale, err)
	}
	got, _ := s.GetRun(r.ID)
	if got.Status != RunFailed || got.Reason != "daemon restarted" || got.CostUSD != 0.25 {
		t.Fatalf("stale run: %+v", got)
	}
}

func TestInboxAcceptsAutomationSource(t *testing.T) {
	s := openTest(t)
	if _, err := s.CreateInboxItem(InboxItemParams{Kind: InboxResult, SourceKind: InboxFromAutomation, SourceID: "aut-1", Reason: "automation finished", Title: "Nightly ran", Body: "pong"}); err != nil {
		t.Fatalf("automation source rejected: %v", err)
	}
}
