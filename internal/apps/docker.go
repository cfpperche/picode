package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

type dockerApp struct{}

func (dockerApp) Manifest() Manifest {
	return Manifest{ID: "docker", Name: "Docker", Icon: "box", APIVersion: APIVersion}
}
func (dockerApp) Badge(ctx context.Context, h Host) (Badge, error) {
	if h.Store == nil {
		return Badge{}, nil
	}
	ops, err := h.Store.DockerOperations(50)
	if err != nil {
		return Badge{}, err
	}
	for _, op := range ops {
		if op.State == "running" {
			return Badge{Dot: true}, nil
		}
	}
	return Badge{}, nil
}

func dockerTabs() []Tab {
	return []Tab{{ID: "containers", Label: "Containers", Path: ""}, {ID: "history", Label: "History", Path: "history"}}
}
func dockerView() View {
	return View{APIVersion: APIVersion, Title: "Docker", Tabs: dockerTabs(), Blocks: []Block{}}
}
func dockerText(title, text string) Block { return Block{Type: "detail", Title: title, Text: &text} }
func dockerRetry() Block {
	return Block{Type: "actions", Actions: []Action{{ID: "refresh", Label: "Check again"}}}
}

func (a dockerApp) View(ctx context.Context, h Host, path string) (View, error) {
	v := dockerView()
	if h.Store == nil {
		return v, fmt.Errorf("Docker history is unavailable")
	}
	ops, err := h.Store.DockerOperations(50)
	if err != nil {
		return v, err
	}
	if path == "history" {
		items := []ListItem{}
		for _, op := range ops {
			tone := "ok"
			if op.State == "running" {
				tone = "info"
			} else if op.State == "unknown" {
				tone = "warn"
			} else if op.State == "failed" {
				tone = "danger"
			}
			items = append(items, ListItem{ID: op.ID, Title: actionLabel(op.Action) + " · " + op.ContainerName, Subtitle: op.Message, Meta: []string{op.Actor}, At: op.CreatedAt, Badge: op.State, Tone: tone, Busy: op.State == "running", Path: "item/" + op.ContainerID})
		}
		v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Recent operations", Empty: "No Docker operations yet.", Items: items})
		return v, nil
	}
	if h.Docker == nil {
		v.Blocks = []Block{dockerText("", "Docker integration is unavailable."), dockerRetry()}
		return v, nil
	}
	if strings.HasPrefix(path, "item/") {
		id := strings.TrimPrefix(path, "item/")
		d, err := h.Docker.Detail(ctx, id)
		if err != nil {
			v.Blocks = []Block{dockerText("Container unavailable", err.Error()), dockerRetry()}
			return v, nil
		}
		c := d.Container
		meta := []string{c.State, c.Image}
		if c.Project != "" {
			meta = append(meta, c.Project)
		}
		if c.Health != "" {
			meta = append(meta, "Health: "+c.Health)
		}
		v.Blocks = append(v.Blocks, Block{Type: "detail", Pane: "detail", Title: c.Name, Meta: meta, Text: &c.ID})
		var active *store.DockerOperation
		for i := range ops {
			if ops[i].ContainerID == id && ops[i].Endpoint == d.Endpoint && ops[i].State == "running" {
				active = &ops[i]
				break
			}
		}
		if active != nil {
			b := dockerText("Operation", actionLabel(active.Action)+" in progress…")
			b.Busy = true
			v.Blocks = append(v.Blocks, b)
		} else {
			for _, op := range ops {
				if op.ContainerID == id && op.Endpoint == d.Endpoint {
					b := dockerText("Last operation", actionLabel(op.Action)+": "+op.State+". "+op.Message)
					b.At = op.FinishedAt
					v.Blocks = append(v.Blocks, b)
					break
				}
			}
			actions := []Action{}
			for _, verb := range []string{"start", "stop", "restart"} {
				if !docker.ValidAction(verb, c.State) {
					continue
				}
				act := Action{ID: verb, Label: actionLabel(verb), Primary: verb == "start", Args: map[string]string{"containerId": id}}
				if verb != "start" {
					act.Confirm = actionLabel(verb) + " " + c.Name + "? Connections to this container may be interrupted."
					act.Danger = verb == "stop"
				}
				actions = append(actions, act)
			}
			if len(actions) > 0 {
				v.Blocks = append(v.Blocks, Block{Type: "actions", Actions: actions})
			}
		}
		if d.Stats != nil {
			v.Blocks = append(v.Blocks, dockerText("Resource sample", fmt.Sprintf("CPU %.1f%% · Memory %.1f / %.1f MiB", d.Stats.CPUPercent, float64(d.Stats.MemoryBytes)/(1024*1024), float64(d.Stats.LimitBytes)/(1024*1024))))
			v.Blocks[len(v.Blocks)-1].At = d.Stats.SampledAt
		} else if d.StatsError != "" {
			v.Blocks = append(v.Blocks, dockerText("Resource sample", "Resource usage could not be read. Refresh to try again."))
		}
		if d.Logs != nil {
			content := d.Logs.Text
			if strings.TrimSpace(content) == "" {
				content = "No recent logs."
			}
			b := dockerText("Recent logs", content)
			b.At = d.Logs.SampledAt
			if d.Logs.Truncated {
				b.Meta = []string{"Output limited to 64 KiB"}
			}
			v.Blocks = append(v.Blocks, b)
		} else {
			v.Blocks = append(v.Blocks, dockerText("Recent logs", "Logs are unavailable: "+d.LogsError))
		}
		return v, nil
	}
	if path != "" {
		return v, fmt.Errorf("Unknown Docker view")
	}
	inv, err := h.Docker.Inventory(ctx)
	if err != nil {
		v.Blocks = []Block{dockerText("Docker unavailable", err.Error()), {Type: "detail", Markdown: "[Set up Docker](https://cfpperche.github.io/picode/guide/docker)"}, dockerRetry()}
		return v, nil
	}
	v.Blocks = dockerContainerGroups(inv, ops)
	if len(inv.Containers) == 0 {
		v.Blocks = []Block{{Type: "list", Title: "Containers", At: inv.SampledAt, Empty: "No containers on this Docker connection.", Items: []ListItem{}}, {Type: "detail", Markdown: "[Set up an application](https://cfpperche.github.io/picode/guide/docker)"}}
	}
	return v, nil
}

func dockerContainerGroups(inv docker.Inventory, ops []store.DockerOperation) []Block {
	type group struct {
		block  Block
		states map[string]int
	}
	groups := map[string]*group{}
	for _, c := range inv.Containers {
		g := groups[c.Project]
		if g == nil {
			title := c.Project
			if title == "" {
				title = "Standalone containers"
			}
			// The complete endpoint/project pair survives sorting, filtering,
			// refreshes and connection changes without sharing fold preferences.
			key, _ := json.Marshal([]string{inv.Endpoint, c.Project})
			g = &group{block: Block{Type: "list", ID: "docker-group:" + string(key), Title: title, Collapsible: true, Items: []ListItem{}}, states: map[string]int{}}
			groups[c.Project] = g
		}
		state := c.State
		if state == "created" || state == "exited" {
			state = "stopped"
		} else if state == "" {
			state = "unknown"
		}
		g.states[state]++
		tone := ""
		if c.State == "running" {
			tone = "ok"
		}
		badge := c.State
		busy := false
		for _, op := range ops {
			if op.ContainerID == c.ID && op.Endpoint == inv.Endpoint && op.State == "running" {
				busy = true
				badge = actionLabel(op.Action) + " in progress"
				tone = "info"
				break
			}
		}
		meta := []string{}
		if c.Service != "" {
			meta = append(meta, c.Service)
		}
		g.block.Busy = g.block.Busy || busy
		g.block.Items = append(g.block.Items, ListItem{ID: c.ID, Title: c.Name, Subtitle: c.Image, Meta: meta, Badge: badge, Tone: tone, Busy: busy, Path: "item/" + c.ID})
	}
	projects := make([]string, 0, len(groups))
	for project := range groups {
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		if (projects[i] == "") != (projects[j] == "") {
			return projects[j] == ""
		}
		return projects[i] < projects[j]
	})
	blocks := make([]Block, 0, len(projects))
	for _, project := range projects {
		g := groups[project]
		states := make([]string, 0, len(g.states))
		for state := range g.states {
			states = append(states, state)
		}
		sort.Slice(states, func(i, j int) bool {
			if (states[i] == "running") != (states[j] == "running") {
				return states[i] == "running"
			}
			return states[i] < states[j]
		})
		for _, state := range states {
			g.block.Meta = append(g.block.Meta, fmt.Sprintf("%d %s", g.states[state], state))
		}
		sort.Slice(g.block.Items, func(i, j int) bool { return g.block.Items[i].Title < g.block.Items[j].Title })
		blocks = append(blocks, g.block)
	}
	return blocks
}

func actionLabel(action string) string {
	switch action {
	case "start":
		return "Start"
	case "stop":
		return "Stop"
	case "restart":
		return "Restart"
	}
	return action
}

func (a dockerApp) Action(ctx context.Context, h Host, req ActionRequest) (ActionResult, error) {
	if req.Action == "refresh" {
		v, err := a.View(ctx, h, req.Path)
		return ActionResult{View: &v}, err
	}
	if h.Docker == nil {
		return ActionResult{}, fmt.Errorf("Docker integration is unavailable")
	}
	op, err := h.Docker.Start(ctx, docker.Request{Action: req.Action, ContainerID: req.Args["containerId"], RequestKey: req.Args["requestKey"], Actor: h.Actor})
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Path: "item/" + op.ContainerID}, nil
}
