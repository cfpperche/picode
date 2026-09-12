package communication

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIAndMCPShareMailbox(t *testing.T) {
	s, p, pt, q, qt := fixture(t)
	server := httptest.NewServer(Handler(s))
	defer server.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "connection.json")
	raw, _ := json.Marshal(LaunchConfig{Connection: p, Token: pt, URL: server.URL})
	os.WriteFile(path, raw, 0600)
	run := func(args ...string) map[string]any {
		t.Helper()
		args = append([]string{args[0], "--connection", path}, args[1:]...)
		var out, errout bytes.Buffer
		if err := RunCLI(context.Background(), args, strings.NewReader(""), &out, &errout, func(string) string { return "" }); err != nil {
			t.Fatal(err, errout.String())
		}
		var v map[string]any
		if err := json.Unmarshal(out.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if len(run("contacts")["contacts"].([]any)) != 1 {
		t.Fatal("missing terminal contact")
	}
	sent := run("send", "--to", q.ID, "--request-id", "cli-retry", "--body", "hello from CLI")
	retry := run("send", "--to", q.ID, "--request-id", "cli-retry", "--body", "hello from CLI")
	if sent["message"].(map[string]any)["id"] != retry["message"].(map[string]any)["id"] {
		t.Fatal("duplicated retry")
	}
	receiver := client(t, context.Background(), server.URL, qt)
	inbox := invoke[readOutput](t, context.Background(), receiver, "read_messages", readInput{})
	if len(inbox.Messages) != 1 || inbox.Messages[0].Body != "hello from CLI" {
		t.Fatal(inbox)
	}
	if _, err := s.SendPeerMessage(qt, p.ID, "mcp-reply", "reply", inbox.Messages[0].ID); err != nil {
		t.Fatal(err)
	}
	got := run("read")
	received := got["messages"].([]any)[0].(map[string]any)["id"].(string)
	run("ack", received)
	if len(run("read")["messages"].([]any)) != 0 {
		t.Fatal("ack not shared")
	}
	if err := s.RevokePeer(p.ID); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RunCLI(context.Background(), []string{"contacts", "--connection", path}, strings.NewReader(""), &out, &out, func(string) string { return "" }); err == nil {
		t.Fatal("revoked CLI remained authorized")
	}
}

func TestNativeCLIIdentity(t *testing.T) {
	_, p, _, _, _ := fixture(t)
	for _, cli := range []string{"grok", "hermes", "codex"} {
		t.Run(cli, func(t *testing.T) {
			dir := t.TempDir()
			p.CLI = cli
			p.SessionKey = "native-A"
			path := filepath.Join(dir, "peer_one", "connection.json")
			os.MkdirAll(filepath.Dir(path), 0700)
			raw, _ := json.Marshal(LaunchConfig{Connection: p, Token: "private", URL: "http://localhost/mcp"})
			os.WriteFile(path, raw, 0600)
			key := map[string]string{"grok": "GROK_SESSION_ID", "hermes": "HERMES_SESSION_ID", "codex": "CODEX_THREAD_ID"}[cli]
			options, err := NativeOptions(filepath.Dir(dir), cli)
			if err != nil {
				t.Fatal(err)
			}
			count := 3
			if len(options.Env) != count || options.Env["PICODE_MESSAGES_BIN"] == "" {
				t.Fatal("native bootstrap must contain discovery only", options.Env)
			}
			// A daemon started by Codex inherits its marker and thread context.
			env := map[string]string{"PICODE_MESSAGES_CLI": "codex", "CODEX_THREAD_ID": "parent", "CODEX_SESSION_ID": "parent", "PICODE_TERM_ID": p.OwnerID}
			for k, v := range options.Env {
				env[k] = v
			}
			env["PICODE_MESSAGES_DIR"], env[key] = dir, "native-A"
			if cli == "codex" {
				env["CODEX_SESSION_ID"] = "native-A"
			}
			if options.Env["PICODE_MESSAGES_CLI"] != cli {
				t.Fatal("inherited CLI marker was not replaced")
			}
			parent := p
			parent.CLI, parent.SessionKey = "codex", "parent"
			parentRaw, _ := json.Marshal(LaunchConfig{Connection: parent, Token: "parent-private", URL: "http://localhost/mcp"})
			os.MkdirAll(filepath.Join(dir, "peer_parent"), 0700)
			os.WriteFile(filepath.Join(dir, "peer_parent", "connection.json"), parentRaw, 0600)
			get := func(k string) string { return env[k] }
			if c, err := ResolveCLIConnection("", get); err != nil || c.Connection.SessionKey != "native-A" {
				t.Fatal(c, err)
			}
			for _, sid := range []string{"native-B", ""} {
				env[key] = sid
				if _, err := ResolveCLIConnection("", get); err == nil {
					t.Fatal("new/missing session inherited old identity")
				}
			}
			env[key] = "native-B"
			if _, err := ResolveCLIConnection(path, get); err == nil {
				t.Fatal("explicit stale file overrode native identity")
			}
			env[key] = "native-A"
			if c, err := ResolveCLIConnection("", get); err != nil || c.Connection.SessionKey != "native-A" {
				t.Fatal("native resume did not rediscover setup", err)
			}
			otherKey := map[string]string{"grok": "HERMES_SESSION_ID", "hermes": "GROK_SESSION_ID", "codex": "GROK_SESSION_ID"}[cli]
			env[otherKey] = "other"
			if _, err := ResolveCLIConnection("", get); err == nil {
				t.Fatal("contradictory identity")
			}
			delete(env, otherKey)
			if cli != "codex" {
				// An inherited non-Codex session must not fill a missing own SID,
				// including when the caller supplies an explicit connection file.
				env[key], env[otherKey] = "", "foreign-parent"
				foreign := p
				foreign.CLI = map[string]string{"grok": "hermes", "hermes": "grok"}[cli]
				foreign.SessionKey = "foreign-parent"
				foreignRaw, _ := json.Marshal(LaunchConfig{Connection: foreign, Token: "private", URL: "http://localhost/mcp"})
				foreignPath := filepath.Join(dir, "peer_foreign", "connection.json")
				os.MkdirAll(filepath.Dir(foreignPath), 0700)
				os.WriteFile(foreignPath, foreignRaw, 0600)
				for _, explicit := range []string{"", foreignPath, path} {
					if _, err := ResolveCLIConnection(explicit, get); err == nil {
						t.Fatal("missing own SID borrowed a foreign or explicit connection")
					}
				}
				delete(env, otherKey)
				env[key] = "native-A"
			}
			os.MkdirAll(filepath.Join(dir, "peer_duplicate"), 0700)
			os.WriteFile(filepath.Join(dir, "peer_duplicate", "connection.json"), raw, 0600)
			if _, err := ResolveCLIConnection("", get); err == nil {
				t.Fatal("ambiguous match")
			}
		})
	}
}

func TestCodexNativeConnectionCompatibility(t *testing.T) {
	_, p, _, _, _ := fixture(t)
	p.CLI, p.SessionKey, p.OwnerID = "codex", "current-thread", "current-terminal"
	for _, tc := range []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"modern", map[string]string{"PICODE_MESSAGES_CLI": "codex"}, true},
		{"legacy data", map[string]string{}, true},
		{"legacy home", map[string]string{"PICODE_DATA": ""}, true},
		{"matching alias", map[string]string{"CODEX_SESSION_ID": "current-thread"}, true},
		{"conflicting alias", map[string]string{"CODEX_SESSION_ID": "parent-thread"}, false},
		{"another terminal", map[string]string{"PICODE_TERM_ID": "another-terminal"}, false},
		{"parent ambient identity", map[string]string{"PICODE_TERM_ID": ""}, false},
		{"missing thread", map[string]string{"CODEX_THREAD_ID": "", "PICODE_MESSAGES_CLI": "codex"}, false},
		{"new conversation", map[string]string{"CODEX_THREAD_ID": "new-thread"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			data := filepath.Join(home, ".picode")
			path := filepath.Join(data, "communication", "peer_one", "connection.json")
			os.MkdirAll(filepath.Dir(path), 0700)
			raw, _ := json.Marshal(LaunchConfig{Connection: p, Token: "private", URL: "http://localhost/mcp"})
			os.WriteFile(path, raw, 0600)
			env := map[string]string{"HOME": home, "PICODE_DATA": data, "PICODE_TERM_ID": "current-terminal", "CODEX_THREAD_ID": "current-thread"}
			for k, v := range tc.env {
				env[k] = v
			}
			_, err := ResolveCLIConnection("", func(k string) string { return env[k] })
			if (err == nil) != tc.want {
				t.Fatalf("accepted=%v, want %v: %v", err == nil, tc.want, err)
			}
		})
	}
}

func TestOtherNativeCLIIgnoresAmbientCodexParent(t *testing.T) {
	_, p, _, _, _ := fixture(t)
	for _, cli := range []string{"grok", "hermes"} {
		t.Run(cli, func(t *testing.T) {
			dir := t.TempDir()
			p.CLI = cli
			p.SessionKey = "native-child"
			path := filepath.Join(dir, "peer_one", "connection.json")
			os.MkdirAll(filepath.Dir(path), 0700)
			raw, _ := json.Marshal(LaunchConfig{Connection: p, Token: "private", URL: "http://localhost/mcp"})
			os.WriteFile(path, raw, 0600)
			env := map[string]string{"PICODE_MESSAGES_DIR": dir, "CODEX_THREAD_ID": "parent", "CODEX_SESSION_ID": "parent", "PICODE_TERM_ID": "child", strings.ToUpper(cli) + "_SESSION_ID": "native-child"}
			if _, err := ResolveCLIConnection("", func(k string) string { return env[k] }); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCLIRefusesInvalidInputBeforeConnecting(t *testing.T) {
	for _, args := range [][]string{{"send"}, {"send", "--to", "p", "--request-id", "r", "--body", strings.Repeat("x", (16<<10)+1)}, {"read", "--limit", "101"}, {"ack"}, {"contacts", "unexpected"}, {"unknown"}} {
		var out bytes.Buffer
		if err := RunCLI(context.Background(), args, strings.NewReader(""), &out, &out, func(string) string { t.Fatal("resolved identity before validating input"); return "" }); err == nil {
			t.Fatal(args)
		}
	}
}

func TestCLIRefusesRedirectAndUntrustedTLS(t *testing.T) {
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusTemporaryRedirect))
	defer redirect.Close()
	for _, endpoint := range []string{redirect.URL, "http://example.org/mcp", "https://user:secret@example.org/mcp"} {
		_, err := Call(context.Background(), LaunchConfig{URL: endpoint, Token: "private-fixture"}, "list_contacts", emptyInput{})
		if err == nil || strings.Contains(err.Error(), "private-fixture") {
			t.Fatal(err)
		}
	}
	if reached {
		t.Fatal("redirect leaked capability")
	}
	s, _, token, _, _ := fixture(t)
	tlsServer := httptest.NewTLSServer(Handler(s))
	defer tlsServer.Close()
	if _, err := Call(context.Background(), LaunchConfig{URL: tlsServer.URL, Token: token}, "list_contacts", emptyInput{}); err == nil {
		t.Fatal("accepted untrusted TLS")
	}
}
