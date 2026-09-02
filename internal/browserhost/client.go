package browserhost

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client talks to a running PiCode server. Resolve is called per request so
// a port rebind (ADR-0007) is picked up without restarting Chrome.
type Client struct {
	Resolve func() (string, error)
	HTTP    *http.Client
}

// NewClient reads ~/.picode/server.json (or PICODE_DATA) on every call.
func NewClient() *Client {
	return &Client{
		Resolve: ReadServerURL,
		HTTP:    insecureClient(),
	}
}

func insecureClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

// Handle is the native-messaging Handler.
func (c *Client) Handle(req Request) Reply {
	switch req.Type {
	case "ping":
		return c.ping(req)
	case "agents":
		return c.agents(req)
	case "send":
		return c.send(req)
	default:
		return Reply{OK: false, Type: req.Type, ID: req.ID, Error: "unknown request", Code: "bad_type"}
	}
}

func (c *Client) ping(req Request) Reply {
	base, err := c.base()
	if err != nil {
		return down(req, err)
	}
	if err := c.get(base+"/api/health", nil); err != nil {
		return down(req, err)
	}
	c.noteDevice(base, req.DeviceID)
	return Reply{OK: true, Type: "ping", ID: req.ID, URL: base}
}

func (c *Client) agents(req Request) Reply {
	base, err := c.base()
	if err != nil {
		return down(req, err)
	}
	var body struct {
		Agents []AgentInfo `json:"agents"`
	}
	if err := c.get(base+"/api/extension/agents", &body); err != nil {
		return down(req, err)
	}
	c.noteDevice(base, req.DeviceID)
	return Reply{OK: true, Type: "agents", ID: req.ID, URL: base, Agents: body.Agents}
}

func (c *Client) send(req Request) Reply {
	base, err := c.base()
	if err != nil {
		return down(req, err)
	}
	payload := map[string]any{
		"agentId": req.AgentID,
		"message": req.Message,
	}
	if req.Tab != nil {
		payload["tab"] = req.Tab
	}
	if req.Image != nil {
		payload["image"] = req.Image
	}
	var body struct {
		OK      bool   `json:"ok"`
		Error   string `json:"error"`
		Started bool   `json:"started"`
	}
	status, err := c.post(base+"/api/extension/send", payload, &body)
	if err != nil {
		code := "picode_down"
		if status > 0 {
			code = "http"
		}
		msg := err.Error()
		if body.Error != "" {
			msg = body.Error
		}
		if status == http.StatusConflict {
			code = "interactive"
		}
		if status == http.StatusBadRequest && strings.Contains(strings.ToLower(msg), "can't be sent") {
			code = "chrome_page"
		}
		if status == http.StatusNotFound {
			code = "not_found"
		}
		return Reply{OK: false, Type: "send", ID: req.ID, Error: msg, Code: code, URL: base}
	}
	return Reply{OK: true, Type: "send", ID: req.ID, URL: base, Started: body.Started}
}

func (c *Client) base() (string, error) {
	if c == nil || c.Resolve == nil {
		return "", fmt.Errorf("PiCode is not running")
	}
	u, err := c.Resolve()
	if err != nil {
		return "", err
	}
	u = strings.TrimRight(u, "/")
	if u == "" {
		return "", fmt.Errorf("PiCode is not running")
	}
	return u, nil
}

func (c *Client) get(url string, dest any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	authorize(req)
	res, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("PiCode is not running")
	}
	defer res.Body.Close()
	return decode(res, dest)
}

func (c *Client) post(url string, payload any, dest any) (int, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	authorize(req)
	res, err := c.http().Do(req)
	if err != nil {
		return 0, fmt.Errorf("PiCode is not running")
	}
	defer res.Body.Close()
	if err := decode(res, dest); err != nil {
		return res.StatusCode, err
	}
	if res.StatusCode >= 400 {
		return res.StatusCode, fmt.Errorf("http %d", res.StatusCode)
	}
	return res.StatusCode, nil
}

func decode(res *http.Response, dest any) error {
	b, err := io.ReadAll(io.LimitReader(res.Body, MaxMessage))
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		var env struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(b, &env) == nil && env.Error != "" {
			return fmt.Errorf("%s", env.Error)
		}
		return fmt.Errorf("http %d", res.StatusCode)
	}
	if dest == nil || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return insecureClient()
}

func (c *Client) noteDevice(base, id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	_, _ = c.post(base+"/api/devices/ping", map[string]any{
		"id": id, "host": true, "kind": "extension",
	}, nil)
}

func down(req Request, err error) Reply {
	msg := "PiCode is not running."
	if err != nil && err.Error() != "" && err.Error() != "PiCode is not running" {
		// Keep the empty-state copy stable; the code is what the UI switches on.
		_ = err
	}
	return Reply{OK: false, Type: req.Type, ID: req.ID, Error: msg, Code: "picode_down"}
}

// ReadServerURL parses <data>/server.json. The file is rewritten on every
// bind, so callers should not cache it across requests.
// authorize adds the install token (<data>/token, ADR-0049) when present.
func authorize(req *http.Request) {
	b, err := os.ReadFile(filepath.Join(dataDir(), "token"))
	if err != nil {
		return
	}
	if tok := strings.TrimSpace(string(b)); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

func ReadServerURL() (string, error) {
	path := filepath.Join(dataDir(), "server.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("PiCode is not running")
	}
	var s struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(b, &s); err != nil || strings.TrimSpace(s.URL) == "" {
		return "", fmt.Errorf("PiCode is not running")
	}
	return strings.TrimRight(s.URL, "/"), nil
}

// DataDir is where the daemon keeps server.json and the install token.
func DataDir() string { return dataDir() }

func dataDir() string {
	if d := strings.TrimSpace(os.Getenv("PICODE_DATA")); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picode")
}
