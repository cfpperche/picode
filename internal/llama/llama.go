package llama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultURL = "http://127.0.0.1:8080"

// Model is one router catalog row (no secrets).
type Model struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// NormalizeURL matches pi: http(s), strip query/hash, drop trailing /v1.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
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
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rdr)
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
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("llama.cpp HTTP %d", res.StatusCode)
	}
	return raw, nil
}

func (c *Client) List() ([]Model, error) {
	raw, err := c.do(http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data []struct {
			ID     string `json:"id"`
			Status struct {
				Value string `json:"value"`
			} `json:"status"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &payload) != nil || payload.Data == nil {
		return nil, fmt.Errorf("not a llama.cpp router")
	}
	out := make([]Model, 0, len(payload.Data))
	for _, m := range payload.Data {
		if m.ID == "" {
			return nil, fmt.Errorf("not a llama.cpp router")
		}
		st := m.Status.Value
		if st == "" {
			st = "unknown"
		}
		out = append(out, Model{ID: m.ID, Status: st})
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
			if st != "" && st != "downloading" {
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
