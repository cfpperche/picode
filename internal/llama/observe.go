package llama

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

type FileProgress struct {
	File  string `json:"file"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
}

func parseProgress(raw json.RawMessage) []FileProgress {
	var files map[string]struct {
		Done  *int64 `json:"done"`
		Total *int64 `json:"total"`
	}
	if json.Unmarshal(raw, &files) != nil || len(files) > 64 {
		return nil
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	out := []FileProgress{}
	for _, name := range names {
		f := files[name]
		if f.Done == nil || *f.Done < 0 || (f.Total != nil && *f.Total < 0) {
			continue
		}
		u, err := url.Parse(name)
		if err != nil {
			continue
		}
		label := path.Base(u.Path)
		if len(label) > 256 {
			label = label[:256]
		}
		if label == "." || label == "/" || label == "" {
			label = "Model file"
		}
		done, total := *f.Done, int64(0)
		if f.Total != nil {
			total = *f.Total
		}
		if total > 0 && done > total {
			done = total
		}
		out = append(out, FileProgress{label, done, total})
	}
	return out
}

type Capabilities struct {
	Build          string `json:"build"`
	Router         bool   `json:"router"`
	Events         bool   `json:"events"`
	CancelDownload bool   `json:"cancelDownload"`
}

// Capabilities never tests support by sending a mutation. Cancellation is
// enabled only for the build family exercised by the real acceptance fixture.
func (c *Client) Capabilities(ctx context.Context) Capabilities {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var props struct {
		Role  string `json:"role"`
		Build string `json:"build_info"`
	}
	raw, err := c.doContext(ctx, http.MethodGet, "/props", nil)
	if err != nil || json.Unmarshal(raw, &props) != nil {
		return Capabilities{}
	}
	caps := Capabilities{Build: props.Build, Router: props.Role == "router"}
	if len(caps.Build) > 128 {
		caps.Build = ""
	}
	if !caps.Router {
		return caps
	}
	response, err := c.events(ctx)
	if err == nil {
		caps.Events = true
		response.Body.Close()
	}
	caps.CancelDownload = caps.Events && strings.HasPrefix(caps.Build, "b10809-")
	return caps
}

func (c *Client) events(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/models/sse", nil)
	if err != nil {
		return nil, err
	}
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	client := *c.http
	client.Timeout = 0
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		res.Body.Close()
		return nil, fmt.Errorf("model events unavailable")
	}
	return res, nil
}

// Observe supplies invalidations, not outcomes. A catalog read reconciles even
// missed or coalesced events. The caller owns a bounded context and fallback.
type ModelEvent struct {
	Model    string
	Type     string
	Progress []FileProgress
}

func (c *Client) Observe(ctx context.Context, notify func(ModelEvent)) error {
	res, err := c.events(ctx)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 4096), 256<<10)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data:") {
			var event struct {
				Model string          `json:"model"`
				Type  string          `json:"event"`
				Data  json.RawMessage `json:"data"`
			}
			if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "data:"))), &event) != nil {
				continue
			}
			progress := []FileProgress(nil)
			if event.Type == "download_progress" {
				var wrapped struct {
					Progress json.RawMessage `json:"progress"`
				}
				_ = json.Unmarshal(event.Data, &wrapped)
				if len(wrapped.Progress) > 0 {
					progress = parseProgress(wrapped.Progress)
				} else {
					progress = parseProgress(event.Data)
				}
			}
			notify(ModelEvent{Model: event.Model, Type: event.Type, Progress: progress})
		}
	}
	return scanner.Err()
}

func (c *Client) Mutate(ctx context.Context, operation, model string) error {
	routes := map[string]string{"load": "/models/load", "unload": "/models/unload", "download": "/models"}
	route, ok := routes[operation]
	if !ok {
		return fmt.Errorf("invalid model operation")
	}
	_, err := c.doContext(ctx, http.MethodPost, route, map[string]string{"model": model})
	return err
}
