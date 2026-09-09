package communication

// Process-local setup for an explicitly enrolled conversation (ADR-0106).
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

const ServerName = "picode_communication"

type LaunchConfig struct {
	Connection store.PeerConnection `json:"connection"`
	Token      string               `json:"token"`
	URL        string               `json:"url"`
	Adapter    string               `json:"adapter,omitempty"`
	CABundle   string               `json:"caBundle,omitempty"`
}

type LaunchOptions struct {
	Args []string
	Env  map[string]string
}

func LaunchSupported(cli string) bool {
	switch cli {
	case "pi", "claude-code", "codex", "opencode":
		return true
	}
	return false
}

func launchDir(data, id string) string {
	if data == "" || !strings.HasPrefix(id, "peer_") || filepath.Base(id) != id || strings.ContainsAny(id, `/\\`) {
		return ""
	}
	return filepath.Join(data, "communication", id)
}

// SaveLaunch writes immutable private setup. The database remains authoritative:
// partial files cannot authorize a revoked credential. Never rewrite native files.
func SaveLaunch(data string, c LaunchConfig) error {
	dir := launchDir(data, c.Connection.ID)
	if dir == "" || !LaunchSupported(c.Connection.CLI) {
		return errors.New("automatic connection is unavailable for this CLI")
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return err
	}
	pending, err := os.MkdirTemp(filepath.Dir(dir), ".setup-")
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(pending)
		}
	}()
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(pending, "connection.json"), raw, 0600); err != nil {
		return err
	}
	if c.CABundle != "" {
		if err = os.WriteFile(filepath.Join(pending, "ca.pem"), []byte(c.CABundle), 0600); err != nil {
			return err
		}
	}
	server := map[string]any{"type": "http", "url": c.URL, "headers": map[string]string{"Authorization": "Bearer " + c.Token}}
	raw, _ = json.Marshal(map[string]any{"mcpServers": map[string]any{ServerName: server}})
	if err = os.WriteFile(filepath.Join(pending, "mcp.json"), raw, 0600); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(pending, "pi.mjs"), []byte(piLaunchExtension), 0600); err != nil {
		return err
	}
	if err = os.Rename(pending, dir); err != nil {
		return err
	}
	ok = true
	return nil
}

func RemoveLaunch(data, id string) error {
	if dir := launchDir(data, id); dir != "" {
		return os.RemoveAll(dir)
	}
	return nil
}

func HasLaunch(data, id string) bool {
	dir := launchDir(data, id)
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "connection.json"))
	return err == nil
}

// LoadLaunch only returns the active connection for this owner. Revalidation
// excludes disabled credentials, another workspace and stale recorded sessions.
func LoadLaunch(s *store.Store, data, kind, owner string) (*LaunchConfig, error) {
	if s == nil || data == "" {
		return nil, nil
	}
	peers, err := s.ListPeerConnections()
	if err != nil {
		return nil, err
	}
	for _, p := range peers {
		if !p.Active || p.Kind != kind || p.OwnerID != owner {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(launchDir(data, p.ID), "connection.json"))
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var c LaunchConfig
		if err = json.Unmarshal(raw, &c); err != nil {
			return nil, errors.New("connection setup is unreadable; replace the connection")
		}
		valid, err := s.AuthorizePeer(c.Token)
		if err != nil || valid.ID != p.ID || c.Connection.ID != p.ID {
			return nil, store.ErrPeerDenied
		}
		c.Connection = valid
		return &c, nil
	}
	return nil, nil
}

// Options never carries tokens in argv. They live in a private file (Pi/Claude)
// or the launched command's environment (Codex/OpenCode), not the pane shell.
func Options(data string, c LaunchConfig) (LaunchOptions, error) {
	dir := launchDir(data, c.Connection.ID)
	if dir == "" {
		return LaunchOptions{}, store.ErrPeerInput
	}
	o := LaunchOptions{Env: map[string]string{}}
	if c.CABundle != "" {
		key := "NODE_EXTRA_CA_CERTS"
		if c.Connection.CLI == "codex" {
			key = "CODEX_CA_CERTIFICATE"
		}
		o.Env[key] = filepath.Join(dir, "ca.pem")
	}
	switch c.Connection.CLI {
	case "pi":
		if c.Adapter == "" {
			return o, errors.New("install pi-mcp-adapter in Packages before connecting")
		}
		o.Args = []string{"-e", c.Adapter, "-e", filepath.Join(dir, "pi.mjs")}
	case "claude-code":
		o.Args = []string{"--mcp-config", filepath.Join(dir, "mcp.json")}
	case "codex":
		u, _ := json.Marshal(c.URL)
		o.Args = []string{"-c", fmt.Sprintf("mcp_servers.%s.url=%s", ServerName, u), "-c", "mcp_servers." + ServerName + ".bearer_token_env_var=\"PICODE_PEER_TOKEN\""}
		o.Env["PICODE_PEER_TOKEN"] = c.Token
	case "opencode":
		raw, _ := json.Marshal(map[string]any{"mcp": map[string]any{ServerName: map[string]any{"type": "remote", "url": c.URL, "headers": map[string]string{"Authorization": "Bearer " + c.Token}, "oauth": false}}})
		o.Env["OPENCODE_CONFIG_CONTENT"] = string(raw)
	default:
		return o, errors.New("automatic connection is unavailable for this CLI")
	}
	return o, nil
}

func AgentOptions(s *store.Store, data, id string) (LaunchOptions, error) {
	c, err := LoadLaunch(s, data, "agent", id)
	if err != nil || c == nil {
		return LaunchOptions{}, err
	}
	return Options(data, *c)
}

const piLaunchExtension = `import { readFileSync } from "node:fs";
const config = JSON.parse(readFileSync(new URL("./connection.json", import.meta.url), "utf8"));
export default function(pi) {
 let registration;
 pi.on("session_start", async (_event, ctx) => {
  await registration?.dispose(); registration = undefined;
  const recorded = config.connection.sessionKey;
  const actual = config.connection.kind === "agent" ? ctx.sessionManager.getSessionFile() : ctx.sessionManager.getSessionId();
  if (actual !== recorded) return;
  const request = { version: 1, name: "picode_communication", definition: {
   url: config.url, headers: { Authorization: "Bearer " + config.token }
  }};
  pi.events.emit("pi-mcp-adapter:runtime-register:v1", request);
  if (!request.result?.ok) throw new Error("PiCode messages need pi-mcp-adapter 2.32.1 or later. Check Packages and resume this conversation.");
  registration = request.result.registration;
 });
 pi.on("session_before_switch", async () => { await registration?.dispose(); registration = undefined; });
 pi.on("session_before_fork", async () => { await registration?.dispose(); registration = undefined; });
 pi.on("session_shutdown", async () => { await registration?.dispose(); registration = undefined; });
}
`
