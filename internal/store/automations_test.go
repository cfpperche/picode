package store

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestCreateAutomationDefaults(t *testing.T) {
	s := openTest(t)
	a, secret, err := s.CreateAutomation(AutomationParams{Name: " Nightly ", Action: AutomationStart, Prompt: "hi", Cron: "0  9 * * 1-5", Webhook: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Name != "Nightly" || !a.Enabled || a.WorkspaceID != FreeWorkspaceID || len(a.Schedules) != 1 || a.Schedules[0].Cron != "0 9 * * 1-5" || !a.Schedules[0].Enabled || a.Schedules[0].TZ != "" || !a.Webhook {
		t.Fatalf("defaults: %+v", a)
	}
	if len(secret) != 64 {
		t.Fatalf("secret length %d", len(secret))
	}
	got, err := s.GetAutomation(a.ID)
	if err != nil || !got.Webhook || got.MaxCostUSD != nil || got.MaxRuns != nil || len(got.Schedules) != 1 || got.Schedules[0].AutomationID != a.ID {
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
		{"bad schedule cron", func(p *AutomationParams) { p.Schedules = []ScheduleParams{{Cron: "0 9 * *", Enabled: true}} }},
		{"unknown zone", func(p *AutomationParams) {
			p.Schedules = []ScheduleParams{{Cron: "0 9 * * *", TZ: "Mars/Olympus", Enabled: true}}
		}},
		{"long label", func(p *AutomationParams) {
			p.Schedules = []ScheduleParams{{Cron: "0 9 * * *", Label: string(make([]byte, 41)), Enabled: true}}
		}},
		{"same rule twice", func(p *AutomationParams) {
			p.Schedules = []ScheduleParams{{Cron: "0 9 * * *", Enabled: true}, {Cron: "0  9 * * *", Enabled: false}}
		}},
		{"unknown schedule id", func(p *AutomationParams) {
			p.Schedules = []ScheduleParams{{ID: "sch-nope", Cron: "0 9 * * *", Enabled: true}}
		}},
		{"too many schedules", func(p *AutomationParams) {
			for i := 0; i < 11; i++ {
				p.Schedules = append(p.Schedules, ScheduleParams{Cron: "0 " + string(rune('0'+i%10)) + " " + string(rune('1'+i/10)) + " * *", Enabled: true})
			}
		}},
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
	r1, err := s.CreateRun(RunParams{AutomationID: a.ID, Trigger: TriggerSchedule, Status: RunRunning, Reason: ""})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if r, _ := s.RunningRun(a.ID); r == nil || r.ID != r1.ID {
		t.Fatal("running run not found")
	}
	if _, err := s.CreateRun(RunParams{AutomationID: a.ID, Trigger: TriggerManual, Status: RunSkipped, Reason: "busy"}); err != nil {
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
	if _, err := s.CreateRun(RunParams{AutomationID: a.ID, Trigger: "cosmic", Status: RunRunning}); err == nil {
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
	r, _ := s.CreateRun(RunParams{AutomationID: a.ID, Trigger: TriggerSchedule, Status: RunRunning, Reason: ""})
	_, _ = s.CreateRun(RunParams{AutomationID: a.ID, Trigger: TriggerSchedule, Status: RunDone, Reason: ""})
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

func TestAutomationNotifyURL(t *testing.T) {
	s := openTest(t)
	a, _, err := s.CreateAutomation(AutomationParams{Name: "n", Action: "start", Prompt: "p", Cron: "0 7 * * *", NotifyURL: " https://hooks.slack.com/services/x "})
	if err != nil || a.NotifyURL == nil || *a.NotifyURL != "https://hooks.slack.com/services/x" {
		t.Fatalf("%+v %v", a.NotifyURL, err)
	}
	if _, _, err := s.CreateAutomation(AutomationParams{Name: "bad", Action: "start", Prompt: "p", Cron: "0 7 * * *", NotifyURL: "hooks.slack.com/x"}); err == nil {
		t.Fatal("bare host accepted")
	}
	empty := ""
	b, err := s.UpdateAutomation(a.ID, AutomationPatch{NotifyURL: &empty})
	if err != nil || b.NotifyURL != nil {
		t.Fatalf("clear: %+v %v", b.NotifyURL, err)
	}
	got, _ := s.GetAutomation(a.ID)
	if got.NotifyURL != nil {
		t.Fatal("not persisted")
	}
}

func TestAutomationSchedules(t *testing.T) {
	s := openTest(t)
	a, _, err := s.CreateAutomation(AutomationParams{Name: "two", Action: AutomationStart, Prompt: "p", Schedules: []ScheduleParams{
		{Label: " Morning ", Cron: "0  9 * * 1-5", TZ: "America/Sao_Paulo", Enabled: true},
		{Label: "Saturday", Cron: "0 12 * * 6", TZ: "America/Sao_Paulo", Enabled: false},
	}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(a.Schedules) != 2 || a.Schedules[0].Label != "Morning" || a.Schedules[0].Cron != "0 9 * * 1-5" ||
		a.Schedules[0].Position != 0 || a.Schedules[1].Position != 1 || a.Schedules[1].Enabled ||
		a.Schedules[0].Location().String() != "America/Sao_Paulo" {
		t.Fatalf("schedules: %+v", a.Schedules)
	}
	// Schedules win over the Cron convenience.
	both, _, err := s.CreateAutomation(AutomationParams{Name: "both", Action: AutomationStart, Prompt: "p", Cron: "0 1 * * *",
		Schedules: []ScheduleParams{{Cron: "0 2 * * *", Enabled: true}}})
	if err != nil || len(both.Schedules) != 1 || both.Schedules[0].Cron != "0 2 * * *" {
		t.Fatalf("both: %+v %v", both.Schedules, err)
	}
	// List carries every automation's rows, in position order.
	list, _ := s.ListAutomations()
	for _, it := range list {
		if it.ID == a.ID && (len(it.Schedules) != 2 || it.Schedules[1].Label != "Saturday") {
			t.Fatalf("list schedules: %+v", it.Schedules)
		}
	}

	// A fire stamps the row, and only that row.
	first := a.Schedules[0]
	fired := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	if err := s.TouchScheduleFired(first.ID, fired); err != nil {
		t.Fatal(err)
	}
	if err := s.TouchScheduleFired("sch-nope", fired); err != ErrNotFound {
		t.Fatalf("touch unknown = %v", err)
	}
	a, _ = s.GetAutomation(a.ID)
	if a.Schedules[0].LastFiredAt == nil || a.Schedules[1].LastFiredAt != nil {
		t.Fatalf("touch: %+v", a.Schedules)
	}

	// Reconcile: keep the first (same rule → last fire survives, label
	// and switch move), edit the second's cron (starts over), add a third.
	on := true
	a, err = s.UpdateAutomation(a.ID, AutomationPatch{Schedules: &[]ScheduleParams{
		{ID: a.Schedules[1].ID, Label: "Saturday", Cron: "30 12 * * 6", TZ: "America/Sao_Paulo", Enabled: true},
		{ID: first.ID, Label: "Weekday morning", Cron: first.Cron, TZ: first.TZ, Enabled: false},
		{Label: "Nightly", Cron: "0 23 * * *", Enabled: true},
	}, Enabled: &on})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(a.Schedules) != 3 || a.Schedules[0].Cron != "30 12 * * 6" || a.Schedules[0].LastFiredAt != nil ||
		a.Schedules[1].ID != first.ID || a.Schedules[1].LastFiredAt == nil || a.Schedules[1].Enabled ||
		a.Schedules[1].Label != "Weekday morning" || a.Schedules[1].CreatedAt != first.CreatedAt ||
		a.Schedules[2].Label != "Nightly" || a.Schedules[2].Position != 2 {
		t.Fatalf("reconciled: %+v", a.Schedules)
	}
	if _, err := s.UpdateAutomation(a.ID, AutomationPatch{Schedules: &[]ScheduleParams{{ID: "sch-nope", Cron: "0 1 * * *", Enabled: true}}}); err == nil {
		t.Fatal("unknown id accepted")
	}
	// Dropping every schedule without a webhook is refused; with one it is fine.
	none := []ScheduleParams{}
	if _, err := s.UpdateAutomation(a.ID, AutomationPatch{Schedules: &none}); err == nil {
		t.Fatal("clearing the only trigger must fail")
	}
	if _, err := s.SetAutomationWebhook(a.ID, true); err != nil {
		t.Fatal(err)
	}
	a, err = s.UpdateAutomation(a.ID, AutomationPatch{Schedules: &none})
	if err != nil || len(a.Schedules) != 0 {
		t.Fatalf("clear: %+v %v", a.Schedules, err)
	}
	// The Cron convenience on PATCH is one row in the daemon zone.
	one := "15 * * * *"
	a, err = s.UpdateAutomation(a.ID, AutomationPatch{Cron: &one})
	if err != nil || len(a.Schedules) != 1 || a.Schedules[0].Cron != one || a.Schedules[0].TZ != "" || !a.Schedules[0].Enabled {
		t.Fatalf("cron patch: %+v %v", a.Schedules, err)
	}
	// Deleting the automation takes its rows.
	if err := s.DeleteAutomation(a.ID); err != nil {
		t.Fatal(err)
	}
	if rows, _ := s.schedulesFor(s.db, a.ID); len(rows) != 0 {
		t.Fatalf("schedules must cascade: %+v", rows)
	}
}

func TestRunRecordsSchedule(t *testing.T) {
	s := openTest(t)
	a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
	r, err := s.CreateRun(RunParams{AutomationID: a.ID, ScheduleID: a.Schedules[0].ID, Trigger: TriggerSchedule, Status: RunRunning})
	if err != nil || r.ScheduleID == nil || *r.ScheduleID != a.Schedules[0].ID {
		t.Fatalf("run: %+v %v", r, err)
	}
	got, _ := s.GetRun(r.ID)
	if got.ScheduleID == nil || *got.ScheduleID != a.Schedules[0].ID {
		t.Fatalf("persisted: %+v", got)
	}
	m, _ := s.CreateRun(RunParams{AutomationID: a.ID, Trigger: TriggerManual, Status: RunSkipped, Reason: "busy"})
	if m.ScheduleID != nil {
		t.Fatalf("manual run must not name a schedule: %+v", m)
	}
}

// ADR-0217: a start run names its CLI (pi by default); only the CLIs the
// unattended runner reaches are accepted, and a patch can move it.
func TestAutomationCLI(t *testing.T) {
	s := openTest(t)
	a, _, err := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
	if err != nil || a.CLI != CLIPi {
		t.Fatalf("%+v %v", a, err)
	}
	b, _, err := s.CreateAutomation(AutomationParams{Name: "b", Action: AutomationStart, CLI: "claude-code", Prompt: "p", Cron: "0 9 * * *"})
	if err != nil || b.CLI != "claude-code" {
		t.Fatalf("%+v %v", b, err)
	}
	if got, _ := s.GetAutomation(b.ID); got.CLI != "claude-code" {
		t.Fatalf("read back %q", got.CLI)
	}
	if _, _, err := s.CreateAutomation(AutomationParams{Name: "c", Action: AutomationStart, CLI: "muse", Prompt: "p", Cron: "0 9 * * *"}); err == nil {
		t.Fatal("a CLI without a measured composer was accepted for start")
	}
	codex := "codex"
	if got, err := s.UpdateAutomation(b.ID, AutomationPatch{CLI: &codex}); err != nil || got.CLI != "codex" {
		t.Fatalf("%+v %v", got, err)
	}
}

// The editor's CLI list is the store's (web/shared/domain/automationsPi.js).
func TestStartCLIsMatchTheEditor(t *testing.T) {
	body, err := os.ReadFile("../../web/shared/domain/automationsPi.js")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`START_CLIS = \[([^\]]*)\]`).FindSubmatch(body)
	if m == nil {
		t.Fatal("START_CLIS not found")
	}
	var js []string
	for _, part := range strings.Split(string(m[1]), ",") {
		if id := strings.Trim(strings.TrimSpace(part), `"`); id != "" {
			js = append(js, id)
		}
	}
	if !reflect.DeepEqual(js, UnattendedCLIs) {
		t.Fatalf("js %v, go %v", js, UnattendedCLIs)
	}
}
