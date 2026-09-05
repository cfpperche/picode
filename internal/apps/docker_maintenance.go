package apps

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/store"
)

func dockerTarget(endpoint, project string) string {
	raw, _ := json.Marshal([]string{endpoint, project})
	return base64.RawURLEncoding.EncodeToString(raw)
}
func readDockerTarget(raw string) (string, string, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	var pair []string
	if err != nil || json.Unmarshal(data, &pair) != nil || len(pair) != 2 {
		return "", "", errors.New("Project link is invalid. Open the project again")
	}
	return pair[0], pair[1], nil
}
func dockerProjectName(name string) string {
	if name == "" {
		return "Standalone containers"
	}
	return name
}
func dockerActions(actions ...Action) Block { return Block{Type: "actions", Actions: actions} }
func dockerUnavailable(title string, err error) View {
	v := dockerView()
	v.Blocks = []Block{dockerText(title, err.Error()), dockerRetry()}
	return v
}

func (a dockerApp) maintenanceView(ctx context.Context, h Host, path string) (View, bool, error) {
	if path == "history" || strings.HasPrefix(path, "history/job/") {
		v, err := dockerHistory(h, path)
		return v, true, err
	}
	if strings.HasPrefix(path, "plan/") {
		v, err := dockerPlanView(h, strings.TrimPrefix(path, "plan/"))
		return v, true, err
	}
	if path != "resources" && !strings.HasPrefix(path, "resources/") && path != "health" && !strings.HasPrefix(path, "health/") && !strings.HasPrefix(path, "project/") {
		return View{}, false, nil
	}
	if h.Docker == nil {
		return dockerUnavailable("Docker unavailable", errors.New("Docker integration is unavailable.")), true, nil
	}
	var v View
	var err error
	switch {
	case path == "resources" || strings.HasPrefix(path, "resources/"):
		v, err = dockerResourceView(ctx, h, path)
	case path == "health":
		v, err = dockerHealthProjects(ctx, h)
	case strings.HasPrefix(path, "health/"):
		config := strings.HasPrefix(path, "health/config/")
		diagnosis := strings.HasPrefix(path, "health/diagnosis/")
		target := strings.TrimPrefix(path, "health/")
		if config {
			target = strings.TrimPrefix(path, "health/config/")
		}
		if diagnosis {
			target = strings.TrimPrefix(path, "health/diagnosis/")
		}
		ep, project, e := readDockerTarget(target)
		if e != nil {
			err = e
			break
		}
		health, e := h.Docker.Health(ep, project)
		if e != nil {
			err = e
			break
		}
		if config {
			v = dockerMonitorView(health.Monitor)
		} else if diagnosis {
			v = dockerDiagnosisView(docker.DiagnoseSnapshot(health))
		} else {
			v = dockerProjectHealth(health)
		}
	case strings.HasPrefix(path, "project/"):
		v, err = dockerProjectView(ctx, h, strings.TrimPrefix(path, "project/"))
	}
	if err != nil {
		return dockerUnavailable("View unavailable", err), true, nil
	}
	return v, true, nil
}

func dockerProjectView(ctx context.Context, h Host, target string) (View, error) {
	ep, project, err := readDockerTarget(target)
	if err != nil {
		return View{}, err
	}
	inv, err := h.Docker.Inventory(ctx)
	if err != nil {
		return View{}, err
	}
	if inv.Endpoint != ep {
		return View{}, errors.New("Docker connection changed. Open the project from Containers")
	}
	v := dockerView()
	head := dockerText(dockerProjectName(project), "Review an action for the existing containers in this project.")
	head.Pane = "detail"
	v.Blocks = append(v.Blocks, head)
	args := map[string]string{"project": project, "endpoint": ep}
	v.Blocks = append(v.Blocks, dockerActions(Action{ID: "preview-start", Label: "Review start", Args: args}, Action{ID: "preview-stop", Label: "Review stop", Args: args}, Action{ID: "preview-restart", Label: "Review restart", Args: args}, Action{ID: "open-health", Label: "Project health", Args: args}, Action{ID: "project-resources", Label: "Project resources", Args: args}))
	items := []ListItem{}
	for _, ct := range inv.Containers {
		if ct.Project == project {
			items = append(items, ListItem{Wrap: true, ID: ct.ID, Title: ct.Name, Subtitle: ct.Image, Badge: ct.State, Tone: dockerTone(ct.State), Path: "item/" + ct.ID})
		}
	}
	v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Services", Items: items, Empty: "No containers remain in this project.", At: inv.SampledAt})
	return v, nil
}

func dockerResourceView(ctx context.Context, h Host, path string) (View, error) {
	inv, err := h.Docker.Resources(ctx)
	if err != nil {
		return View{}, err
	}
	v := dockerView()
	projectResources := strings.HasPrefix(path, "resources/project/")
	project := ""
	if projectResources {
		ep, p, e := readDockerTarget(strings.TrimPrefix(path, "resources/project/"))
		if e != nil {
			return v, e
		}
		if ep != inv.Endpoint {
			return v, errors.New("Docker connection changed. Open Resources again")
		}
		project = p
		filtered := []docker.Resource{}
		for _, resource := range inv.Items {
			for _, consumer := range resource.Consumers {
				if consumer.Project == project {
					filtered = append(filtered, resource)
					break
				}
			}
		}
		inv.Items = filtered
		v.Blocks = append(v.Blocks, dockerText("Resources · "+dockerProjectName(project), "Resources currently referenced by this project's containers, including stopped services."))
	}
	if path == "resources" || projectResources {
		if len(inv.Items) == 0 {
			v.Blocks = []Block{{Type: "list", Title: "Resources", Items: []ListItem{}, Empty: "No images, volumes or networks on this connection.", Actions: []Action{{ID: "refresh", Label: "Refresh resources"}}}}
			return v, nil
		}
		for _, kind := range []string{"image", "volume", "network"} {
			items := []ListItem{}
			for _, r := range inv.Items {
				if r.Kind != kind {
					continue
				}
				badge := "In use"
				if len(r.Consumers) == 0 {
					badge = "Unreferenced"
				}
				meta := []string{docker.CountLabel(len(r.Consumers), "consumer")}
				if r.SizeBytes != nil {
					meta = append(meta, fmt.Sprintf("%.1f MiB reported", float64(*r.SizeBytes)/(1024*1024)))
				} else {
					meta = append(meta, "Size unavailable")
				}
				items = append(items, ListItem{Wrap: true, ID: r.ID, Title: r.Name, Meta: meta, Badge: badge, Path: "resources/" + kind + "/" + base64.RawURLEncoding.EncodeToString([]byte(r.ID))})
			}
			v.Blocks = append(v.Blocks, Block{Type: "list", ID: "docker-resources:" + inv.Endpoint + ":" + kind + ":" + path, Title: strings.ToUpper(kind[:1]) + kind[1:] + "s", Collapsible: true, Items: items, Empty: "No " + kind + "s in this view.", Actions: []Action{{ID: "refresh", Label: "Refresh " + kind + "s"}}})
		}
		return v, nil
	}
	parts := strings.Split(strings.TrimPrefix(path, "resources/"), "/")
	if len(parts) != 2 {
		return v, errors.New("Resource link is invalid")
	}
	id, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return v, errors.New("Resource link is invalid")
	}
	for _, r := range inv.Items {
		if r.Kind != parts[0] || r.ID != string(id) {
			continue
		}
		head := dockerText(r.Name, r.ID)
		head.Pane = "detail"
		head.Meta = []string{r.Kind}
		head.At = inv.SampledAt
		if r.Driver != "" {
			head.Meta = append(head.Meta, r.Driver)
		}
		v.Blocks = append(v.Blocks, head)
		if r.SizeBytes != nil {
			v.Blocks = append(v.Blocks, dockerText("Reported size", fmt.Sprintf("%.1f MiB. Shared layers mean this is not an estimate of reclaimable space.", float64(*r.SizeBytes)/(1024*1024))))
		} else {
			v.Blocks = append(v.Blocks, dockerText("Size", "Size is not reported by this inventory."))
		}
		if len(r.Tags) > 1 {
			v.Blocks = append(v.Blocks, dockerText("Image tags", strings.Join(r.Tags, "\n")))
		}
		if r.Removable {
			v.Blocks = append(v.Blocks, dockerActions(Action{ID: "preview-remove", Label: "Review removal", Args: map[string]string{"kind": r.Kind, "id": r.ID}}))
		} else {
			v.Blocks = append(v.Blocks, dockerText("Removal unavailable", r.Reason), dockerRetry())
		}
		items := []ListItem{}
		for _, ct := range r.Consumers {
			items = append(items, ListItem{Wrap: true, ID: ct.ID, Title: ct.Name, Badge: ct.State, Tone: dockerTone(ct.State), Meta: []string{dockerProjectName(ct.Project)}, Path: "item/" + ct.ID})
		}
		v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Consumers", Items: items, Empty: "No containers currently reference this resource."})
		if len(r.Consumers) > 0 {
			seen := map[string]bool{}
			projects := []ListItem{}
			for _, ct := range r.Consumers {
				if !seen[ct.Project] {
					seen[ct.Project] = true
					target := dockerTarget(inv.Endpoint, ct.Project)
					projects = append(projects, ListItem{ID: target, Title: dockerProjectName(ct.Project), Wrap: true, Path: "project/" + target})
				}
			}
			sort.Slice(projects, func(i, j int) bool { return projects[i].Title < projects[j].Title })
			v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Projects", Items: projects, Empty: "No project references."})
		}
		return v, nil
	}
	return v, errors.New("Resource is no longer present. Open Resources again")
}

func dockerPlanView(h Host, id string) (View, error) {
	p, err := h.Store.DockerPlan(id)
	if err != nil {
		return View{}, err
	}
	v := dockerView()
	head := dockerText(p.Title, p.Impact)
	head.Pane = "detail"
	head.Meta = []string{"Requested by " + p.Actor}
	head.At = p.CreatedAt
	v.Blocks = append(v.Blocks, head, dockerText("Connection", p.Endpoint))
	items := dockerStepItems(p.Steps)
	v.Blocks = append(v.Blocks, Block{Type: "list", Title: "Reviewed steps", Items: items, Empty: "No steps in this plan."})
	if j, e := h.Store.DockerJobForPlan(p.ID); e == nil {
		v.Blocks = append(v.Blocks, dockerActions(Action{ID: "open-job", Label: "View recorded result", Args: map[string]string{"id": j.ID}}))
		return v, nil
	} else if !errors.Is(e, sql.ErrNoRows) {
		return v, e
	}
	expires, _ := time.Parse(time.RFC3339Nano, p.ExpiresAt)
	if !time.Now().Before(expires) {
		v.Blocks = append(v.Blocks, dockerText("Preview expired", "Create a fresh preview to check the current targets."), dockerActions(Action{ID: "repreview", Label: "Create fresh preview", Primary: true, Args: map[string]string{"id": p.ID}}))
	} else {
		b := dockerText("Review window", "This preview expires at "+expires.UTC().Format("15:04:05 UTC")+". Targets are checked again before execution.")
		v.Blocks = append(v.Blocks, b)
		v.Blocks = append(v.Blocks, dockerActions(Action{ID: "execute-plan", Label: "Confirm and execute", Primary: true, Danger: p.Kind == "resource", Confirm: p.Title + "? " + p.Impact, Args: map[string]string{"id": p.ID}}, Action{ID: "repreview", Label: "Create fresh preview", Args: map[string]string{"id": p.ID}}))
	}
	return v, nil
}

func dockerStepItems(steps []store.DockerStep) []ListItem {
	items := []ListItem{}
	for i, step := range steps {
		path := ""
		if step.Kind == "container" || step.Kind == "verify" {
			path = "item/" + step.Target
		}
		items = append(items, ListItem{Wrap: true, ID: strconv.Itoa(i), Title: fmt.Sprintf("%d. %s · %s", i+1, actionLabel(step.Action), step.Name), Subtitle: step.Message, Meta: []string{step.Target}, Badge: step.State, Tone: dockerTone(step.State), Busy: step.State == "running", Path: path})
	}
	return items
}

func dockerTone(state string) string {
	switch state {
	case "succeeded", "healthy", "running":
		return "ok"
	case "queued":
		return "info"
	case "unknown", "partial", "unhealthy", "restarting", "open":
		return "warn"
	case "failed":
		return "danger"
	}
	return ""
}

func dockerHistory(h Host, path string) (View, error) {
	v := dockerView()
	if strings.HasPrefix(path, "history/job/") {
		j, err := h.Store.DockerJob(strings.TrimPrefix(path, "history/job/"))
		if err != nil {
			return v, err
		}
		head := dockerText(j.Title, j.Message)
		head.Pane = "detail"
		head.Busy = j.State == "running"
		head.At = j.CreatedAt
		head.Meta = []string{j.State, "Requested by " + j.Actor, "Approved by " + j.ApprovedBy}
		next := Action{ID: "open-health", Label: "Check project health", Args: map[string]string{"endpoint": j.Endpoint, "project": j.Project}}
		if j.Kind == "resource" {
			next = Action{ID: "open-resources", Label: "View resources"}
		}
		v.Blocks = []Block{head, {Type: "list", Title: "Step results", Items: dockerStepItems(j.Steps), Empty: "No recorded steps."}, dockerActions(next)}
		return v, nil
	}
	ops, err := h.Store.DockerOperations(50)
	if err != nil {
		return v, err
	}
	jobs, err := h.Store.DockerJobs()
	if err != nil {
		return v, err
	}
	items := []ListItem{}
	for _, j := range jobs {
		items = append(items, ListItem{Wrap: true, ID: j.ID, Title: j.Title, Subtitle: j.Message, Badge: j.State, Tone: dockerTone(j.State), Busy: j.State == "running", At: j.CreatedAt, Meta: []string{j.ApprovedBy}, Path: "history/job/" + j.ID})
	}
	for _, op := range ops {
		items = append(items, ListItem{Wrap: true, ID: op.ID, Title: actionLabel(op.Action) + " · " + op.ContainerName, Subtitle: op.Message, Badge: op.State, Tone: dockerTone(op.State), Busy: op.State == "running", At: op.CreatedAt, Meta: []string{op.Actor}, Path: "item/" + op.ContainerID})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].At > items[j].At })
	v.Blocks = []Block{{Type: "list", Title: "Recent operations", Items: items, Empty: "No Docker operations yet.", Actions: []Action{{ID: "open-containers", Label: "Open containers"}}}}
	return v, nil
}
