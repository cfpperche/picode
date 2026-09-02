package browserhost

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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

// NewClient reads <data>/remote.json, else server.json, on every call.
// HTTP is nil: the client is picked per request from the resolved URL
// (see clientFor).
func NewClient() *Client {
	return &Client{Resolve: ReadServerURL}
}

func insecureClient() *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

// Remote is <data>/remote.json (ADR-0050): a PiCode on another machine.
// When present it wins over server.json and the local token, which
// describe a daemon that is not there.
type Remote struct {
	URL    string `json:"url"`
	Token  string `json:"token"`
	CAFile string `json:"caFile,omitempty"` // PEM to trust for that server (mkcert root)
}

// RemoteFile is the file name inside the data dir.
const RemoteFile = "remote.json"

// ReadRemote parses the remote file; ok is false when absent or unusable.
func ReadRemote() (Remote, bool) {
	b, err := os.ReadFile(filepath.Join(dataDir(), RemoteFile))
	if err != nil {
		return Remote{}, false
	}
	var rc Remote
	if json.Unmarshal(b, &rc) != nil || !strings.HasPrefix(rc.URL, "http") {
		return Remote{}, false
	}
	rc.URL = strings.TrimRight(strings.TrimSpace(rc.URL), "/")
	rc.Token = strings.TrimSpace(rc.Token)
	return rc, true
}

// WriteRemote records a remote PiCode (0600: it holds the token).
func WriteRemote(rc Remote) (string, error) {
	rc.URL = strings.TrimRight(strings.TrimSpace(rc.URL), "/")
	if !strings.HasPrefix(rc.URL, "https://") && !strings.HasPrefix(rc.URL, "http://") {
		return "", fmt.Errorf("server must be an http(s) URL")
	}
	raw, err := json.MarshalIndent(rc, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dataDir(), RemoteFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, append(raw, '\n'), 0o600)
}

// VerifyTLSFor: a self-signed or mkcert cert is accepted for loopback and
// for IP literals (the LAN/tailnet addresses mkcert covers); a name must
// present a chain the system — or remote.json's caFile — trusts.
func VerifyTLSFor(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return true
	}
	h := u.Hostname()
	if h == "localhost" || net.ParseIP(h) != nil {
		return false
	}
	return true
}

// clientFor picks the HTTP client for a resolved base URL.
func (c *Client) clientFor(base string) *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	if !VerifyTLSFor(base) {
		return insecureClient()
	}
	cfg := &tls.Config{}
	if rc, ok := ReadRemote(); ok && rc.CAFile != "" {
		if pem, err := os.ReadFile(rc.CAFile); err == nil {
			pool, _ := x509.SystemCertPool()
			if pool == nil {
				pool = x509.NewCertPool()
			}
			pool.AppendCertsFromPEM(pem)
			cfg.RootCAs = pool
		}
	}
	return &http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{TLSClientConfig: cfg}}
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
	case "act-next":
		return c.actNext(req)
	case "act-result":
		return c.actResult(req)
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
	if req.Act {
		payload["act"] = true
	}
	var body struct {
		OK       bool   `json:"ok"`
		Error    string `json:"error"`
		Started  bool   `json:"started"`
		Watching bool   `json:"watching"`
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
	return Reply{OK: true, Type: "send", ID: req.ID, URL: base, Started: body.Started, Watching: body.Watching}
}

// actNext polls the server for the agent's next actuation batch. tab is
// the asking tab's origin; a batch for another origin is not claimed.
func (c *Client) actNext(req Request) Reply {
	base, err := c.base()
	if err != nil {
		return down(req, err)
	}
	q := "?agent=" + url.QueryEscape(req.AgentID)
	if req.Tab != nil && req.Tab.URL != "" {
		q += "&tab=" + url.QueryEscape(req.Tab.URL)
	}
	var body struct {
		Watching bool          `json:"watching"`
		Blocked  string        `json:"blocked"`
		Batch    *ActBatchWire `json:"batch"`
	}
	if err := c.get(base+"/api/extension/act/next"+q, &body); err != nil {
		return down(req, err)
	}
	c.noteDevice(base, req.DeviceID)
	return Reply{OK: true, Type: "act-next", ID: req.ID, URL: base,
		Watching: body.Watching, Blocked: body.Blocked, Batch: body.Batch}
}

// actResult posts one batch's outcomes back to PiCode.
func (c *Client) actResult(req Request) Reply {
	base, err := c.base()
	if err != nil {
		return down(req, err)
	}
	var body struct {
		Watching bool `json:"watching"`
	}
	if _, err := c.post(base+"/api/extension/act/"+url.PathEscape(req.ID)+"/result", map[string]any{
		"outcomes": req.Outcomes,
		"stopped":  req.Stopped,
	}, &body); err != nil {
		return Reply{OK: false, Type: "act-result", ID: req.ID, Error: err.Error(), Code: "http", URL: base}
	}
	return Reply{OK: true, Type: "act-result", ID: req.ID, URL: base, Watching: body.Watching}
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
	res, err := c.clientFor(url).Do(req)
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
	res, err := c.clientFor(url).Do(req)
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

// authorize adds the bearer: remote.json's token when a remote is
// configured, else the install token (<data>/token, ADR-0049).
func authorize(req *http.Request) {
	if rc, ok := ReadRemote(); ok {
		if rc.Token != "" {
			req.Header.Set("Authorization", "Bearer "+rc.Token)
		}
		return
	}
	b, err := os.ReadFile(filepath.Join(dataDir(), "token"))
	if err != nil {
		return
	}
	if tok := strings.TrimSpace(string(b)); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

// ReadServerURL is remote.json's URL when present, else <data>/server.json.
// server.json is rewritten on every bind, so callers should not cache it
// across requests.
func ReadServerURL() (string, error) {
	if rc, ok := ReadRemote(); ok {
		return rc.URL, nil
	}
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
