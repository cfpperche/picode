package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

func dockerHealthProjects(ctx context.Context, h Host) (View, error) {
	v := dockerView()
	monitors, err := h.Store.DockerMonitors()
	if err != nil {
		return v, err
	}
	targets := map[string][2]string{}
	for _, m := range monitors {
		targets[dockerTarget(m.Endpoint, m.Project)] = [2]string{m.Endpoint, m.Project}
	}
	inv, invErr := h.Docker.Inventory(ctx)
	if invErr == nil {
		for _, c := range inv.Containers {
			targets[dockerTarget(inv.Endpoint, c.Project)] = [2]string{inv.Endpoint, c.Project}
		}
	}
	if invErr != nil {
		v.Blocks = append(v.Blocks, dockerText("Connection unavailable", "Current Docker state could not be read. Saved project samples remain available."), dockerRetry())
	}
	items := []ListItem{}
	for target, pair := range targets {
		hv, err := h.Docker.Health(pair[0], pair[1])
		if err != nil {
			return v, err
		}
		badge := "Monitoring off"
		if hv.Monitor.Enabled {
			badge = "Monitoring on"
		}
		meta := []string{pair[0]}
		if hv.Snapshot == nil {
			meta = append(meta, "No sample yet")
		} else if hv.Stale {
			meta = append(meta, "Sample is stale")
		} else {
			meta = append(meta, "Recent sample")
		}
		open := 0
		for _, inc := range hv.Incidents {
			if inc.State == "open" {
				open++
			}
		}
		if open > 0 {
			meta = append(meta, fmt.Sprintf("%d open incidents", open))
		}
		tone := ""
		if open > 0 {
			tone = "warn"
		}
		items = append(items, ListItem{Wrap: true, ID: target, Title: dockerProjectName(pair[1]), Meta: meta, At: hv.Monitor.SampledAt, Badge: badge, Tone: tone, Path: "health/" + target})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Project health", Items: items, Empty: "No projects to check on this connection.", Actions: []Action{{ID: "refresh", Label: "Refresh projects"}}})
	return v, nil
}

func dockerProjectHealth(hv docker.HealthView) View {
	m := hv.Monitor
	v := dockerView()
	state := "Monitoring is off. Check health whenever you need a sample."
	if m.Enabled {
		state = fmt.Sprintf("Monitoring every %d seconds. Incidents open after %d bad samples and resolve after two good samples. Retention: %d days.", m.IntervalSeconds, m.BadSamples, m.RetentionDays)
	}
	head := dockerText(dockerProjectName(m.Project), state)
	head.Pane = "detail"
	head.At = m.SampledAt
	v.Blocks = append(v.Blocks, head)
	args := map[string]string{"endpoint": m.Endpoint, "project": m.Project}
	v.Blocks = append(v.Blocks, dockerActions(Action{ID: "check-health", Label: "Check health", Primary: hv.Snapshot == nil, Args: args}, Action{ID: "diagnose", Label: "Diagnose", Args: args}, Action{ID: "configure-monitor", Label: "Configure monitoring", Args: args}))
	if hv.Snapshot == nil {
		v.Blocks = append(v.Blocks, dockerText("No sample yet", "Choose Check health to inspect this project's services."))
		return v
	}
	if hv.Snapshot.Error != "" {
		v.Blocks = append(v.Blocks, dockerText("Sample unavailable", hv.Snapshot.Error))
	} else if hv.Stale {
		v.Blocks = append(v.Blocks, dockerText("Stale sample", "These observations are old. Check health for current state."))
	}
	items := []ListItem{}
	for _, hc := range hv.Snapshot.Containers {
		c := hc.Container
		badge := c.State
		meta := []string{fmt.Sprintf("%d restarts", c.RestartCount)}
		if hc.Error != "" {
			badge = "Unknown"
			meta = []string{hc.Error}
		} else {
			if !c.HasHealthCheck {
				meta = append(meta, "No health check")
			} else if c.State != "running" {
				meta = append(meta, "Last health check before stopping: "+c.Health)
			} else {
				meta = append(meta, "Health: "+c.Health)
			}
			if hc.Stats != nil {
				usage := fmt.Sprintf("CPU %.1f%% · Memory %.1f MiB", hc.Stats.CPUPercent, float64(hc.Stats.MemoryBytes)/(1024*1024))
				if hc.Stats.LimitBytes > 0 {
					usage += fmt.Sprintf(" / %.1f MiB limit", float64(hc.Stats.LimitBytes)/(1024*1024))
				} else {
					usage += " · Limit unavailable"
				}
				meta = append(meta, usage)
			} else if c.State != "running" {
				meta = append(meta, "Resource usage unavailable while stopped")
			} else {
				meta = append(meta, "Resource sample unavailable")
			}
		}
		tone := dockerTone(badge)
		if c.Health == "unhealthy" {
			tone = "warn"
		}
		items = append(items, ListItem{Wrap: true, ID: c.ID, Title: c.Name, Badge: badge, Tone: tone, Meta: meta, Path: "item/" + c.ID})
	}
	v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Service observations", Items: items, At: hv.Snapshot.SampledAt, Empty: "No containers were observed in this project."})
	incidents := []ListItem{}
	for _, inc := range hv.Incidents {
		incidents = append(incidents, ListItem{Wrap: true, ID: inc.ID, Title: inc.Title, Subtitle: inc.Detail, Badge: inc.State, Tone: dockerTone(inc.State), At: inc.UpdatedAt})
	}
	v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Incidents", Items: incidents, Empty: "No recorded incidents. Monitoring must be enabled to record incidents."})
	return v
}

func dockerMonitorView(m store.DockerMonitor) View {
	v := dockerView()
	text := "Choose the sampling cadence and incident thresholds. Monitoring only observes; it does not repair services."
	head := dockerText("Monitoring · "+dockerProjectName(m.Project), text)
	head.Pane = "detail"
	v.Blocks = append(v.Blocks, head)
	selectField := func(name, title, value string, options ...string) Field {
		return Field{Name: name, Method: "select", Title: title, Prefill: value, Options: options}
	}
	enabled := "Disabled"
	if m.Enabled {
		enabled = "Enabled"
	}
	fields := []Field{
		selectField("enabled", "Monitoring", enabled, "Disabled", "Enabled"),
		selectField("interval", "Seconds between samples", strconv.Itoa(m.IntervalSeconds), "30", "60", "300"),
		selectField("cpu", "CPU threshold (% of one core)", strconv.Itoa(m.CPUPercent), "80", "90", "200"),
		selectField("memory", "Memory threshold (% of reported limit)", strconv.Itoa(m.MemoryPercent), "80", "85", "95"),
		selectField("samples", "Consecutive bad samples to open an incident", strconv.Itoa(m.BadSamples), "2", "3", "5"),
		selectField("retention", "Days to retain closed incidents", strconv.Itoa(m.RetentionDays), "7", "30"),
	}
	v.Blocks = append(v.Blocks, Block{Type: "form", Form: &Form{ID: "save-monitor:" + strconv.Itoa(m.Revision), Submit: "Save monitoring", Fields: fields}})
	return v
}

func dockerDiagnosisView(d docker.Diagnosis) View {
	v := dockerView()
	head := dockerText("Diagnosis · "+dockerProjectName(d.Snapshot.Project), "Observations and possible causes from a current sample. Review any procedure before execution.")
	head.Pane = "detail"
	head.At = d.Snapshot.SampledAt
	v.Blocks = append(v.Blocks, head)
	args := map[string]string{"endpoint": d.Snapshot.Endpoint, "project": d.Snapshot.Project}
	if d.Snapshot.Error != "" {
		v.Blocks = append(v.Blocks, dockerText("Diagnosis unavailable", d.Snapshot.Error), dockerActions(Action{ID: "diagnose", Label: "Check again", Args: args}))
		return v
	}
	if len(d.Findings) == 0 {
		v.Blocks = append(v.Blocks, dockerText("No procedure suggested", "No supported maintenance condition was found in this sample."), dockerActions(Action{ID: "open-health", Label: "View project health", Args: args}))
		return v
	}
	for _, f := range d.Findings {
		v.Blocks = append(v.Blocks, dockerText(f.Name, "Observed: "+f.Observation+"\n\nPossible cause: "+f.Hypothesis))
		actions := []Action{{ID: "open-container", Label: "Inspect service", Args: map[string]string{"id": f.ContainerID}}}
		if f.Procedure != "" {
			actions = append(actions, Action{ID: "preview-procedure", Label: "Review procedure", Primary: true, Args: map[string]string{"procedure": f.Procedure, "containerId": f.ContainerID, "project": d.Snapshot.Project}})
		}
		v.Blocks = append(v.Blocks, dockerActions(actions...))
	}
	return v
}

func (a dockerApp) maintenanceAction(ctx context.Context, h Host, req ActionRequest) (ActionResult, bool, error) {
	args := req.Args
	target := dockerTarget(args["endpoint"], args["project"])
	path := ""
	switch req.Action {
	case "open-project":
		path = "project/" + target
	case "project-resources":
		path = "resources/project/" + target
	case "open-health":
		path = "health/" + target
	case "configure-monitor":
		path = "health/config/" + target
	case "open-container":
		path = "item/" + args["id"]
	case "open-job":
		path = "history/job/" + args["id"]
	case "open-containers":
		path = "containers"
	case "open-resources":
		path = "resources"
	case "check-health":
		_, err := h.Docker.CheckHealth(ctx, args["endpoint"], args["project"])
		return ActionResult{Path: "health/" + target}, true, err
	case "diagnose":
		_, err := h.Docker.CheckHealth(ctx, args["endpoint"], args["project"])
		return ActionResult{Path: "health/diagnosis/" + target}, true, err
	case "execute-plan":
		j, err := h.Docker.Execute(ctx, docker.ExecuteRequest{PlanID: args["id"], RequestKey: args["requestKey"], Approved: true, Actor: h.Actor})
		return ActionResult{Path: "history/job/" + j.ID}, true, err
	case "preview-start", "preview-stop", "preview-restart", "preview-remove", "preview-procedure", "repreview":
		p := docker.PlanRequest{Kind: "project", Project: args["project"], Action: strings.TrimPrefix(req.Action, "preview-"), Actor: h.Actor}
		if req.Action == "preview-remove" {
			p = docker.PlanRequest{Kind: "resource", ResourceKind: args["kind"], ResourceID: args["id"], Actor: h.Actor}
		}
		if req.Action == "preview-procedure" {
			p = docker.PlanRequest{Kind: "procedure", Project: args["project"], ContainerID: args["containerId"], Procedure: args["procedure"], Actor: h.Actor}
		}
		if req.Action == "repreview" {
			old, err := h.Store.DockerPlan(args["id"])
			if err != nil {
				return ActionResult{}, true, err
			}
			if err = json.Unmarshal(old.Input, &p); err != nil {
				return ActionResult{}, true, err
			}
			p.Actor = h.Actor
		}
		plan, err := h.Docker.Preview(ctx, p)
		return ActionResult{Path: "plan/" + plan.ID}, true, err
	default:
		if strings.HasPrefix(req.Action, "save-monitor:") {
			ep, project, err := readDockerTarget(strings.TrimPrefix(req.Path, "health/config/"))
			if err != nil {
				return ActionResult{}, true, err
			}
			parse := func(value string) int { n, _ := strconv.Atoi(value); return n }
			if args["enabled"] != "Enabled" && args["enabled"] != "Disabled" {
				return ActionResult{}, true, fmt.Errorf("Choose Enabled or Disabled")
			}
			m := store.DockerMonitor{Endpoint: ep, Project: project, Revision: parse(strings.TrimPrefix(req.Action, "save-monitor:")), Enabled: args["enabled"] == "Enabled", IntervalSeconds: parse(args["interval"]), CPUPercent: parse(args["cpu"]), MemoryPercent: parse(args["memory"]), BadSamples: parse(args["samples"]), RetentionDays: parse(args["retention"]), Actor: h.Actor}
			_, err = h.Docker.ConfigureMonitor(ctx, m)
			return ActionResult{Path: "health/" + dockerTarget(ep, project), Toast: "Monitoring settings saved"}, true, err
		}
		return ActionResult{}, false, nil
	}
	return ActionResult{Path: path}, true, nil
}
