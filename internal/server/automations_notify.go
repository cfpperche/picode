package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/store"
)

// notifyHTTP posts to notify URLs; tests replace it.
var notifyHTTP = &http.Client{Timeout: 10 * time.Second}

// notifyRetryAfter is the pause before the one retry; tests shorten it.
var notifyRetryAfter = 5 * time.Second

// notifyOut POSTs the run's outcome to the automation's notify URL, in
// the background: one try, one retry on a network error or a 5xx. The
// outcome — sent or not — is an event on the feed (automation.notify),
// so it is visible without watching logs. done reports completion to
// tests; nil callers do not wait.
func (r *automationRunner) notifyOut(a store.Automation, run store.Run, status, reason, summary string, done chan<- error) {
	if a.NotifyURL == nil || strings.TrimSpace(*a.NotifyURL) == "" {
		if done != nil {
			done <- nil
		}
		return
	}
	link := ""
	if pub := strings.TrimRight(r.deps.publicURL(), "/"); pub != "" {
		link = pub + "/#/automations/" + a.ID
	}
	body := automate.BuildNotify(a.Name, status, reason, run.CostUSD, link, summary).Marshal()
	url := *a.NotifyURL
	go func() {
		err := postNotify(url, body)
		if err != nil && !strings.Contains(err.Error(), "not retried") {
			time.Sleep(notifyRetryAfter)
			err = postNotify(url, body)
		}
		out := map[string]string{"automationId": a.ID, "runId": run.ID, "status": status}
		if err != nil {
			out["error"] = err.Error()
			log.Printf("automations: notify %s for %s: %v", url, a.Name, err)
		}
		_ = r.deps.Store.AppendEvent("automation.notify", nil, nil, out)
		if done != nil {
			done <- err
		}
	}()
}

func postNotify(url string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "picode-automations")
	res, err := notifyHTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 500 {
		return fmt.Errorf("%s", res.Status)
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("%s (not retried)", res.Status)
	}
	return nil
}
