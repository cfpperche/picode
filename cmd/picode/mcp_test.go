package main

import (
	"bytes"
	"strings"
	"testing"
)

// `picode mcp` must be a command the dispatcher owns: the binary's default
// is the server on the production data dir, and a fall-through would start
// a second daemon under a CLI's nose (ADR-0154).
func TestDispatchClaimsMCP(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := mcpMain(nil, strings.NewReader(""), &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage: picode mcp") {
		t.Fatalf("no family: code %d, stderr %q", code, errOut.String())
	}
	errOut.Reset()
	if code := mcpMain([]string{"flying"}, strings.NewReader(""), &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "unknown family") {
		t.Fatalf("unknown family: code %d, stderr %q", code, errOut.String())
	}
	out.Reset()
	t.Setenv("PICODE_DATA", t.TempDir()) // no server.json: the server still starts and answers in words
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	if code := mcpMain([]string{"computer", "browser", "computer"}, in, &out, &errOut); code != 0 {
		t.Fatalf("serve: code %d, stderr %q", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"name":"picode"`) || !strings.Contains(lines[1], `"name":"browser"`) || !strings.Contains(lines[1], `"name":"computer"`) {
		t.Fatalf("out =\n%s", out.String())
	}
	if strings.Count(lines[1], `"name":"computer"`) != 1 {
		t.Fatal("a family named twice is served once")
	}
}
