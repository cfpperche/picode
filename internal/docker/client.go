// Package docker implements the bounded Docker Engine subset used by the App
// and pi-sysadmin (ADR-0065). It never executes a shell or elevates privileges.
package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const apiVersion = "1.44"
const MaxLogBytes = 64 * 1024

var fullID = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Client struct {
	Endpoint string
	HTTP     *http.Client
}

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return e.Message }

type Container struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Image      string `json:"image"`
	State      string `json:"state"`
	Status     string `json:"status"`
	Project    string `json:"project,omitempty"`
	Service    string `json:"service,omitempty"`
	WorkingDir string `json:"workingDir,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	Health     string `json:"health,omitempty"`
	TTY        bool   `json:"-"`
}

type Stats struct {
	CPUPercent  float64 `json:"cpuPercent"`
	MemoryBytes uint64  `json:"memoryBytes"`
	LimitBytes  uint64  `json:"limitBytes"`
	SampledAt   string  `json:"sampledAt"`
}
type Logs struct {
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
	SampledAt string `json:"sampledAt"`
}

// EndpointFrom respects Docker's explicit context before DOCKER_HOST. The
// injected context lookup keeps resolution testable without local credentials.
func EndpointFrom(env func(string) string, contextHost func() (string, error)) (string, error) {
	endpoint := strings.TrimSpace(env("PICODE_DOCKER_HOST"))
	if endpoint == "" {
		if env("DOCKER_CONTEXT") == "" {
			endpoint = strings.TrimSpace(env("DOCKER_HOST"))
		}
		if endpoint == "" {
			if contextHost == nil {
				return "", errors.New("Choose a local Docker connection")
			}
			var err error
			endpoint, err = contextHost()
			if err != nil {
				return "", err
			}
		}
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "unix" || u.Host != "" || !filepath.IsAbs(u.Path) || u.Path == "/" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", errors.New("Choose a local Docker connection. This version supports Unix sockets")
	}
	return "unix://" + filepath.Clean(u.Path), nil
}

func LocalClient(ctx context.Context) (*Client, error) {
	endpoint, err := EndpointFrom(os.Getenv, func() (string, error) {
		if _, err := exec.LookPath("docker"); err != nil {
			return "unix:///var/run/docker.sock", nil
		}
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cctx, "docker", "context", "inspect", "--format", "{{json .Endpoints.docker.Host}}")
		out, err := cmd.Output()
		if err != nil {
			return "", errors.New("Docker connection could not be resolved. Check the selected Docker context")
		}
		var endpoint string
		if err = json.Unmarshal(out, &endpoint); err != nil {
			return "", errors.New("Docker returned an invalid connection")
		}
		return endpoint, nil
	})
	if err != nil {
		return nil, err
	}
	return NewClient(endpoint)
}

func NewClient(endpoint string) (*Client, error) {
	endpoint, err := EndpointFrom(func(k string) string {
		if k == "PICODE_DOCKER_HOST" {
			return endpoint
		}
		return ""
	}, nil)
	if err != nil {
		return nil, err
	}
	u, _ := url.Parse(endpoint)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "unix", u.Path)
	}, MaxIdleConns: 4, IdleConnTimeout: 30 * time.Second}
	return &Client{Endpoint: endpoint, HTTP: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Close() { c.HTTP.CloseIdleConnections() }

func (c *Client) request(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Docker is not reachable: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		defer res.Body.Close()
		var body struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body)
		if body.Message == "" {
			body.Message = "Docker returned " + res.Status
		}
		return nil, &APIError{Status: res.StatusCode, Message: body.Message}
	}
	return res, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	res, err := c.request(ctx, "GET", path)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return json.NewDecoder(io.LimitReader(res.Body, 4*1024*1024)).Decode(out)
}

func (c *Client) Check(ctx context.Context) error {
	var v struct {
		APIVersion    string
		MinAPIVersion string
	}
	if err := c.get(ctx, "/version", &v); err != nil {
		return err
	}
	parse := func(v string) int {
		var a, b int
		if _, err := fmt.Sscanf(v, "%d.%d", &a, &b); err != nil {
			return 0
		}
		return a*1000 + b
	}
	if parse(v.APIVersion) < parse(apiVersion) || (v.MinAPIVersion != "" && parse(v.MinAPIVersion) > parse(apiVersion)) {
		return errors.New("This Docker API version is incompatible with PiCode (requires API 1.44)")
	}
	return nil
}

func labels(c *Container, l map[string]string) {
	c.Project = l["com.docker.compose.project"]
	c.Service = l["com.docker.compose.service"]
	c.WorkingDir = l["com.docker.compose.project.working_dir"]
}

func (c *Client) Containers(ctx context.Context) ([]Container, error) {
	var rows []struct {
		ID                   string `json:"Id"`
		Names                []string
		Image, State, Status string
		Labels               map[string]string
	}
	if err := c.get(ctx, "/v"+apiVersion+"/containers/json?all=true", &rows); err != nil {
		return nil, err
	}
	out := []Container{}
	for _, r := range rows {
		v := Container{ID: r.ID, Image: r.Image, State: r.State, Status: r.Status, Name: r.ID[:min(12, len(r.ID))]}
		if len(r.Names) > 0 {
			v.Name = strings.TrimPrefix(r.Names[0], "/")
		}
		labels(&v, r.Labels)
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Client) Inspect(ctx context.Context, id string) (Container, error) {
	if !fullID.MatchString(id) {
		return Container{}, errors.New("Select a container using its full ID")
	}
	var r struct {
		ID     string `json:"Id"`
		Name   string
		Config struct {
			Image  string
			Tty    bool
			Labels map[string]string
		}
		State struct {
			Status, StartedAt string
			Health            *struct{ Status string }
		}
	}
	if err := c.get(ctx, "/v"+apiVersion+"/containers/"+id+"/json", &r); err != nil {
		return Container{}, err
	}
	v := Container{ID: r.ID, Name: strings.TrimPrefix(r.Name, "/"), Image: r.Config.Image, State: r.State.Status, Status: r.State.Status, StartedAt: r.State.StartedAt, TTY: r.Config.Tty}
	if r.State.Health != nil {
		v.Health = r.State.Health.Status
	}
	labels(&v, r.Config.Labels)
	if v.ID != id {
		return Container{}, errors.New("Docker returned a different container")
	}
	return v, nil
}

func (c *Client) Stats(ctx context.Context, id string) (Stats, error) {
	if !fullID.MatchString(id) {
		return Stats{}, errors.New("invalid container ID")
	}
	type cpu struct {
		CPUUsage struct {
			TotalUsage  uint64
			PercpuUsage []uint64
		}
		SystemCPUUsage uint64
		OnlineCPUs     uint64
	}
	// Docker uses snake_case for resource statistics.
	var r struct {
		CPU    json.RawMessage `json:"cpu_stats"`
		Prev   json.RawMessage `json:"precpu_stats"`
		Memory struct {
			Usage, Limit uint64
			Stats        map[string]uint64
		} `json:"memory_stats"`
	}
	if err := c.get(ctx, "/v"+apiVersion+"/containers/"+id+"/stats?stream=false", &r); err != nil {
		return Stats{}, err
	}
	parseCPU := func(raw json.RawMessage) cpu {
		var x struct {
			Usage struct {
				Total uint64   `json:"total_usage"`
				Per   []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			System uint64 `json:"system_cpu_usage"`
			Online uint64 `json:"online_cpus"`
		}
		_ = json.Unmarshal(raw, &x)
		var v cpu
		v.CPUUsage.TotalUsage = x.Usage.Total
		v.CPUUsage.PercpuUsage = x.Usage.Per
		v.SystemCPUUsage = x.System
		v.OnlineCPUs = x.Online
		return v
	}
	a, b := parseCPU(r.CPU), parseCPU(r.Prev)
	n := a.OnlineCPUs
	if n == 0 {
		n = uint64(len(a.CPUUsage.PercpuUsage))
	}
	s := Stats{MemoryBytes: r.Memory.Usage, LimitBytes: r.Memory.Limit, SampledAt: time.Now().UTC().Format(time.RFC3339)}
	cache := r.Memory.Stats["inactive_file"]
	if cache == 0 {
		cache = r.Memory.Stats["total_inactive_file"]
	}
	if cache < s.MemoryBytes {
		s.MemoryBytes -= cache
	}
	if a.SystemCPUUsage > b.SystemCPUUsage && a.CPUUsage.TotalUsage >= b.CPUUsage.TotalUsage {
		s.CPUPercent = float64(a.CPUUsage.TotalUsage-b.CPUUsage.TotalUsage) / float64(a.SystemCPUUsage-b.SystemCPUUsage) * float64(n) * 100
	}
	return s, nil
}

func (c *Client) Logs(ctx context.Context, container Container) (Logs, error) {
	if !fullID.MatchString(container.ID) {
		return Logs{}, errors.New("invalid container ID")
	}
	res, err := c.request(ctx, "GET", "/v"+apiVersion+"/containers/"+container.ID+"/logs?stdout=true&stderr=true&timestamps=true&tail=200")
	if err != nil {
		return Logs{}, err
	}
	defer res.Body.Close()
	text, truncated, err := ReadLogs(res.Body, container.TTY)
	return Logs{Text: text, Truncated: truncated, SampledAt: time.Now().UTC().Format(time.RFC3339)}, err
}

var ansi = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)

// ReadLogs handles Docker's multiplex framing and TTY's raw byte stream.
func ReadLogs(r io.Reader, tty bool) (string, bool, error) {
	var out bytes.Buffer
	if tty {
		if _, err := io.Copy(&out, io.LimitReader(r, MaxLogBytes+1)); err != nil {
			return "", false, err
		}
	} else {
		for out.Len() <= MaxLogBytes {
			var head [8]byte
			_, err := io.ReadFull(r, head[:])
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", false, err
			}
			if head[0] > 2 || head[1] != 0 || head[2] != 0 || head[3] != 0 {
				return "", false, errors.New("invalid Docker log frame")
			}
			n := int64(binary.BigEndian.Uint32(head[4:]))
			remaining := int64(MaxLogBytes + 1 - out.Len())
			if _, err := io.CopyN(&out, r, min(n, remaining)); err != nil {
				return "", false, err
			}
			if n > remaining {
				break
			}
		}
	}
	truncated := out.Len() > MaxLogBytes
	raw := out.Bytes()
	if truncated {
		raw = raw[:MaxLogBytes]
	}
	clean := ansi.ReplaceAllString(strings.ToValidUTF8(string(raw), "�"), "")
	clean = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, clean)
	return clean, truncated, nil
}

func (c *Client) Mutate(ctx context.Context, id, action string) error {
	if !fullID.MatchString(id) || (action != "start" && action != "stop" && action != "restart") {
		return errors.New("invalid Docker action")
	}
	path := "/v" + apiVersion + "/containers/" + id + "/" + action
	if action != "start" {
		path += "?t=10"
	}
	res, err := c.request(ctx, "POST", path)
	if err == nil {
		_ = res.Body.Close()
	}
	return err
}

func (c *Client) Events(ctx context.Context, changed func()) error {
	res, err := c.request(ctx, "GET", "/v"+apiVersion+"/events?filters="+url.QueryEscape(`{"type":["container"],"event":["create","destroy","start","stop","restart","die","pause","unpause","rename","update","health_status"]}`))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	d := json.NewDecoder(res.Body)
	for {
		var ev json.RawMessage
		if err = d.Decode(&ev); err != nil {
			return err
		}
		changed()
	}
}
