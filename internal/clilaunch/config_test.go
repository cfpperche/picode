package clilaunch

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestResolveLaunchDecisionTable(t *testing.T) {
	base := Config{Executable: "/bin/pi", Args: []string{"base"}, Env: map[string]string{"KEEP": "a", "CHANGE": "b", "DROP": "c"}, Path: []string{"/base"}, Integration: true, Tools: []string{"computer"}}
	argEmpty := []string{}
	tools := []string{"browser", "computer"}
	path := []string{"/override"}
	exe := "/other/pi"
	value := "new"
	off := false
	for _, tc := range []struct {
		name     string
		override Overrides
		check    func(Config) bool
	}{
		{"inherit", Overrides{}, func(c Config) bool { return reflect.DeepEqual(c, base) }},
		{"empty args", Overrides{Args: &argEmpty}, func(c Config) bool { return len(c.Args) == 0 }},
		{"path override", Overrides{Path: &path}, func(c Config) bool { return reflect.DeepEqual(c.Path, path) }},
		{"explicit executable", Overrides{Executable: &exe}, func(c Config) bool { return c.Executable == exe }},
		{"turn off", Overrides{Integration: &off}, func(c Config) bool { return !c.Integration }},
		{"environment merge and deletion", Overrides{Env: map[string]*string{"CHANGE": &value, "DROP": nil}}, func(c Config) bool { return reflect.DeepEqual(c.Env, map[string]string{"KEEP": "a", "CHANGE": "new"}) }},
		{"tools inherit", Overrides{}, func(c Config) bool { return reflect.DeepEqual(c.Tools, []string{"computer"}) }},
		{"tools override", Overrides{Tools: &tools}, func(c Config) bool { return reflect.DeepEqual(c.Tools, []string{"browser", "computer"}) }},
		{"tools cleared", Overrides{Tools: &argEmpty}, func(c Config) bool { return len(c.Tools) == 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Resolve(base, tc.override)
			if !tc.check(c) {
				t.Fatalf("resolution: %+v", c)
			}
			c.Env["KEEP"] = "mutated"
			if base.Env["KEEP"] != "a" {
				t.Fatal("mutated shared defaults")
			}
		})
	}
}

func TestValidateLaunchDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config Config
		valid  bool
	}{
		{"ordinary", Config{Args: []string{"a b", "", "$(literal)"}, Env: map[string]string{"MODEL": "small"}, Path: []string{"/tools with spaces"}}, true},
		{"nul argument", Config{Args: []string{"a\x00b"}}, false},
		{"newline env", Config{Env: map[string]string{"X": "a\nb"}}, false},
		{"invalid env name", Config{Env: map[string]string{"X-Y": "a"}}, false},
		{"reserved correlation", Config{Env: map[string]string{"PICODE_TERM_ID": "other"}}, false},
		{"reserved home", Config{Env: map[string]string{"HOME": "/other"}}, false},
		{"reserved grok home", Config{Env: map[string]string{"GROK_HOME": "/other"}}, false},
		{"reserved hermes home", Config{Env: map[string]string{"HERMES_HOME": "/other"}}, false},
		{"reserved opencode config", Config{Env: map[string]string{"OPENCODE_CONFIG": "/other.json"}}, false},
		{"reserved omp agent dir", Config{Env: map[string]string{"PI_CODING_AGENT_DIR": "/home/x/.omp"}}, false},
		{"reserved path", Config{Env: map[string]string{"PATH": "/other"}}, false},
		{"relative path", Config{Path: []string{"relative"}}, false},
		{"path separator", Config{Path: []string{"/a:/b"}}, false},
		{"tools named", Config{Tools: []string{"computer", "browser"}}, true},
		{"tool name shape", Config{Tools: []string{"Computer Use"}}, false},
		{"too many tools", Config{Tools: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.config); (got == nil) != tc.valid {
				t.Fatalf("validation: %v", got)
			}
		})
	}
}

func TestLaunchDiagnosticsRedactValues(t *testing.T) {
	c := Config{Args: []string{"--api-key", "secret-one", "--token=secret-two", "hello"}, Env: map[string]string{"MY_KEY": "secret-three"}}
	d := Describe(c, "/bin/cli", "now")
	raw, _ := json.Marshal(d)
	for _, s := range []string{"secret-one", "secret-two", "secret-three"} {
		if strings.Contains(string(raw), s) {
			t.Fatalf("leaked %s", s)
		}
	}
	if !strings.Contains(string(raw), "MY_KEY") || !strings.Contains(string(raw), "hello") {
		t.Fatal("diagnostics missing")
	}
	if Fingerprint(c) != Fingerprint(Resolve(c, Overrides{})) {
		t.Fatal("unstable normalization")
	}
}

func TestCatalogCapabilities(t *testing.T) {
	// Decision table: every catalog row opens a terminal and carries an
	// adapter — Omp joined the full rows in the omp-adapter slice (it left
	// the terminal-only group the omp-cli slice created). The `terminal`
	// set stays as the shape a future CLI without an adapter rejoins.
	terminal := map[string]bool{}
	seen := map[string]bool{}
	for _, c := range Catalog() {
		if c.Launchable() != true {
			t.Errorf("%s Launchable = false, every catalog row opens a terminal", c.ID)
		}
		if c.Integrable() == terminal[c.ID] {
			t.Errorf("%s Integrable = %v", c.ID, c.Integrable())
		}
		if c.Surface == SurfaceDetect && c.Launchable() {
			t.Errorf("%s surface=detect must not launch", c.ID)
		}
		if terminal[c.ID] {
			seen[c.ID] = true
			if c.Surface != SurfaceTerminal {
				t.Errorf("%s surface = %q, want %q", c.ID, c.Surface, SurfaceTerminal)
			}
			if c.Command == "" || c.Docs == "" {
				t.Errorf("%s missing command or docs", c.ID)
			}
		}
	}
	for id := range terminal {
		if !seen[id] {
			t.Errorf("catalog missing %s", id)
		}
	}
}
