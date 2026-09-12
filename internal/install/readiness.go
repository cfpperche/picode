package install

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Busy is one owner the running daemon reports as working (ADR-0086).
type Busy struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Why  string `json:"why"`
}

// ErrDeployBusy: the restart would end someone's turn. Deploy refuses
// before touching the installed binary; `picode deploy --force` overrides.
var ErrDeployBusy = errors.New("agents are working")

// Readiness asks the daemon named by <data>/server.json who is working.
// Nil, nil means "nothing to protect": no server.json, no daemon listening,
// or a daemon too old to answer the route — a refusal there would make the
// first deploy of the guard impossible. Tests replace it.
var Readiness = func(dataDir string) ([]Busy, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, "server.json"))
	if err != nil {
		return nil, nil
	}
	var disc struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &disc); err != nil || disc.URL == "" {
		return nil, nil
	}
	client := &http.Client{
		Timeout: 8 * time.Second,
		// The daemon's certificate is self-signed or mkcert; this call never
		// leaves the machine and carries nothing worth stealing.
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
	}
	resp, err := client.Get(strings.TrimRight(disc.URL, "/") + "/api/deploy/readiness")
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var out struct {
		Busy []Busy `json:"busy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("readiness: %w", err)
	}
	return out.Busy, nil
}

// Env vars the daemon injects into the panes it manages: a terminal carries
// its own id, a managed agent its agent id. Named here rather than imported
// so this package keeps depending on nothing of the server's.
const (
	termIDEnv  = "PICODE_TERM_ID"
	agentIDEnv = "PICODE_AGENT_ID"
)

// dropSelf removes the pane the deploy is running in. The readiness route
// answers about the whole fleet and cannot know who asked, so a session
// deploying from its own pane always finds itself mid-turn — it is working
// by the very act of asking — and a guard that counted it could never open
// for its own caller. Whoever types `picode deploy` is consenting to lose
// that pane; every other turn is what the guard exists to protect.
func dropSelf(busy []Busy, getenv func(string) string) []Busy {
	self := map[string]bool{}
	for _, key := range []string{termIDEnv, agentIDEnv} {
		if id := getenv(key); id != "" {
			self[id] = true
		}
	}
	if len(self) == 0 {
		return busy
	}
	out := make([]Busy, 0, len(busy))
	for _, o := range busy {
		if !self[o.ID] {
			out = append(out, o)
		}
	}
	return out
}

// guardDeploy turns a busy fleet into the refusal the caller prints.
func guardDeploy(dataDir string, force bool) error {
	if force {
		return nil
	}
	busy, err := Readiness(dataDir)
	if err != nil {
		return err
	}
	busy = dropSelf(busy, os.Getenv)
	if len(busy) == 0 {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, " — a restart would end %d turn(s):\n", len(busy))
	for _, o := range busy {
		name := o.Name
		if name == "" {
			name = o.ID
		}
		fmt.Fprintf(&b, "  %s %q is %s\n", o.Kind, name, o.Why)
	}
	b.WriteString("Wait, or `picode deploy --force` (PICODE_DEPLOY_FORCE=1 for make deploy).")
	return fmt.Errorf("%w%s", ErrDeployBusy, b.String())
}
