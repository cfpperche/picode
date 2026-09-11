package communication

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type tokenTransport struct{ token string }

func (t tokenTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(r)
}
func fixture(t *testing.T) (*store.Store, store.PeerConnection, string, store.PeerConnection, string) {
	t.Helper()
	dir := t.TempDir()
	s, e := store.Open(filepath.Join(dir, "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	w, e := s.AddWorkspace("Owned fixture", dir)
	if e != nil {
		t.Fatal(e)
	}
	a, e := s.AddAgent(w.ID, "Managed Pi", "")
	if e != nil {
		t.Fatal(e)
	}
	session := filepath.Join(dir, "pi-session.jsonl")
	if _, e = s.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &session}); e != nil {
		t.Fatal(e)
	}
	terminal, e := s.CreateTerminalIn(w.ID, "Codex", dir)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetTerminalLaunch(terminal.ID, "codex", clilaunch.Overrides{}); e != nil {
		t.Fatal(e)
	}
	if e = s.SetTerminalLastSession(terminal.ID, store.TerminalLastSession{CLI: "codex", SessionID: "native-codex"}); e != nil {
		t.Fatal(e)
	}
	p, pt, e := s.EnablePeer("agent", a.ID, session)
	if e != nil {
		t.Fatal(e)
	}
	q, qt, e := s.EnablePeer("terminal", terminal.ID, "native-codex")
	if e != nil {
		t.Fatal(e)
	}
	return s, p, pt, q, qt
}
func client(t *testing.T, ctx context.Context, url, token string) *mcp.ClientSession {
	t.Helper()
	c := mcp.NewClient(&mcp.Implementation{Name: "owned-test", Version: "1"}, nil)
	session, e := c.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url, HTTPClient: &http.Client{Transport: tokenTransport{token}}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { session.Close() })
	return session
}
func invoke[T any](t *testing.T, ctx context.Context, c *mcp.ClientSession, name string, args any) T {
	t.Helper()
	r, e := c.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if e != nil {
		t.Fatal(e)
	}
	if r.IsError {
		t.Fatalf("%s: %+v", name, r.Content)
	}
	b, e := json.Marshal(r.StructuredContent)
	if e != nil {
		t.Fatal(e)
	}
	var out T
	if e = json.Unmarshal(b, &out); e != nil {
		t.Fatal(e)
	}
	return out
}
func TestHTTPMCPManagedAndTerminal(t *testing.T) {
	s, p, pt, q, qt := fixture(t)
	a, e := auth.New(auth.Config{Store: s, DataDir: t.TempDir(), Insecure: true})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetSetting(auth.ModeSettingKey, auth.ModeAll); e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(a.Wrap(Handler(s)))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pc := client(t, ctx, server.URL+Path, pt)
	qc := client(t, ctx, server.URL+Path, qt)
	tools, e := pc.ListTools(ctx, nil)
	if e != nil || len(tools.Tools) != 4 {
		t.Fatalf("tools %v %v", tools, e)
	}
	contacts := invoke[contactsOutput](t, ctx, pc, "list_contacts", map[string]any{})
	if len(contacts.Contacts) != 1 || contacts.Contacts[0].ID != q.ID {
		t.Fatal(contacts)
	}
	input := map[string]any{"to": q.ID, "request_id": "one", "body": "review this"}
	sent := invoke[sendOutput](t, ctx, pc, "send_message", input)
	again := invoke[sendOutput](t, ctx, pc, "send_message", input)
	if sent.Status != "accepted" || sent.Message.ID != again.Message.ID {
		t.Fatal("retry")
	}
	inbox := invoke[readOutput](t, ctx, qc, "read_messages", map[string]any{})
	if len(inbox.Messages) != 1 || inbox.Messages[0].Body != "review this" {
		t.Fatal(inbox)
	}
	invoke[ackOutput](t, ctx, qc, "ack_messages", map[string]any{"ids": []string{sent.Message.ID}})
	empty := invoke[readOutput](t, ctx, qc, "read_messages", map[string]any{})
	if len(empty.Messages) != 0 {
		t.Fatal("ack")
	}
	reply := invoke[sendOutput](t, ctx, qc, "send_message", map[string]any{"to": p.ID, "request_id": "answer", "body": "received", "reply_to": sent.Message.ID})
	if reply.Message.ReplyTo != sent.Message.ID {
		t.Fatal(reply)
	}
	back := invoke[readOutput](t, ctx, pc, "read_messages", map[string]any{})
	if len(back.Messages) != 1 || back.Messages[0].ID != reply.Message.ID {
		t.Fatal(back)
	}
	if e = s.RevokePeer(q.ID); e != nil {
		t.Fatal(e)
	}
	result, e := qc.CallTool(ctx, &mcp.CallToolParams{Name: "read_messages", Arguments: map[string]any{}})
	if e == nil && !result.IsError {
		t.Fatal("established client retained revoked capability")
	}
}
func TestHTTPMCPGate(t *testing.T) {
	s, _, token, _, _ := fixture(t)
	for _, mode := range []string{auth.ModeOff, auth.ModeRemote, auth.ModeAll} {
		t.Run(mode, func(t *testing.T) {
			if e := s.SetSetting(auth.ModeSettingKey, mode); e != nil {
				t.Fatal(e)
			}
			a, e := auth.New(auth.Config{Store: s, DataDir: t.TempDir(), Insecure: true})
			if e != nil {
				t.Fatal(e)
			}
			h := a.Wrap(Handler(s))
			cases := []struct {
				name, token, origin, host, body string
				want                            int
			}{
				{name: "anonymous", host: "localhost", want: 401},
				{name: "invalid", token: "wrong", host: "localhost", want: 401},
				{name: "cross origin", token: token, host: "localhost", origin: "https://evil.test", want: 403},
				{name: "null origin", token: token, host: "localhost", origin: "null", want: 403},
				{name: "rebind", token: token, host: "evil.test", want: 403},
				{name: "oversize", token: token, host: "localhost", body: strings.Repeat(" ", 65537), want: 413},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					r := httptest.NewRequest("POST", "http://"+tc.host+Path, strings.NewReader(tc.body))
					r.RemoteAddr = "127.0.0.1:1234"
					r.Header.Set("Content-Type", "application/json")
					r.Header.Set("Accept", "application/json, text/event-stream")
					if tc.token != "" {
						r.Header.Set("Authorization", "Bearer "+tc.token)
					}
					if tc.origin != "" {
						r.Header.Set("Origin", tc.origin)
					}
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if w.Code != tc.want {
						t.Fatalf("status %d: %s", w.Code, w.Body.String())
					}
				})
			}
		})
	}
}

// ADR-0116's last row: **the MCP surface gains no verb.** An edge is the
// owner's grant, drawn in the browser through the authenticated owner API;
// a capability whose beneficiary can grant it to itself is not a
// capability. This test is what stops a later session from adding a
// convenient link_sessions tool: the list is exactly ADR-0104's four verbs,
// and no tool — nor the CLI that fronts them — so much as names an edge, a
// canvas or a transcript. The word "canvas" joined the list with ADR-0118's
// rename, so the mechanism cannot leak through the new noun either.
func TestMCPSurfaceGainsNoEdgeVerb(t *testing.T) {
	s, _, pt, _, _ := fixture(t)
	a, e := auth.New(auth.Config{Store: s, DataDir: t.TempDir(), Insecure: true})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetSetting(auth.ModeSettingKey, auth.ModeAll); e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(a.Wrap(Handler(s)))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tools, err := client(t, ctx, server.URL+Path, pt).ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"list_contacts": true, "send_message": true, "read_messages": true, "ack_messages": true}
	if len(tools.Tools) != len(want) {
		t.Fatalf("tools = %d, want %d", len(tools.Tools), len(want))
	}
	// Whole words, so "acknowledge" is not an edge.
	forbidden := regexp.MustCompile(`(?i)\b(edges?|canvas|canvases|matrix|matrices|links?|linked|transcripts?|scrollback)\b`)
	for _, tool := range tools.Tools {
		if !want[tool.Name] {
			t.Fatalf("unknown tool %q: an edge is the owner's grant, not an agent's", tool.Name)
		}
		if hit := forbidden.FindString(tool.Name + " " + tool.Description); hit != "" {
			t.Fatalf("tool %s mentions %q: %s", tool.Name, hit, tool.Description)
		}
	}
	// The shell client fronts the same four verbs and nothing else; its help
	// is what an agent reads to find out what it may do.
	if hit := forbidden.FindString(MessagesHelp); hit != "" {
		t.Fatalf("picode messages help mentions %q", hit)
	}
	if !strings.Contains(MessagesHelp, "contacts|send|read|ack") {
		t.Fatalf("picode messages help changed shape: %s", MessagesHelp)
	}
}
