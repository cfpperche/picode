package llama

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const DefaultURL = "http://127.0.0.1:8080"

// Model is one router catalog row (no secrets).
type Model struct {
	Path     string         `json:"-"`
	ID       string         `json:"id"`
	Status   string         `json:"status"`
	Progress []FileProgress `json:"progress,omitempty"`
}

// NormalizeURL matches pi: http(s), strip query/hash, drop trailing /v1.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return "", fmt.Errorf("invalid llama.cpp URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("llama.cpp URL must be http or https")
	}
	u.Fragment = ""
	u.RawQuery = ""
	p := strings.TrimRight(u.Path, "/")
	p = strings.TrimSuffix(p, "/v1")
	if p == "" {
		p = ""
	}
	u.Path = p
	return strings.TrimRight(u.String(), "/"), nil
}

// ConnectionError classifies connection failures without returning endpoint secrets.
type ConnectionError struct {
	Code    string
	Message string
}

func (e *ConnectionError) Error() string { return e.Message }

func ConnectionFailure(err error) *ConnectionError {
	var failure *ConnectionError
	if errors.As(err, &failure) {
		return failure
	}
	// Each cause gets its own plain line, so the pane says what to change
	// rather than one "cannot reach" for all of them (2026-09-23 review).
	if errors.Is(err, context.Canceled) {
		return &ConnectionError{"cancelled", "The check was interrupted before the server answered."}
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return &ConnectionError{"timeout", "The server did not respond in time. Check that llama.cpp is running at this address."}
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return &ConnectionError{"dns", "The server name could not be found. Check the address."}
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return &ConnectionError{"refused", "Nothing is listening at this address. Start llama.cpp or check the port."}
	}
	var unknownCA x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var certInvalid x509.CertificateInvalidError
	var tlsRecord tls.RecordHeaderError
	if errors.As(err, &unknownCA) || errors.As(err, &hostname) || errors.As(err, &certInvalid) || errors.As(err, &tlsRecord) {
		return &ConnectionError{"tls", "The secure connection failed: the server's certificate is not trusted, or it does not speak HTTPS."}
	}
	return &ConnectionError{"unreachable", "Cannot reach the server from PiCode."}
}

type Client struct {
	base string
	key  string
	http *http.Client
}

func New(base, key string) (*Client, error) {
	u, err := NormalizeURL(base)
	if err != nil {
		return nil, err
	}
	return &Client{base: u, key: key, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	return c.doContext(context.Background(), method, path, body)
}

func (c *Client) doContext(ctx context.Context, method, path string, body any) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if readErr != nil {
		return nil, readErr
	}
	if res.StatusCode >= 300 {
		if res.StatusCode == 401 || res.StatusCode == 403 {
			return nil, &ConnectionError{"authentication", "The server rejected the API key."}
		}
		if res.StatusCode == 400 || res.StatusCode == 409 || res.StatusCode == 422 {
			return nil, &ConnectionError{"request_rejected", "The server rejected this model operation."}
		}
		if res.StatusCode == 404 || res.StatusCode == 405 {
			return nil, &ConnectionError{"unsupported", "This server does not support model management."}
		}
		return nil, &ConnectionError{"server_error", fmt.Sprintf("The server returned HTTP %d.", res.StatusCode)}
	}
	return raw, nil
}

func (c *Client) List() ([]Model, error) { return c.ListContext(context.Background()) }

func (c *Client) ListContext(ctx context.Context) ([]Model, error) {
	raw, err := c.doContext(ctx, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data []struct {
			Path   string `json:"path"`
			ID     string `json:"id"`
			Status struct {
				Value    string          `json:"value"`
				Progress json.RawMessage `json:"progress"`
			} `json:"status"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &payload) != nil || payload.Data == nil {
		return nil, &ConnectionError{"unsupported", "This server did not return a llama.cpp model catalog."}
	}
	out := make([]Model, 0, len(payload.Data))
	for _, m := range payload.Data {
		if m.ID == "" || len(m.ID) > 512 {
			return nil, &ConnectionError{"unsupported", "This server did not return a llama.cpp model catalog."}
		}
		st := m.Status.Value
		if st == "" || len(st) > 32 {
			return nil, &ConnectionError{"unsupported", "This server did not return model management states."}
		}
		out = append(out, Model{ID: m.ID, Path: m.Path, Status: st, Progress: parseProgress(m.Status.Progress)})
	}
	return out, nil
}

func (c *Client) Load(id string) error {
	_, err := c.do(http.MethodPost, "/models/load", map[string]string{"model": id})
	return err
}

func (c *Client) Unload(id string) error {
	_, err := c.do(http.MethodPost, "/models/unload", map[string]string{"model": id})
	return err
}

func (c *Client) Download(id string) error {
	_, err := c.do(http.MethodPost, "/models", map[string]string{"model": id})
	return err
}

func (c *Client) Wait(id, want string, max time.Duration) error {
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		list, err := c.List()
		if err != nil {
			return err
		}
		var st string
		for _, m := range list {
			if m.ID == id {
				st = m.Status
				break
			}
		}
		if st == "failed" {
			return fmt.Errorf("model operation failed")
		}
		switch want {
		case "loaded":
			if st == "loaded" || st == "sleeping" {
				return nil
			}
			if st == "failed" {
				return fmt.Errorf("load failed")
			}
		case "unloaded":
			if st == "" || st == "unloaded" {
				return nil
			}
		case "downloaded":
			if st == "unloaded" || st == "loaded" || st == "sleeping" {
				return nil
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	return fmt.Errorf("timed out")
}

func (c *Client) ShortTimeout() {
	c.http.Timeout = 2 * time.Second
}
