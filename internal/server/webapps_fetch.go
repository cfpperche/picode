package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	webappTimeout     = 4 * time.Second
	webappMaxRedirect = 4
	webappBodyCap     = 256 << 10
	webappIconCap     = 256 << 10
	webappTitleCap    = 120
)

type webappFetchResult struct {
	finalURL *url.URL
	body     []byte
	headers  http.Header
	status   int
}

type webappInaccessibleError struct{ reason string }

func (e webappInaccessibleError) Error() string { return e.reason }

func isPrivateHost(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
	}
	h := strings.ToLower(host)
	return h == "localhost" || strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local")
}

func isLocalName(host string) bool {
	h := strings.ToLower(host)
	return h == "localhost" || strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local")
}

func webappNewClient() *http.Client {
	return &http.Client{
		Timeout: webappTimeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           webappDialer,
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
			MaxIdleConns:          2,
			IdleConnTimeout:       time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= webappMaxRedirect {
				return errors.New("too many redirects")
			}
			if req.URL.User != nil || (req.URL.Scheme != "http" && req.URL.Scheme != "https") {
				return errors.New("redirect left http(s)")
			}
			next := req.URL.Hostname()
			origin := via[0].URL.Hostname()
			if isLocalName(next) && !isLocalName(origin) {
				return errors.New("redirect moved onto a local-only host")
			}
			if isPrivateHost(next) && !isPrivateHost(origin) {
				return errors.New("redirect from a public host to a private address")
			}
			return nil
		},
	}
}

func webappDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ipAddrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	explicitLocal := isLocalName(host) || net.ParseIP(host) != nil
	if len(ipAddrs) == 0 {
		return nil, fmt.Errorf("webapp fetch: %s has no addresses", host)
	}
	for _, ia := range ipAddrs {
		ip := ia.IP
		private := ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
		if private && !explicitLocal {
			return nil, fmt.Errorf("webapp fetch: %s resolves to a private address", host)
		}
	}
	var d net.Dialer
	return d.DialContext(ctx, network, (&net.TCPAddr{IP: ipAddrs[0].IP, Port: portNum(port)}).String())
}

func portNum(port string) int {
	n, _ := strconv.Atoi(port)
	return n
}

// webappFetchPage retrieves the page the user named for metadata only:
// a body past the cap is truncated, not refused — the <head> (title,
// manifest link, icon) sits in the first kilobytes, and sites like
// github.com stream more than any sane cap. Everything a truncated body
// cannot prove falls back honestly (title → host name, favicon).
func webappFetchPage(ctx context.Context, client *http.Client, raw string, maxBytes int64) (webappFetchResult, error) {
	return webappFetch(ctx, client, raw, maxBytes, true)
}

// webappFetch retrieves manifest and icon bytes, where a truncated body is
// a wrong answer: over the cap is an error.
func webappFetch(ctx context.Context, client *http.Client, raw string, maxBytes int64, truncate bool) (webappFetchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, webappTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return webappFetchResult{}, webappInaccessibleError{"invalid address"}
	}
	req.Header.Set("User-Agent", "PiCode/1.0 (webapp install)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,image/*;q=0.9,*/*;q=0.8")
	res, err := client.Do(req)
	if err != nil {
		return webappFetchResult{}, webappInaccessibleError{err.Error()}
	}
	defer func() { _ = res.Body.Close() }()
	if truncate {
		body, err := ioReadTruncated(res.Body, maxBytes)
		if err != nil {
			return webappFetchResult{}, webappInaccessibleError{err.Error()}
		}
		return webappFetchResult{finalURL: res.Request.URL, body: body, headers: res.Header, status: res.StatusCode}, nil
	}
	body, err := ioReadLimited(res.Body, maxBytes)
	if err != nil {
		return webappFetchResult{}, webappInaccessibleError{err.Error()}
	}
	return webappFetchResult{finalURL: res.Request.URL, body: body, headers: res.Header, status: res.StatusCode}, nil
}

func ioReadLimited(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("response too large")
	}
	return data, nil
}

func ioReadTruncated(r io.Reader, max int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, max))
}
