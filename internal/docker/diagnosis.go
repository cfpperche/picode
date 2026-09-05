package docker

import (
	"context"
	"fmt"
)

type Finding struct {
	ContainerID string `json:"containerId"`
	Name        string `json:"name"`
	Observation string `json:"observation"`
	Hypothesis  string `json:"hypothesis"`
	Procedure   string `json:"procedure,omitempty"`
}

type Diagnosis struct {
	Snapshot HealthSnapshot `json:"snapshot"`
	Findings []Finding      `json:"findings"`
}

func (s *Service) Diagnose(ctx context.Context, endpoint, project string) (Diagnosis, error) {
	v, err := s.CheckHealth(ctx, endpoint, project)
	if err != nil {
		return Diagnosis{}, err
	}
	return DiagnoseSnapshot(v), nil
}

func DiagnoseSnapshot(v HealthView) Diagnosis {
	if v.Snapshot == nil {
		return Diagnosis{Snapshot: HealthSnapshot{Endpoint: v.Monitor.Endpoint, Project: v.Monitor.Project, Error: "No health sample yet. Check again."}, Findings: []Finding{}}
	}
	d := Diagnosis{Snapshot: *v.Snapshot, Findings: []Finding{}}
	if d.Snapshot.Error != "" {
		return d
	}
	m := v.Monitor
	for _, h := range d.Snapshot.Containers {
		c := h.Container
		if h.Error != "" {
			continue
		}
		add := func(observation, hypothesis, procedure string) {
			d.Findings = append(d.Findings, Finding{ContainerID: c.ID, Name: c.Name, Observation: observation, Hypothesis: hypothesis, Procedure: procedure})
		}
		if c.State == "restarting" {
			add(fmt.Sprintf("Docker reports restarting; %d recorded restarts.", c.RestartCount), "The process may be failing repeatedly. Inspect recent logs and configuration before starting it again.", "stop-restart-loop")
		}
		if c.State == "running" && c.Health == "unhealthy" {
			add("The configured health check is failing.", "The service or one of its dependencies may be unavailable. A restart may only mitigate the symptom.", "restart-unhealthy")
		}
		if ValidAction("start", c.State) {
			add(fmt.Sprintf("Container is %s; exit code %d.", c.State, c.ExitCode), "This may be intentional or an unavailable dependency. No dependency relationship has been established.", "start-stopped-service")
		}
		if h.Stats != nil && h.Stats.LimitBytes > 0 {
			percent := float64(h.Stats.MemoryBytes) / float64(h.Stats.LimitBytes) * 100
			if percent >= float64(m.MemoryPercent) {
				add(fmt.Sprintf("Memory is %.1f%% of the reported limit.", percent), "Load, caching or a leak may explain pressure. A restart is temporary mitigation and interrupts connections.", "restart-high-memory")
			}
			if h.Stats.CPUPercent >= float64(m.CPUPercent) {
				add(fmt.Sprintf("CPU is %.1f%% of one core.", h.Stats.CPUPercent), "Load or a busy process may explain the sample. Inspect recent logs and repeat the sample before choosing an action.", "")
			}
		}
		if c.OOMKilled {
			add("Docker recorded an out-of-memory termination.", "The process exceeded available memory. Review its workload and memory limit; restarting does not change that limit.", "")
		}
	}
	return d
}
