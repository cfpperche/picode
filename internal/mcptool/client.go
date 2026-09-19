package mcptool

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Identity is the house identity tuple (ADR-0143): a managed agent's id,
// else the terminal the CLI runs in. Both come from the environment the CLI
// inherited from PiCode; the server never invents one.
type Identity struct {
	Agent string `json:"agent"`
	Term  string `json:"term,omitempty"`
}

// Env is how the server reads its environment — os.Getenv in production, a
// map in tests.
type Env func(key string) string

// MapEnv makes an Env from a map.
func MapEnv(m map[string]string) Env { return func(k string) string { return m[k] } }

// IdentityFrom reads PICODE_AGENT_ID / PICODE_TERM_ID.
func IdentityFrom(env Env) Identity {
	return Identity{Agent: strings.TrimSpace(env("PICODE_AGENT_ID")), Term: strings.TrimSpace(env("PICODE_TERM_ID"))}
}

// Empty is true for a caller PiCode cannot name.
func (id Identity) Empty() bool { return id.Agent == "" && id.Term == "" }

// Principal is the grant key the daemon will derive (internal/grant.Key),
// used here only to name the captures directory.
func (id Identity) Principal() string {
	if id.Agent != "" {
		return id.Agent
	}
	if id.Term != "" {
		return "term:" + id.Term
	}
	return ""
}

// NoIdentity is the answer to a call from outside PiCode's launch paths.
const NoIdentity = "no identity — open the CLI in a PiCode terminal (or run it as a PiCode agent) so PiCode knows who is asking"

// DataDir mirrors the pi packages: PICODE_DATA, else ~/.picode.
func DataDir(env Env, home string) string {
	if d := strings.TrimSpace(env("PICODE_DATA")); d != "" {
		return d
	}
	return filepath.Join(strings.TrimRight(home, "/"), ".picode")
}

var originRe = regexp.MustCompile(`^https?://[^/\s@]+/?$`)

// ResolveURL finds the daemon: PICODE_URL (an origin), PICODE_TERM_URL (what
// PiCode's terminals carry, ADR-0056), else server.json in the data dir.
func ResolveURL(env Env, dataDir string) (string, error) {
	for _, key := range []string{"PICODE_URL", "PICODE_TERM_URL"} {
		if v := strings.TrimSpace(env(key)); v != "" {
			if !originRe.MatchString(v) {
				return "", errors.New(key + " must be an origin like https://box:8445")
			}
			return strings.TrimRight(v, "/"), nil
		}
	}
	raw, err := os.ReadFile(filepath.Join(dataDir, "server.json"))
	if err != nil {
		return "", errors.New("no server.json")
	}
	return ParseServerJSON(raw)
}

// ParseServerJSON reads the daemon's server.json for its url.
func ParseServerJSON(raw []byte) (string, error) {
	var s struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", errors.New("server.json is not valid JSON")
	}
	if !strings.HasPrefix(s.URL, "http://") && !strings.HasPrefix(s.URL, "https://") {
		return "", errors.New("server.json has no usable url")
	}
	return strings.TrimRight(s.URL, "/"), nil
}

var tokenRe = regexp.MustCompile(`^[0-9a-fA-F]{32,128}$`)

// ParseToken accepts only a hex install token.
func ParseToken(text string) string {
	t := strings.TrimSpace(text)
	if tokenRe.MatchString(t) {
		return t
	}
	return ""
}

// ReadToken is PICODE_TOKEN, else the token file — read on every call so a
// rotation (`picode token rotate`) needs no restart.
func ReadToken(env Env, dataDir string) string {
	if t := ParseToken(env("PICODE_TOKEN")); t != "" {
		return t
	}
	raw, _ := os.ReadFile(filepath.Join(dataDir, "token"))
	return ParseToken(string(raw))
}

// RejectUnauthorizedFor is the pi packages' rule: the daemon's self-signed
// certificate is accepted on loopback and verified anywhere else.
func RejectUnauthorizedFor(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return true
	}
	h := strings.Trim(u.Hostname(), "[]")
	if h == "localhost" {
		return false
	}
	ip := net.ParseIP(h)
	return ip == nil || !ip.IsLoopback()
}

// Daemon posts one JSON body to one route, or reads one, and returns the
// status and body.
type Daemon interface {
	Post(ctx context.Context, path string, body []byte) (int, []byte, error)
	Get(ctx context.Context, path string) (int, []byte, error)
}

// HTTPDaemon is the production Daemon.
type HTTPDaemon struct {
	URL    string
	Token  func() string
	client *http.Client
}

// NewHTTPDaemon builds a client for the daemon at base.
func NewHTTPDaemon(base string, token func() string) *HTTPDaemon {
	tr := &http.Transport{}
	if !RejectUnauthorizedFor(base) {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // loopback, the daemon's own self-signed certificate
	}
	return &HTTPDaemon{URL: base, Token: token, client: &http.Client{Timeout: 90 * time.Second, Transport: tr}}
}

// Post implements Daemon.
func (d *HTTPDaemon) Post(ctx context.Context, path string, body []byte) (int, []byte, error) {
	return d.do(ctx, http.MethodPost, path, body)
}

// Get implements Daemon.
func (d *HTTPDaemon) Get(ctx context.Context, path string) (int, []byte, error) {
	return d.do(ctx, http.MethodGet, path, nil)
}

func (d *HTTPDaemon) do(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, d.URL+path, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if d.Token != nil {
		if t := d.Token(); t != "" {
			req.Header.Set("Authorization", "Bearer "+t)
		}
	}
	res, err := d.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	out, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	return res.StatusCode, out, err
}

// Caller is what a family needs to make one call: where the daemon is, who
// is asking, and where captures go on disk.
type Caller struct {
	Daemon   Daemon
	Identity Identity
	Captures string // directory; empty disables the on-disk copy
	// Unreachable, when set, is why there is no daemon (no server.json):
	// every call answers it instead of dialing.
	Unreachable string
}

// Answer is the daemon's reply, parsed the way the pi packages parse it: a
// non-2xx carries its message verbatim.
type Answer struct {
	Head   string // "action" or "verb", whatever the route names
	Output map[string]any
	Raw    json.RawMessage
}

// ParseAnswer reads a reply body; the head key names the route's own label.
func ParseAnswer(status int, body []byte, head string) (Answer, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(body, &data); err != nil {
		if status >= 400 {
			return Answer{}, errors.New("PiCode answered " + itoa(status))
		}
		return Answer{}, errors.New("PiCode answered with something that is not JSON")
	}
	if status >= 400 {
		var msg string
		_ = json.Unmarshal(data["error"], &msg)
		if strings.TrimSpace(msg) == "" {
			msg = "PiCode answered " + itoa(status)
		}
		return Answer{}, errors.New(msg)
	}
	a := Answer{Raw: data["output"]}
	_ = json.Unmarshal(data[head], &a.Head)
	if a.Head == "" {
		a.Head = "unknown"
	}
	if len(a.Raw) > 0 {
		_ = json.Unmarshal(a.Raw, &a.Output)
	}
	return a, nil
}

// post runs one call through the caller: identity first, reachability
// second, then the wire. An error here is what the model reads.
func (c *Caller) post(ctx context.Context, path string, body map[string]any, head string) (Answer, error) {
	if c.Identity.Empty() {
		return Answer{}, errors.New(NoIdentity)
	}
	if c.Unreachable != "" {
		return Answer{}, errors.New("PiCode is not reachable (" + c.Unreachable + ") — is the daemon running?")
	}
	body["agent"] = c.Identity.Agent
	body["term"] = c.Identity.Term
	raw, err := json.Marshal(body)
	if err != nil {
		return Answer{}, err
	}
	status, res, err := c.Daemon.Post(ctx, path, raw)
	if err != nil {
		return Answer{}, err
	}
	return ParseAnswer(status, res, head)
}
