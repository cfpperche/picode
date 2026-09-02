package provision

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/install"
)

func TestMemberStepsShape(t *testing.T) {
	want := []string{"account", "linger", "binary", "member-env", "member-service", "member-health"}
	steps := MemberSteps()
	if len(steps) != len(want) {
		t.Fatalf("%d steps", len(steps))
	}
	for i, id := range want {
		if steps[i].ID != id || steps[i].Scope != ScopeRoot {
			t.Errorf("step %d = %q/%s", i, steps[i].ID, steps[i].Scope)
		}
	}
}

func TestAccountStep(t *testing.T) {
	old := lookupAccount
	t.Cleanup(func() { lookupAccount = old })
	lookupAccount = func(string) (*user.User, error) { return nil, errors.New("no") }
	if got := accountStep().Check(Env{User: "alice"}); got.Status != StatusFix {
		t.Fatalf("%+v", got)
	}
	oldRun := run
	t.Cleanup(func() { run = oldRun })
	var got []string
	run = func(name string, args ...string) error { got = append([]string{name}, args...); return nil }
	if err := accountStep().Fix(Env{User: "alice"}); err != nil || strings.Join(got, " ") != "useradd -m -s /bin/bash alice" {
		t.Fatalf("%v %v", got, err)
	}
	lookupAccount = func(string) (*user.User, error) { return &user.User{Username: "alice"}, nil }
	if got := accountStep().Check(Env{User: "alice"}); got.Status != StatusOK {
		t.Fatalf("%+v", got)
	}
}

func TestMemberEnvStep(t *testing.T) {
	old := gatewayHostname
	t.Cleanup(func() { gatewayHostname = old })
	gatewayHostname = func() (string, error) { return "", errors.New("no config") }
	home := t.TempDir()
	env := Env{User: "alice", Home: home}
	if got := memberEnvStep().Check(env); got.Status != StatusBlocked || !strings.Contains(got.Detail, "gateway install") {
		t.Fatalf("%+v", got)
	}
	gatewayHostname = func() (string, error) { return "box.tail1234.ts.net", nil }
	if got := memberEnvStep().Check(env); got.Status != StatusFix {
		t.Fatalf("%+v", got)
	}
	oldRun := run
	t.Cleanup(func() { run = oldRun })
	run = func(name string, args ...string) error { return nil } // chown
	if err := memberEnvStep().Fix(env); err != nil {
		t.Fatal(err)
	}
	have, _ := install.ReadEnvDropIn(home)
	if have["PICODE_PUBLIC_URL"] != "https://box.tail1234.ts.net" || have["PICODE_AUTH_MODE"] != "all" || have["PICODE_HOST"] != "127.0.0.1" || have["PICODE_INSECURE"] != "1" {
		t.Fatalf("%v", have)
	}
	if got := memberEnvStep().Check(env); got.Status != StatusOK {
		t.Fatalf("after fix: %+v", got)
	}
}

func TestMemberServiceAndHealthReadTheDaemon(t *testing.T) {
	home := t.TempDir()
	env := Env{User: "alice", Home: home}
	if got := memberServiceStep().Check(env); got.Status != StatusFix {
		t.Fatalf("no server.json: %+v", got)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	t.Cleanup(ts.Close)
	_ = os.MkdirAll(filepath.Join(home, ".picode"), 0o755)
	_ = os.WriteFile(filepath.Join(home, ".picode", "server.json"), []byte(`{"url":"`+ts.URL+`"}`), 0o644)
	if got := memberServiceStep().Check(env); got.Status != StatusOK {
		t.Fatalf("running: %+v", got)
	}
	if got := memberHealthStep().Check(env); got.Status != StatusOK {
		t.Fatalf("health: %+v", got)
	}
	_ = os.WriteFile(filepath.Join(home, ".picode", "server.json"), []byte(`{"url":"http://0.0.0.0:8445"}`), 0o644)
	if got := memberHealthStep().Check(env); got.Status != StatusBlocked || !strings.Contains(got.Detail, "loopback") {
		t.Fatalf("network-facing member must be refused: %+v", got)
	}
}
