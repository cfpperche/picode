package docker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Mount struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Destination string `json:"destination"`
}

type networkAttachment struct{ NetworkID string }

func networkIDs(networks map[string]networkAttachment) []string {
	ids := []string{}
	for _, n := range networks {
		if n.NetworkID != "" {
			ids = append(ids, n.NetworkID)
		}
	}
	sort.Strings(ids)
	return ids
}

type Consumer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Project string `json:"project,omitempty"`
	State   string `json:"state"`
}

type Resource struct {
	Kind      string     `json:"kind"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Tags      []string   `json:"tags,omitempty"`
	Driver    string     `json:"driver,omitempty"`
	Scope     string     `json:"scope,omitempty"`
	SizeBytes *int64     `json:"sizeBytes"`
	Consumers []Consumer `json:"consumers"`
	Removable bool       `json:"removable"`
	Reason    string     `json:"reason,omitempty"`
}

type Resources struct {
	Endpoint  string     `json:"endpoint"`
	SampledAt string     `json:"sampledAt"`
	Items     []Resource `json:"items"`
}

var imageID = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func (s *Service) Resources(ctx context.Context) (Resources, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	c, err := s.client(ctx)
	if err != nil {
		return Resources{}, err
	}
	defer c.Close()
	return c.Resources(ctx)
}

func (c *Client) Resources(ctx context.Context) (Resources, error) {
	out := Resources{Endpoint: c.Endpoint, SampledAt: time.Now().UTC().Format(time.RFC3339), Items: []Resource{}}
	containers, err := c.Containers(ctx)
	if err != nil {
		return out, err
	}
	var images []struct {
		ID       string `json:"Id"`
		RepoTags []string
		Size     *int64
	}
	if err = c.get(ctx, "/v"+apiVersion+"/images/json?all=true", &images); err != nil {
		return out, err
	}
	var volumes struct {
		Volumes []struct{ Name, Driver, Scope string }
	}
	if err = c.get(ctx, "/v"+apiVersion+"/volumes", &volumes); err != nil {
		return out, err
	}
	var networks []struct {
		ID                  string `json:"Id"`
		Name, Driver, Scope string
		Ingress             bool
	}
	if err = c.get(ctx, "/v"+apiVersion+"/networks", &networks); err != nil {
		return out, err
	}
	for _, im := range images {
		name := im.ID
		tags := []string{}
		for _, tag := range im.RepoTags {
			if tag != "<none>:<none>" {
				tags = append(tags, tag)
			}
		}
		sort.Strings(tags)
		if len(tags) > 0 {
			name = tags[0]
		}
		if im.Size != nil && *im.Size < 0 {
			im.Size = nil
		}
		r := Resource{Kind: "image", ID: im.ID, Name: name, Tags: tags, SizeBytes: im.Size, Removable: imageID.MatchString(im.ID)}
		out.Items = append(out.Items, r)
	}
	for _, v := range volumes.Volumes {
		out.Items = append(out.Items, Resource{Kind: "volume", ID: v.Name, Name: v.Name, Driver: v.Driver, Scope: v.Scope, Reason: "Volumes are available for inspection only."})
	}
	for _, n := range networks {
		removable := n.Name != "bridge" && n.Name != "host" && n.Name != "none" && !n.Ingress && n.Scope == "local" && fullID.MatchString(n.ID)
		r := Resource{Kind: "network", ID: n.ID, Name: n.Name, Driver: n.Driver, Scope: n.Scope, Removable: removable}
		if !removable {
			r.Reason = "Built-in, ingress and non-local networks are protected."
		}
		out.Items = append(out.Items, r)
	}
	for i := range out.Items {
		r := &out.Items[i]
		r.Consumers = []Consumer{}
		for _, ct := range containers {
			used := r.Kind == "image" && ct.ImageID == r.ID
			for _, m := range ct.Mounts {
				used = used || (r.Kind == "volume" && m.Type == "volume" && m.Name == r.ID)
			}
			for _, id := range ct.Networks {
				used = used || (r.Kind == "network" && id == r.ID)
			}
			if used {
				r.Consumers = append(r.Consumers, Consumer{ID: ct.ID, Name: ct.Name, Project: ct.Project, State: ct.State})
			}
		}
		sort.Slice(r.Consumers, func(i, j int) bool { return r.Consumers[i].ID < r.Consumers[j].ID })
		if len(r.Consumers) > 0 {
			r.Removable = false
			r.Reason = "Referenced by containers, including any stopped consumers listed below."
		}
		if !r.Removable && r.Reason == "" {
			r.Reason = "Docker did not report a valid resource ID."
		}
	}
	sort.Slice(out.Items, func(i, j int) bool {
		if out.Items[i].Kind != out.Items[j].Kind {
			return out.Items[i].Kind < out.Items[j].Kind
		}
		if out.Items[i].Name != out.Items[j].Name {
			return out.Items[i].Name < out.Items[j].Name
		}
		return out.Items[i].ID < out.Items[j].ID
	})
	return out, nil
}

func findResource(inv Resources, kind, id string) (Resource, error) {
	for _, r := range inv.Items {
		if r.Kind == kind && r.ID == id {
			return r, nil
		}
	}
	return Resource{}, errors.New("Resource is no longer present. Refresh the inventory")
}

func resourcePath(kind, id string) (string, error) {
	switch {
	case kind == "image" && imageID.MatchString(id):
		return "/v" + apiVersion + "/images/" + url.PathEscape(id), nil
	case kind == "network" && fullID.MatchString(id):
		return "/v" + apiVersion + "/networks/" + id, nil
	default:
		return "", errors.New("Select an image or a custom local network using its full ID")
	}
}

func (c *Client) RemoveResource(ctx context.Context, kind, id string) error {
	path, err := resourcePath(kind, id)
	if err != nil {
		return err
	}
	if kind == "image" {
		path += "?force=false&noprune=true"
	}
	res, err := c.request(ctx, http.MethodDelete, path)
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	path, _ = resourcePath(kind, id)
	if kind == "image" {
		path += "/json"
	}
	res, err = c.request(ctx, http.MethodGet, path)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
		return nil
	}
	if err == nil {
		_ = res.Body.Close()
	}
	return fmt.Errorf("Docker accepted removal, but absence could not be verified. Refresh the inventory")
}

var secretName = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|private[_-]?key|credential)`)
var labeledSecret = regexp.MustCompile(`(?i)([a-z0-9_.-]*(?:password|passwd|secret|token|api[_-]?key|private[_-]?key|credential)[a-z0-9_.-]*["']?\s*[:=]\s*)(?:"[^"\n]*"|'[^'\n]*'|[^\s,;]+)`)
var bearerSecret = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/-]+=*`)
var urlSecret = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^\s/@]+:[^\s/@]+@`)

func sensitiveValues(env []string) []string {
	values := []string{}
	for _, entry := range env {
		name, value, ok := strings.Cut(entry, "=")
		if ok && len(value) >= 4 && (secretName.MatchString(name) || urlSecret.MatchString(value)) {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	return values
}

// Redact is best-effort masking, not interpretation of application output.
func Redact(text string, secrets []string) string {
	for _, secret := range secrets {
		if len(secret) >= 4 {
			text = strings.ReplaceAll(text, secret, "[redacted]")
		}
	}
	text = labeledSecret.ReplaceAllString(text, "${1}[redacted]")
	text = bearerSecret.ReplaceAllString(text, "Bearer [redacted]")
	return urlSecret.ReplaceAllString(text, "${1}[redacted]@")
}
