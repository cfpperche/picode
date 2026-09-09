package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeInstallerPreservesFilesAndUsesNativeProfile(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	for _, cli := range []string{"grok", "hermes"} {
		t.Run(cli, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("HOME", root)
			t.Setenv("GROK_HOME", "")
			t.Setenv("HERMES_HOME", "")
			data := filepath.Join(root, "data")
			if err := writeNativeAssets(data); err != nil {
				t.Fatal(err)
			}
			native := filepath.Join(root, "fake-hermes")
			script := `#!/bin/sh
case "$*" in
 '--profile selected config path') printf '%s\n' "$HOME/vendor-selected/config.yaml" ;;
 '--profile selected plugins enable picode-native --no-allow-tool-override') printf '%s\n' "$*" > "$HOME/enabled" ;;
 *) exit 2 ;;
esac
`
			if err := os.WriteFile(native, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			base := filepath.Join(root, ".grok")
			target := "hooks/picode-native.json"
			if cli == "hermes" {
				base = filepath.Join(root, "vendor-selected")
				target = "plugins/picode-native/__init__.py"
			}
			os.MkdirAll(base, 0700)
			sentinel := filepath.Join(base, "config.yaml")
			os.WriteFile(sentinel, []byte("user: preserved\n"), 0600)
			run := func() ([]byte, error) {
				return exec.Command("python3", filepath.Join(nativeAssetsDir(data), "install.py"), cli, nativeAssetsDir(data), native, "chat", "--profile", "selected").CombinedOutput()
			}
			if out, err := run(); err != nil {
				t.Fatalf("install: %v %s", err, out)
			}
			if out, err := run(); err != nil {
				t.Fatalf("idempotent install: %v %s", err, out)
			}
			if raw, _ := os.ReadFile(sentinel); string(raw) != "user: preserved\n" {
				t.Fatal("native settings overwritten")
			}
			if cli == "hermes" {
				if _, err := os.Stat(filepath.Join(root, ".hermes", "plugins")); !os.IsNotExist(err) {
					t.Fatal("installed into wrong profile")
				}
			}
			path := filepath.Join(base, target)
			os.WriteFile(path, []byte("owner edit"), 0600)
			if out, err := run(); err == nil || !strings.Contains(string(out), "was changed") {
				t.Fatalf("modified integration clobbered: %v %s", err, out)
			}
			if raw, _ := os.ReadFile(path); string(raw) != "owner edit" {
				t.Fatal("lost owner edit")
			}
			os.Remove(path)
			os.Symlink(sentinel, path)
			if out, err := run(); err == nil || !strings.Contains(string(out), "symlink") {
				t.Fatalf("followed integration symlink: %v %s", err, out)
			}
		})
	}
}
