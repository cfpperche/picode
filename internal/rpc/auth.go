package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
)

type mcpAuthJob struct {
	mu       sync.Mutex
	client   *Client
	owned    bool
	out      string
	open     string
	cancel   context.CancelFunc
	doneCh   chan struct{}
	err      error
	finished sync.Once
}

func (j *mcpAuthJob) openURL() string {
	if j == nil || j.open == "" {
		return ""
	}
	b, err := os.ReadFile(j.open)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "http://") {
		return ""
	}
	return s
}

func (j *mcpAuthJob) finish(err error) {
	j.finished.Do(func() {
		j.err = err
		close(j.doneCh)
		j.closeOwned()
	})
}

func (j *mcpAuthJob) closeOwned() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.owned && j.client != nil {
		j.client.Close()
		j.owned = false
	}
	if j.cancel != nil {
		j.cancel()
		j.cancel = nil
	}
}

func (j *mcpAuthJob) watchFile() {
	t := time.NewTicker(400 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-j.doneCh:
			return
		case <-t.C:
			b, err := os.ReadFile(j.out)
			if err != nil {
				continue
			}
			var res struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if json.Unmarshal(b, &res) != nil {
				continue
			}
			if !res.OK {
				j.finish(fmt.Errorf("%s", res.Error))
				return
			}
			j.finish(nil)
			return
		}
	}
}

// AuthTestInstant writes a successful result without spawning pi (tests only).
var AuthTestInstant bool

// BeginMCPAuth starts headless adapter OAuth (callback only, no paste UI).
func (r *Runtime) BeginMCPAuth(ctx context.Context, agentID, cwd, name, serverURL, returnTo string) (string, error) {
	if r == nil || r.AgentCmd == "" {
		return "", fmt.Errorf("pi is not configured")
	}
	if serverURL == "" {
		return "", fmt.Errorf("server has no URL")
	}
	r.CloseMCPAuth()
	if cwd == "" {
		cwd, _ = os.UserHomeDir()
	}
	dataDir := r.DataDir
	if dataDir == "" {
		dataDir = os.TempDir()
	}
	ext, err := mcp.EnsureAuthExt(dataDir)
	if err != nil {
		return "", err
	}
	adapter := mcp.AdapterDir()
	if adapter == "" {
		return "", fmt.Errorf("install the MCP adapter first")
	}
	id := newID()
	out := mcp.AuthOutPath(dataDir, id)
	openPath := out + ".url"
	if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
		return "", err
	}
	_ = os.Remove(out)
	_ = os.Remove(openPath)
	if AuthTestInstant {
		job := &mcpAuthJob{out: out, open: openPath, doneCh: make(chan struct{})}
		r.putAuthJob(id, job)
		_ = os.WriteFile(out, []byte(`{"ok":true}`), 0o600)
		go job.watchFile()
		return id, nil
	}
	args := []string{"--mode", "rpc", "--no-session", "-e", ext}
	if agentID != "" && r.store != nil {
		if a, err := r.store.GetAgent(agentID); err == nil {
			args = append(args, a.CLIFlags()...)
		}
	}
	env := []string{
		"PICODE_MCP_AUTH=" + name,
		"PICODE_MCP_AUTH_URL=" + serverURL,
		"PICODE_MCP_AUTH_OUT=" + out,
		"PICODE_MCP_ADAPTER=" + adapter,
		"PICODE_MCP_RETURN=" + returnTo,
		"PICODE_MCP_OPEN=" + openPath,
	}
	client, err := Start(r.AgentCmd, args, cwd, env...)
	if err != nil {
		return "", err
	}
	_, cancel := context.WithCancel(context.Background())
	job := &mcpAuthJob{client: client, owned: true, out: out, open: openPath, cancel: cancel, doneCh: make(chan struct{})}
	r.putAuthJob(id, job)
	go job.watchFile()
	go r.expireAuthJob(id, 5*time.Minute)
	go func() {
		<-client.Done()
		if _, err := os.Stat(out); err != nil {
			job.finish(fmt.Errorf("sign-in stopped"))
		}
	}()
	return id, nil
}

// ReplyMCPAuth cancels a headless sign-in (value is ignored).
func (r *Runtime) ReplyMCPAuth(id, _ string, cancelled bool) error {
	job := r.takeAuthJob(id)
	if job == nil {
		return fmt.Errorf("sign-in expired")
	}
	if cancelled {
		job.finish(fmt.Errorf("cancelled"))
		return nil
	}
	job.finish(nil)
	return nil
}

// MCPAuthStatus is pending, finished, or gone. openURL is the authorize page for the GUI to open.
func (r *Runtime) MCPAuthStatus(id string) (done bool, err error, openURL string, found bool) {
	job := r.peekAuthJob(id)
	if job == nil {
		return false, nil, "", false
	}
	openURL = job.openURL()
	select {
	case <-job.doneCh:
		return true, job.err, openURL, true
	default:
		return false, nil, openURL, true
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
	job.finish(fmt.Errorf("sign-in timed out"))
}

// CloseMCPAuth kills short-lived auth processes.
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
			j.finish(fmt.Errorf("cancelled"))
		}(job)
	}
	wg.Wait()
}
