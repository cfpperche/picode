package mcptool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/version"
)

// ADR-0216: every call names this build, and a 426 reaches the model as the
// daemon's own sentence.
func TestHTTPDaemonSendsTheClientHandshake(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(version.ClientHeader)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUpgradeRequired)
		_, _ = w.Write([]byte(`{"error":"PiCode was updated and this agent's PiCode tools are older than it. Restart the agent to load the new tools."}`))
	}))
	defer srv.Close()
	status, body, err := NewHTTPDaemon(srv.URL, nil).Get(context.Background(), "/api/version")
	if err != nil {
		t.Fatal(err)
	}
	c, ok := version.ParseClient(got)
	if !ok || c.Kind != "picode-mcp" || c.Protocol != version.APIProtocol {
		t.Fatalf("header %q parsed as %+v %v", got, c, ok)
	}
	if _, err := ParseAnswer(status, body, "action"); err == nil || !strings.Contains(err.Error(), "Restart the agent") {
		t.Fatalf("426 must surface the daemon's message, got %v", err)
	}
}
