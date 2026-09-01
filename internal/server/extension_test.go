package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDecideExtensionSend(t *testing.T) {
	tests := []struct {
		name   string
		in     extensionSendInput
		status int
		start  bool
		err    string
	}{
		{
			name:   "empty",
			in:     extensionSendInput{},
			status: http.StatusBadRequest,
			err:    "message or page is required",
		},
		{
			name:   "chrome page",
			in:     extensionSendInput{HasMessage: true, TabURL: "chrome://settings"},
			status: http.StatusBadRequest,
			err:    "This page can't be sent.",
		},
		{
			name:   "file page",
			in:     extensionSendInput{TabURL: "file:///etc/passwd"},
			status: http.StatusBadRequest,
			err:    "This page can't be sent.",
		},
		{
			name:   "extension page",
			in:     extensionSendInput{TabURL: "chrome-extension://abc/sidepanel.html"},
			status: http.StatusBadRequest,
			err:    "This page can't be sent.",
		},
		{
			name: "interactive",
			in: extensionSendInput{
				Mode: modeInteractive, HasMessage: true, TabURL: "https://example.com",
			},
			status: http.StatusConflict,
			err:    "This agent is in the terminal.",
		},
		{
			name:  "stopped starts",
			in:    extensionSendInput{Mode: modeStopped, HasMessage: true, TabURL: "https://example.com"},
			start: true,
		},
		{
			name: "managed idle",
			in:   extensionSendInput{Mode: modeManaged, HasMessage: true, TabURL: "https://example.com"},
		},
		{
			name: "page only",
			in:   extensionSendInput{Mode: modeManaged, TabURL: "http://localhost:3000/app"},
		},
		{
			name: "message only",
			in:   extensionSendInput{Mode: modeManaged, HasMessage: true},
		},
		{
			name: "image only",
			in:   extensionSendInput{Mode: modeManaged, HasImage: true},
		},
		{
			name:  "stopped page only",
			in:    extensionSendInput{Mode: modeStopped, TabURL: "https://linear.app/issue/x"},
			start: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideExtensionSend(tt.in)
			if got.Status != tt.status {
				t.Errorf("status %d want %d (%s)", got.Status, tt.status, got.Error)
			}
			if got.Start != tt.start {
				t.Errorf("start %v want %v", got.Start, tt.start)
			}
			if tt.err != "" && got.Error != tt.err {
				t.Errorf("error %q want %q", got.Error, tt.err)
			}
		})
	}
}

func TestCapturableURL(t *testing.T) {
	if !capturableURL("https://github.com/cfpperche/picode") {
		t.Fatal("https")
	}
	if capturableURL("about:blank") || capturableURL("edge://flags") || capturableURL("not a url") {
		t.Fatal("refused")
	}
}

func TestComposeTabPrompt(t *testing.T) {
	got := composeTabPrompt(&extensionTab{
		URL: "https://ex.com/a", Title: "Alpha", Selection: "picked",
	}, "fix this")
	want := "[browser-tab]\nurl: https://ex.com/a\ntitle: Alpha\nselection:\npicked\n\nfix this"
	if got != want {
		t.Fatalf("got %q", got)
	}
	if composeTabPrompt(nil, "hi") != "hi" {
		t.Fatal("message only")
	}
}

func TestExtensionAgentsEmpty(t *testing.T) {
	ts := newTestServer(t, "cat")
	res, err := ts.Client().Get(ts.URL + "/api/extension/agents")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body struct {
		Agents []extensionAgent `json:"agents"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Agents) != 0 {
		t.Fatalf("got %d", len(body.Agents))
	}
}

func TestExtensionAgentsListsWorkspaceAndFree(t *testing.T) {
	ts := newTestServer(t, "cat")
	wk := addWorkspaceWithAgent(t, ts, "Repo", t.TempDir())
	res := postJSON(t, ts, "/api/agents", map[string]string{"name": "loose", "path": t.TempDir()})
	if res.StatusCode != 201 {
		t.Fatalf("free agent %d", res.StatusCode)
	}

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/extension/agents"))
	if got.StatusCode != 200 {
		t.Fatalf("status %d", got.StatusCode)
	}
	var body struct {
		Agents []extensionAgent `json:"agents"`
	}
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Agents) != 2 {
		t.Fatalf("got %+v", body.Agents)
	}
	byName := map[string]extensionAgent{}
	for _, a := range body.Agents {
		byName[a.Name] = a
		if a.Mode != "stopped" {
			t.Errorf("%s mode %s", a.Name, a.Mode)
		}
	}
	if byName["default"].Workspace != wk.Name {
		t.Fatalf("workspace %q", byName["default"].Workspace)
	}
	if byName["loose"].Workspace != "" {
		t.Fatalf("free agent leaked workspace %q", byName["loose"].Workspace)
	}
}

func TestExtensionSendDecisionHTTP(t *testing.T) {
	ts := newTestServer(t, "cat")
	wk := addWorkspaceWithAgent(t, ts, "Repo", t.TempDir())
	id := wk.Agents[0].ID

	// missing agent
	res := postJSON(t, ts, "/api/extension/send", map[string]any{
		"agentId": "nope", "message": "hi",
	})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing = %d", res.StatusCode)
	}

	// empty payload
	res = postJSON(t, ts, "/api/extension/send", map[string]any{"agentId": id})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty = %d", res.StatusCode)
	}

	// chrome://
	res = postJSON(t, ts, "/api/extension/send", map[string]any{
		"agentId": id,
		"tab":     map[string]string{"url": "chrome://extensions"},
	})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("chrome = %d", res.StatusCode)
	}
}
