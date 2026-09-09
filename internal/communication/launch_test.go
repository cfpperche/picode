package communication

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestLaunchCredentialLifecycle(t *testing.T) {
	s, p, token, _, _ := fixture(t)
	data := t.TempDir()
	c := LaunchConfig{Connection: p, Token: token, URL: "http://localhost:8445" + Path, Adapter: "/adapter/index.ts"}
	if got, err := LoadLaunch(s, data, p.Kind, p.OwnerID); err != nil || got != nil {
		t.Fatalf("manual connection: %v %v", got, err)
	}
	if err := SaveLaunch(data, c); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"connection.json", "mcp.json", "pi.mjs"} {
		st, err := os.Stat(filepath.Join(launchDir(data, p.ID), file))
		if err != nil || st.Mode().Perm() != 0600 {
			t.Fatalf("private %s: %v %v", file, st, err)
		}
	}
	if err := SaveLaunch(data, c); err == nil {
		t.Fatal("rewrote immutable setup")
	}
	got, err := LoadLaunch(s, data, p.Kind, p.OwnerID)
	if err != nil || got == nil || got.Token != token {
		t.Fatal("setup not recovered", err)
	}
	if got, err := LoadLaunch(s, data, "terminal", p.OwnerID); err != nil || got != nil {
		t.Fatal("cross owner setup", err)
	}
	// Owner session changes invalidate setup without relying on file deletion.
	next := "different-session"
	if _, err := s.UpdateAgent(p.OwnerID, store.AgentPatch{SessionPath: &next}); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadLaunch(s, data, p.Kind, p.OwnerID); err != nil || got != nil {
		t.Fatal("stale setup", err)
	}
	original := p.SessionKey
	if _, err := s.UpdateAgent(p.OwnerID, store.AgentPatch{SessionPath: &original}); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokePeer(p.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadLaunch(s, data, p.Kind, p.OwnerID); err != nil || got != nil {
		t.Fatal("revoked setup", err)
	}
	if err := RemoveLaunch(data, p.ID); err != nil || HasLaunch(data, p.ID) {
		t.Fatal("cleanup", err)
	}
}

func TestLaunchOptions(t *testing.T) {
	for _, cli := range []string{"pi", "claude-code", "codex", "opencode", "grok", "hermes"} {
		t.Run(cli, func(t *testing.T) {
			c := LaunchConfig{Connection: store.PeerConnection{ID: "peer_test", PeerOwner: store.PeerOwner{CLI: cli}}, Token: "private-fixture-token", URL: "http://localhost:1234" + Path, Adapter: "/tmp/adapter/index.ts"}
			opts, err := Options(t.TempDir(), c)
			if !LaunchSupported(cli) {
				if err == nil {
					t.Fatal("unsupported CLI accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(strings.Join(opts.Args, " "), c.Token) {
				t.Fatal("credential in argv")
			}
			if cli == "opencode" {
				var config map[string]any
				if err := json.Unmarshal([]byte(opts.Env["OPENCODE_CONFIG_CONTENT"]), &config); err != nil {
					t.Fatal(err)
				}
			}
			if cli == "codex" && opts.Env["PICODE_PEER_TOKEN"] != c.Token {
				t.Fatal("missing scoped env")
			}
		})
	}
}

func TestLaunchTamperAndFailure(t *testing.T) {
	s, p, token, _, other := fixture(t)
	data := t.TempDir()
	c := LaunchConfig{Connection: p, Token: token, URL: "http://localhost" + Path}
	if err := SaveLaunch(data, c); err != nil {
		t.Fatal(err)
	}
	c.Token = other
	raw, _ := json.Marshal(c)
	if err := os.WriteFile(filepath.Join(launchDir(data, p.ID), "connection.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLaunch(s, data, p.Kind, p.OwnerID); err == nil {
		t.Fatal("another owner's credential accepted")
	}
	if launchDir(data, "peer_../../escape") != "" {
		t.Fatal("path traversal")
	}
	if _, err := Options(data, c); err == nil {
		t.Fatal("Pi adapter missing")
	}
}

func TestPiLaunchRegistrationFollowsNativeSession(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	s, p, token, _, _ := fixture(t)
	_ = s
	data := t.TempDir()
	if err := SaveLaunch(data, LaunchConfig{Connection: p, Token: token, URL: "http://localhost" + Path, Adapter: "/adapter"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(launchDir(data, p.ID), "probe.mjs")
	js := `import setup from './pi.mjs';
import assert from 'node:assert/strict';
import {readFileSync,writeFileSync} from 'node:fs';
const c=JSON.parse(readFileSync(new URL('./connection.json',import.meta.url)));
let registrations=0,disposed=0;const handlers={};
const pi={on:(n,h)=>handlers[n]=h,events:{emit:(n,r)=>{assert.equal(n,'pi-mcp-adapter:runtime-register:v1');registrations++;assert.equal(r.definition.headers.Authorization,'Bearer '+c.token);r.result={ok:true,registration:{dispose:async()=>{disposed++}}}}}};
setup(pi);
const ctx=value=>({sessionManager:{getSessionFile:()=>value,getSessionId:()=>value}});
await handlers.session_start({},ctx('another conversation'));assert.equal(registrations,0);
await handlers.session_start({},ctx(c.connection.sessionKey));assert.equal(registrations,1);
await handlers.session_before_switch();assert.equal(disposed,1);
await handlers.session_start({},ctx('another conversation'));assert.equal(registrations,1);
await handlers.session_start({},ctx(c.connection.sessionKey));assert.equal(registrations,2);
await handlers.session_before_fork();assert.equal(disposed,2);
await handlers.session_shutdown();assert.equal(disposed,2);
// A live extension cannot pick up a replacement credential from disk.
writeFileSync(new URL('./connection.json',import.meta.url),JSON.stringify({...c,token:'replacement'}));
await handlers.session_start({},ctx(c.connection.sessionKey));assert.equal(registrations,3);
console.log('native session registration isolation passed');`
	if err := os.WriteFile(path, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(node, path).CombinedOutput(); err != nil {
		t.Fatalf("Pi extension: %v %s", err, out)
	}
}
