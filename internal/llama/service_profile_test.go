package llama

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func testExecutionProfile(t *testing.T) ExecutionProfile {
	t.Helper()
	dir := t.TempDir()
	// macOS keeps TempDir under /var → /private/var, and the binary guard
	// rightly refuses a symlinked parent; a real install path is already
	// resolved, so resolve this one the same way.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	binary := filepath.Join(dir, "llama-server")
	data := []byte("fixture, never executed")
	if err := os.WriteFile(binary, data, 0700); err != nil {
		t.Fatal(err)
	}
	return ExecutionProfile{Binary: binary, BinarySHA256: fmt.Sprintf("%x", sha256.Sum256(data)), ModelsDir: dir, Host: "127.0.0.1", Port: 8080, Context: 8192, Threads: 4}
}

func TestExecutionProfileArgs(t *testing.T) {
	for _, gpu := range []int{0, 12} {
		for _, jinja := range []bool{false, true} {
			for _, noAuto := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%t/%t", gpu, jinja, noAuto), func(t *testing.T) {
					p := testExecutionProfile(t)
					p.GPULayers, p.Jinja, p.NoAutoLoad = gpu, jinja, noAuto
					jf, af := "--no-jinja", "--models-autoload"
					if jinja {
						jf = "--jinja"
					}
					if noAuto {
						af = "--no-models-autoload"
					}
					want := []string{p.Binary, "--models-dir", p.ModelsDir, "--host", p.Host, "--port", "8080", "--ctx-size", "8192", "--threads", "4", "--threads-batch", "4", "--n-gpu-layers", strconv.Itoa(gpu), jf, af}
					got, err := p.Args()
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("got %q; want %q", got, want)
					}
				})
			}
		}
	}
}

func TestExecutionProfileValidation(t *testing.T) {
	for _, field := range []string{"Port", "Context", "Threads", "GPULayers"} {
		bounds := map[string][]int{"Port": {1023, 1024, 65535, 65536}, "Context": {511, 512, 131072, 131073}, "Threads": {0, 1, 4, 5}, "GPULayers": {-1, 0, 12, -2}}[field]
		for i, value := range bounds {
			t.Run(fmt.Sprintf("%s/%d", field, value), func(t *testing.T) {
				p := testExecutionProfile(t)
				reflect.ValueOf(&p).Elem().FieldByName(field).SetInt(int64(value))
				if err := p.Validate(); (err == nil) != (i == 1 || i == 2) {
					t.Fatalf("unexpected validation: %v", err)
				}
			})
		}
	}
	for _, tt := range []struct {
		field, value string
		valid        bool
	}{
		{"Binary", "llama-server", false}, {"Binary", "/tmp/other", false},
		{"BinarySHA256", "", false}, {"BinarySHA256", "zz", false}, {"BinarySHA256", "ab", false},
		{"ModelsDir", "", false}, {"ModelsDir", "relative", false},
		{"Host", "0.0.0.0", false}, {"Host", "::1", true},
	} {
		t.Run(tt.field+"/"+tt.value, func(t *testing.T) {
			p := testExecutionProfile(t)
			reflect.ValueOf(&p).Elem().FieldByName(tt.field).SetString(tt.value)
			if err := p.Validate(); (err == nil) != tt.valid {
				t.Fatalf("unexpected validation: %v", err)
			}
		})
	}
}

func TestExecutionProfileBinaryGuards(t *testing.T) {
	for _, scenario := range []string{"missing", "directory", "permissions", "tampered", "symlink", "parent symlink"} {
		t.Run(scenario, func(t *testing.T) {
			p := testExecutionProfile(t)
			if _, err := p.Args(); err != nil {
				t.Fatal(err)
			}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch scenario {
			case "missing":
				must(os.Remove(p.Binary))
			case "directory":
				must(os.Remove(p.Binary))
				must(os.Mkdir(p.Binary, 0700))
			case "permissions":
				must(os.Chmod(p.Binary, 0600))
			case "tampered":
				must(os.WriteFile(p.Binary, []byte("changed"), 0700))
			case "symlink":
				target := p.Binary + ".real"
				must(os.Rename(p.Binary, target))
				must(os.Symlink(target, p.Binary))
			case "parent symlink":
				link := filepath.Join(t.TempDir(), "linked")
				must(os.Symlink(filepath.Dir(p.Binary), link))
				p.Binary = filepath.Join(link, "llama-server")
			}
			if args, err := p.Args(); err == nil || args != nil {
				t.Fatalf("unsafe preview: %q, %v", args, err)
			}
		})
	}
}
