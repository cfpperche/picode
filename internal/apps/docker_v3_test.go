package apps

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

func TestDockerHealthViewsDistinguishMissingEvidence(t *testing.T) {
	for _, state := range []string{"none", "running", "stopped", "stale", "unreachable", "unknown-limit"} {
		t.Run(state, func(t *testing.T) {
			m := store.DefaultDockerMonitor("unix:///tmp/qa", "demo")
			h := docker.HealthView{Monitor: m, Incidents: []store.DockerIncident{}}
			if state != "none" {
				h.Snapshot = &docker.HealthSnapshot{SampledAt: time.Now().UTC().Format(time.RFC3339), Containers: []docker.HealthContainer{{Container: docker.Container{ID: strings.Repeat("a", 64), Name: "qa", State: "running"}}}}
			}
			if state == "stopped" {
				h.Snapshot.Containers[0].Container.State = "exited"
			}
			if state == "stale" {
				h.Stale = true
			}
			if state == "unknown-limit" {
				h.Snapshot.Containers[0].Stats = &docker.Stats{CPUPercent: 1, MemoryBytes: 1048576}
			}
			if state == "unreachable" {
				h.Snapshot.Error = "Docker could not be sampled."
				h.Snapshot.Containers = nil
				h.Stale = true
			}
			v := dockerProjectHealth(h)
			if err := v.Validate(); err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(v)
			text := string(encoded)
			wanted := map[string]string{"none": "No sample yet", "running": "No health check", "stopped": "Resource usage unavailable while stopped", "stale": "Stale sample", "unreachable": "Sample unavailable", "unknown-limit": "Limit unavailable"}[state]
			if !strings.Contains(text, wanted) || !strings.Contains(text, "Check health") {
				t.Fatalf("missing %q: %s", wanted, text)
			}
			if strings.Contains(text, "CPU 0.0") || strings.Contains(text, `"badge":"healthy"`) {
				t.Fatal("invented health or zero metrics", text)
			}
		})
	}
}

func TestDockerPlansAndJobsAreReviewableWithoutDaemon(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p, err := s.CreateDockerPlan(store.DockerPlan{Kind: "project", Title: "Stop demo", Actor: "Requester", Endpoint: "unix:///tmp/qa", Project: "demo", Input: json.RawMessage(`{}`), Steps: []store.DockerStep{{Kind: "container", Target: strings.Repeat("a", 64), Name: "database", Action: "stop", State: "queued"}}})
	if err != nil {
		t.Fatal(err)
	}
	a := dockerApp{}
	h := Host{Store: s}
	v, err := a.View(context.Background(), h, "plan/"+p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = v.Validate(); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range v.Blocks {
		for _, it := range b.Items {
			if !it.Wrap {
				t.Fatal("reviewed ID may be truncated")
			}
		}
		for _, act := range b.Actions {
			if act.ID == "execute-plan" {
				found = act.Confirm != ""
			}
		}
	}
	if !found {
		t.Fatal("execution lacks confirmation")
	}
	j, _, err := s.BeginDockerJob(store.DockerJob{PlanID: p.ID, RequestKey: "qa-request", Title: p.Title, Actor: p.Actor, ApprovedBy: "Approver", Endpoint: p.Endpoint, Steps: p.Steps})
	if err != nil {
		t.Fatal(err)
	}
	j.State = "unknown"
	j.Message = "Interrupted; inspect before retrying."
	if err = s.UpdateDockerJob(j); err != nil {
		t.Fatal(err)
	}
	v, err = a.View(context.Background(), h, "history/job/"+j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = v.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(v)
	if !strings.Contains(string(raw), "Requester") || !strings.Contains(string(raw), "Approver") || !strings.Contains(string(raw), "unknown") {
		t.Fatal(string(raw))
	}
	v, err = a.View(context.Background(), h, "plan/"+p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range v.Blocks {
		for _, action := range b.Actions {
			if action.ID == "execute-plan" {
				t.Fatal("used plan offered execution again")
			}
		}
	}
}

func TestDockerTargetLinksPreserveExactProjectAndEndpoint(t *testing.T) {
	ep, project := "unix:///tmp/engine.sock", "preview/with spaces & unicode ✓"
	gotEP, gotProject, err := readDockerTarget(dockerTarget(ep, project))
	if err != nil || ep != gotEP || project != gotProject {
		t.Fatal(gotEP, gotProject, err)
	}
	if _, _, err = readDockerTarget("invalid"); err == nil {
		t.Fatal("invalid target accepted")
	}
}
