package usage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) get(ctx context.Context, rawURL, bearer string, extra map[string]string) (body []byte, status int, err error) {
	if rawURL == "" {
		return nil, 0, fmt.Errorf("empty url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return body, res.StatusCode, nil
}

func (c *Client) postForm(ctx context.Context, rawURL string, form map[string]string, extra map[string]string) (body []byte, status int, err error) {
	vals := url.Values{}
	for k, v := range form {
		vals.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(vals.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return body, res.StatusCode, nil
}

func (c *Client) postJSON(ctx context.Context, rawURL, jsonBody string, extra map[string]string) (body []byte, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(jsonBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return body, res.StatusCode, nil
}

func (c *Client) postBytes(ctx context.Context, rawURL string, payload []byte, extra map[string]string) (body []byte, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return body, res.StatusCode, nil
}
