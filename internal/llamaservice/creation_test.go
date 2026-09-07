package llamaservice

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestCreationFailureMatrix(t *testing.T) {
	for _, scenario := range []string{"database-intent", "database-final", "interrupted", "unexpected-file", "permissions", "existing-empty", "existing-file"} {
		t.Run(scenario, func(t *testing.T) {
			if runtime.GOOS != "linux" {
				t.Skip("Linux-only creation")
			}
			s := testService(t)
			c := Config{Port: 18080, Context: 4096, Threads: 2, Jinja: true}
			var stage string
			switch scenario {
			case "database-intent":
				_ = s.st.Close()
			case "existing-empty", "existing-file":
				if err := os.Mkdir(s.root, 0700); err != nil {
					t.Fatal(err)
				}
				if scenario == "existing-file" {
					if err := os.WriteFile(filepath.Join(s.root, "keep"), []byte("keep"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			default:
				s.st.OnEvent = func(e store.Event) {
					if stage != "" {
						return
					}
					raw, _, err := s.st.LlamaService()
					if err != nil {
						t.Fatal(err)
					}
					var d Document
					_ = json.Unmarshal(raw, &d)
					if d.Creation == nil {
						return
					}
					stage = filepath.Join(filepath.Dir(s.root), d.Creation.Stage)
					switch scenario {
					case "database-final":
						_ = s.st.Close()
					case "interrupted":
						s.cancel()
					case "unexpected-file":
						if err := os.WriteFile(filepath.Join(stage, "keep"), []byte("keep"), 0600); err != nil {
							t.Fatal(err)
						}
						_ = s.st.Close()
					case "permissions":
						if os.Geteuid() == 0 {
							t.Skip("root bypasses directory permissions")
						}
						if err := os.Chmod(stage, 0500); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			if _, err := s.Configure(c, s.rev); err == nil {
				t.Fatal("failure accepted")
			}
			if scenario == "existing-empty" || scenario == "existing-file" || scenario == "unexpected-file" {
				if _, err := os.Stat(s.root); err != nil {
					t.Fatal("folder not preserved", err)
				}
				if scenario != "existing-empty" {
					if b, err := os.ReadFile(filepath.Join(s.root, "keep")); err != nil || string(b) != "keep" {
						t.Fatal("unknown file removed", err)
					}
				}
				return
			}
			if scenario == "permissions" {
				if err := os.Chmod(stage, 0700); err != nil {
					t.Fatal(err)
				}
			}
			_ = s.st.Close()
			st, err := store.Open(filepath.Join(filepath.Dir(s.root), "db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			recovered, err := New(st, filepath.Dir(s.root), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer recovered.Close()
			if _, err = os.Stat(s.root); !os.IsNotExist(err) {
				t.Fatal("empty failed setup survived recovery", err)
			}
			if _, err = recovered.Configure(c, recovered.rev); err != nil {
				t.Fatal("retry failed", err)
			}
		})
	}
}

func TestCreationCrashHelper(t *testing.T) {
	dir := os.Getenv("PICODE_TEST_CREATION_CRASH")
	if dir == "" {
		return
	}
	st, err := store.Open(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(st, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	st.OnEvent = func(e store.Event) { os.Exit(23) }
	_, _ = s.Configure(Config{Port: 18080, Context: 4096, Threads: 2, Jinja: true}, 0)
	t.Fatal("crash hook did not run")
}

func TestCreationCrashRecovery(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only creation")
	}
	for _, phase := range []string{"staged", "published", "unknown-root"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			cmd := exec.Command(os.Args[0], "-test.run=^TestCreationCrashHelper$")
			cmd.Env = append(os.Environ(), "PICODE_TEST_CREATION_CRASH="+dir)
			if err := cmd.Run(); err == nil || cmd.ProcessState.ExitCode() != 23 {
				t.Fatal("expected abrupt exit", err)
			}
			st, err := store.Open(filepath.Join(dir, "db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			raw, _, _ := st.LlamaService()
			var d Document
			_ = json.Unmarshal(raw, &d)
			if d.Creation == nil {
				t.Fatal("intent not persisted")
			}
			stage := filepath.Join(dir, d.Creation.Stage)
			root := filepath.Join(dir, "llama-owned")
			if phase == "published" {
				if err = publishCreation(stage, root); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "unknown-root" {
				if err = os.Mkdir(root, 0700); err != nil {
					t.Fatal(err)
				}
				if err = publishCreation(stage, root); err == nil {
					t.Fatal("existing empty folder overwritten")
				}
			}
			s, err := New(st, dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if s.doc.Creation != nil || s.doc.Created {
				t.Fatal("incorrect recovered state")
			}
			_, err = os.Stat(root)
			if phase == "unknown-root" {
				if err != nil {
					t.Fatal("unknown folder removed")
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("owned empty folder retained")
			}
		})
	}
}
