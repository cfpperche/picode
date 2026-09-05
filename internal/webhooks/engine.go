// Package webhooks delivers durable PiCode events to owner-selected services.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/hookurl"
	"github.com/cfpperche/picode/internal/store"
)

var ErrBusy = errors.New("A delivery is in progress. Try again shortly.")

type Engine struct {
	Store  *store.Store
	Client *http.Client
	Now    func() time.Time
	busy   sync.Map
}

// NewClient does not inherit proxies, cookies or authorization. Resolving and
// validating the addresses in DialContext prevents DNS rebinding to metadata.
func NewClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns: 8, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var last error
			for _, ip := range ips {
				if !hookurl.AllowedIP(ip.IP) || ip.Zone != "" {
					return nil, errors.New("destination not allowed")
				}
			}
			for _, ip := range ips {
				conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				last = err
			}
			if last == nil {
				last = errors.New("destination has no addresses")
			}
			return nil, last
		},
	}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func New(st *store.Store) *Engine { return &Engine{Store: st, Client: NewClient()} }
func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func Signature(secret, timestamp string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(timestamp + "."))
	_, _ = h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func Matches(types []string, kind string) bool {
	if strings.HasPrefix(kind, "webhook.") {
		return false
	}
	for _, prefix := range types {
		if strings.HasPrefix(kind, prefix) {
			return true
		}
	}
	return false
}

func Backoff(failures int) time.Duration {
	d := time.Minute
	for i := 1; i < failures && d < time.Hour; i++ {
		d *= 2
	}
	if d > time.Hour {
		return time.Hour
	}
	return d
}

type Result struct {
	Delivered bool   `json:"delivered"`
	Status    int    `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (e *Engine) send(ctx context.Context, w store.Webhook, ev store.Event) Result {
	if hookurl.Validate(w.URL) != nil {
		return Result{Error: "Invalid destination. Edit the webhook URL."}
	}
	body, err := json.Marshal(ev)
	if err != nil {
		return Result{Error: "Could not encode the event."}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return Result{Error: "Invalid destination. Edit the webhook URL."}
	}
	stamp := strconv.FormatInt(e.now().Unix(), 10)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PiCode-Webhooks/1")
	req.Header.Set("X-Picode-Timestamp", stamp)
	req.Header.Set("X-Picode-Event-Id", strconv.FormatInt(ev.ID, 10))
	req.Header.Set("X-Picode-Event-Type", ev.Type)
	req.Header.Set("X-Picode-Signature", Signature(w.Secret, stamp, body))
	res, err := e.Client.Do(req)
	if err != nil {
		// net/http errors contain the full URL, sometimes query credentials.
		return Result{Error: "Could not reach the receiver. Check its address and availability."}
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Result{Status: res.StatusCode, Error: fmt.Sprintf("Receiver returned HTTP %d.", res.StatusCode)}
	}
	return Result{Delivered: true, Status: res.StatusCode}
}

// Test uses the same signing path without acknowledging events or changing
// retry state. A paused subscription can be tested explicitly by the owner.
func (e *Engine) Test(ctx context.Context, id string) (Result, error) {
	if _, loaded := e.busy.LoadOrStore(id, true); loaded {
		return Result{}, ErrBusy
	}
	defer e.busy.Delete(id)
	w, err := e.Store.GetWebhook(id)
	if err != nil {
		return Result{}, err
	}
	ev := store.Event{Type: "webhook.test", Data: json.RawMessage(`{"message":"PiCode webhook test"}`), CreatedAt: e.now().Format(time.RFC3339Nano)}
	return e.send(ctx, w, ev), nil
}

// DeliverOnce processes a bounded page and at most one POST. The caller can
// run it repeatedly; persisted cursor/backoff make a daemon restart harmless.
func (e *Engine) DeliverOnce(ctx context.Context, id string) error {
	if _, loaded := e.busy.LoadOrStore(id, true); loaded {
		return nil
	}
	defer e.busy.Delete(id)
	before, err := e.Store.GetWebhook(id)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !before.Enabled {
		return nil
	}
	if next, err := time.Parse(time.RFC3339Nano, before.NextAttemptAt); err == nil && next.After(e.now()) {
		return nil
	}
	after := before
	oldest, err := e.Store.OldestEventID()
	if err != nil {
		return err
	}
	if oldest > 0 && before.Cursor < oldest-1 {
		after.Cursor, err = e.Store.LatestEventID()
		if err != nil {
			return err
		}
		after.LastStatus = "missed"
		after.LastError = "Some events expired before delivery. Resuming with new events."
		after.Failures = 0
		after.NextAttemptAt = ""
		return e.Store.SaveWebhookProgress(before, after)
	}
	events, err := e.Store.ListEventsSince(before.Cursor, 100)
	if err != nil {
		return err
	}
	for _, ev := range events {
		if !Matches(before.Types, ev.Type) {
			after.Cursor = ev.ID
			continue
		}
		result := e.send(ctx, before, ev)
		if ctx.Err() != nil {
			return ctx.Err()
		} // restart retries, including uncertain successes
		after.LastAttemptAt = e.now().Format(time.RFC3339Nano)
		after.LastError = result.Error
		if result.Delivered {
			after.Cursor = ev.ID
			after.LastStatus = "delivered"
			after.Failures = 0
			after.NextAttemptAt = ""
		} else {
			after.LastStatus = "retrying"
			after.Failures++
			after.NextAttemptAt = e.now().Add(Backoff(after.Failures)).Format(time.RFC3339Nano)
		}
		return e.Store.SaveWebhookProgress(before, after)
	}
	if after.Cursor != before.Cursor {
		return e.Store.SaveWebhookProgress(before, after)
	}
	return nil
}

func (e *Engine) Loop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	slots := make(chan struct{}, 4)
	var wg sync.WaitGroup
	defer wg.Wait()
	// Rotating the first row avoids starvation when every receiver is slow.
	offset := 0
	for {
		rows, err := e.Store.ListWebhooks()
		if err != nil {
			log.Printf("webhooks: list: %v", err)
		}
		for i := 0; i < len(rows); i++ {
			w := rows[(i+offset)%len(rows)]
			if !w.Enabled {
				continue
			}
			if _, active := e.busy.Load(w.ID); active {
				continue
			}
			select {
			case slots <- struct{}{}:
				wg.Add(1)
				go func(id string) {
					defer wg.Done()
					defer func() { <-slots }()
					if err := e.DeliverOnce(ctx, id); err != nil && ctx.Err() == nil && !errors.Is(err, store.ErrWebhookConflict) {
						log.Printf("webhooks: %s: %v", id, err)
					}
				}(w.ID)
			default:
			}
		}
		offset++
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
