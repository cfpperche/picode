package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
)

type mcpAuthJob struct {
	mu     sync.Mutex
	client *Client
	owned  bool
	errCh  chan error
	cancel context.CancelFunc
	doneCh chan struct{}
	err    error
}

func (j *mcpAuthJob) watch() {
	j.err = <-j.errCh
	close(j.doneCh)
	j.closeOwned()
}

func (j *mcpAuthJob) closeOwned() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.owned {
		j.client.Close()
		j.owned = false
	}
	if j.cancel != nil {
		j.cancel()
		j.cancel = nil
	}
}

// BeginMCPAuth runs `/mcp-auth name`. Reuses a live managed agent when
// agentID is running; otherwise a short `pi --mode rpc --no-session`.
func (r *Runtime) BeginMCPAuth(ctx context.Context, agentID, cwd, name string) (string, string, error) {
	if r == nil || r.AgentCmd == "" {
		return "", "", fmt.Errorf("pi is not configured")
	}
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	if ma := r.Get(agentID); ma != nil {
		return r.beginAuthOn(ctx, sendCtx, sendCancel, ma.client, false, func(c context.Context) error {
			return ma.SendPromptCtx(c, "/mcp-auth "+name)
		})
	}
	if cwd == "" {
		cwd, _ = os.UserHomeDir()
	}
	args := []string{"--mode", "rpc", "--no-session"}
	if agentID != "" && r.store != nil {
		if a, err := r.store.GetAgent(agentID); err == nil {
			args = append(args, a.CLIFlags()...)
		}
	}
	client, err := Start(r.AgentCmd, args, cwd)
	if err != nil {
		sendCancel()
		return "", "", err
	}
	return r.beginAuthOn(ctx, sendCtx, sendCancel, client, true, func(c context.Context) error {
		_, err := client.Send(c, Command{Type: "prompt", Body: map[string]any{"message": "/mcp-auth " + name}})
		return err
	})
}

func (r *Runtime) beginAuthOn(waitCtx, sendCtx context.Context, sendCancel context.CancelFunc, client *Client, owned bool, send func(context.Context) error) (string, string, error) {
	uiCh := make(chan map[string]any, 1)
	unsub := client.Subscribe(func(ev Event) {
		if ev.EventType() != "extension_ui_request" {
			return
		}
		var body map[string]any
		if json.Unmarshal([]byte(ev), &body) != nil {
			return
		}
		method, _ := body["method"].(string)
		if method != "input" && method != "editor" {
			return
		}
		select {
		case uiCh <- body:
		default:
		}
	})
	errCh := make(chan error, 1)
	go func() { errCh <- send(sendCtx) }()
	select {
	case ui := <-uiCh:
		unsub()
		id, _ := ui["id"].(string)
		title, _ := ui["title"].(string)
		msg, _ := ui["message"].(string)
		ph, _ := ui["placeholder"].(string)
		if id == "" {
			sendCancel()
			if owned {
				client.Close()
			}
			return "", "", fmt.Errorf("sign-in had no prompt")
		}
		job := &mcpAuthJob{client: client, owned: owned, errCh: errCh, cancel: sendCancel, doneCh: make(chan struct{})}
		r.putAuthJob(id, job)
		go job.watch()
		go r.expireAuthJob(id, 5*time.Minute)
		return id, mcp.AuthURLFromUI(title, msg, ph), nil
	case err := <-errCh:
		unsub()
		sendCancel()
		if owned {
			client.Close()
		}
		if err != nil {
			return "", "", err
		}
		return "", "", nil
	case <-waitCtx.Done():
		unsub()
		sendCancel()
		if owned {
			client.Close()
		}
		return "", "", waitCtx.Err()
	}
}

// ReplyMCPAuth answers the extension UI prompt from BeginMCPAuth.
func (r *Runtime) ReplyMCPAuth(id, value string, cancelled bool) error {
	job := r.peekAuthJob(id)
	if job == nil {
		return fmt.Errorf("sign-in expired")
	}
	body := map[string]any{"type": "extension_ui_response", "id": id}
	if cancelled {
		body["cancelled"] = true
	} else {
		body["value"] = value
	}
	err := job.client.SendRaw(body)
	select {
	case <-job.doneCh:
	case <-time.After(30 * time.Second):
	}
	r.takeAuthJob(id)
	if err != nil {
		return err
	}
	return job.err
}

// MCPAuthStatus is pending, finished, or gone.
func (r *Runtime) MCPAuthStatus(id string) (done bool, err error, found bool) {
	job := r.peekAuthJob(id)
	if job == nil {
		return false, nil, false
	}
	select {
	case <-job.doneCh:
		return true, job.err, true
	default:
		return false, nil, true
	}
}

func (r *Runtime) putAuthJob(id string, job *mcpAuthJob) {
	r.authMu.Lock()
	if r.authJobs == nil {
		r.authJobs = map[string]*mcpAuthJob{}
	}
	r.authJobs[id] = job
	r.authMu.Unlock()
}

func (r *Runtime) peekAuthJob(id string) *mcpAuthJob {
	r.authMu.Lock()
	defer r.authMu.Unlock()
	if r.authJobs == nil {
		return nil
	}
	return r.authJobs[id]
}

func (r *Runtime) takeAuthJob(id string) *mcpAuthJob {
	r.authMu.Lock()
	defer r.authMu.Unlock()
	if r.authJobs == nil {
		return nil
	}
	job := r.authJobs[id]
	delete(r.authJobs, id)
	return job
}

func (r *Runtime) expireAuthJob(id string, d time.Duration) {
	time.Sleep(d)
	job := r.takeAuthJob(id)
	if job == nil {
		return
	}
	job.closeOwned()
}

// CloseMCPAuth kills short-lived auth processes. Borrowed agents stay.
func (r *Runtime) CloseMCPAuth() {
	r.authMu.Lock()
	jobs := r.authJobs
	r.authJobs = map[string]*mcpAuthJob{}
	r.authMu.Unlock()
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(j *mcpAuthJob) {
			defer wg.Done()
			j.closeOwned()
		}(job)
	}
	wg.Wait()
}
