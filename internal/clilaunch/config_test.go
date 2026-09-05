package clilaunch

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestResolveLaunchDecisionTable(t *testing.T) {
	base := Config{Executable: "/bin/pi", Args: []string{"base"}, Env: map[string]string{"KEEP": "a", "CHANGE": "b", "DROP": "c"}, Path: []string{"/base"}, Integration: true}
	argEmpty := []string{}
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
		{"reserved path", Config{Env: map[string]string{"PATH": "/other"}}, false},
		{"relative path", Config{Path: []string{"relative"}}, false},
		{"path separator", Config{Path: []string{"/a:/b"}}, false},
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
