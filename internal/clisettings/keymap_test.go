package clisettings

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The document primitives exist so the guest key-map engine (ADR-0174) reuses
// this package's splice instead of writing its own. What matters is what the
// four guarantees do to a file a person wrote: the bytes PiCode does not touch
// survive, a shape it cannot rewrite is refused, a file that moved is refused,
// and add-then-remove gives the file back.

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readBack(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A value that sits on its own line is spliced in place: the comment beside it,
// the key order and every key PiCode does not know survive.
func TestDocSetStringsSplicesInPlace(t *testing.T) {
	for _, tc := range []struct {
		name, file, key, want string
		values                []string
	}{
		{"yaml", "# mine\ntui.editor.cursorUp: up # the arrow\napp.exit: ctrl+c\n", "tui.editor.cursorUp", "tui.editor.cursorUp: [\"ctrl+p\", \"alt+k\"] # the arrow", []string{"ctrl+p", "alt+k"}},
		{"json", "{\n  \"app.exit\": \"ctrl+c\",\n  \"app.retry\": [\"f5\"]\n}\n", "app.retry", "\"app.retry\": [\"f5\", \"ctrl+r\"]", []string{"f5", "ctrl+r"}},
		{"jsonc", "{\n  // retry the last turn\n  \"app.retry\": \"f5\"\n}\n", "app.retry", "// retry the last turn", []string{"f5"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, "map."+tc.name, tc.file)
			f := FormatYAML
			if strings.HasPrefix(tc.name, "json") {
				f = FormatJSON
				if tc.name == "jsonc" {
					f = FormatJSONC
				}
			}
			d, err := OpenDoc(path, f)
			if err != nil {
				t.Fatal(err)
			}
			if err := d.SetStrings([]string{tc.key}, tc.values); err != nil {
				t.Fatal(err)
			}
			if err := d.Save(d.Revision()); err != nil {
				t.Fatal(err)
			}
			got := readBack(t, path)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("the file lost what it said:\n%s", got)
			}
			if tc.name == "jsonc" && !strings.Contains(got, "// retry the last turn") {
				t.Fatalf("the comment did not survive:\n%s", got)
			}
			// And the value really is the list, read back through the parser.
			again, err := OpenDoc(path, f)
			if err != nil {
				t.Fatal(err)
			}
			values, found, err := again.Strings(tc.key)
			if err != nil || !found {
				t.Fatalf("reading back: found=%v err=%v", found, err)
			}
			if strings.Join(values, ",") != strings.Join(tc.values, ",") {
				t.Fatalf("read back %v, wrote %v", values, tc.values)
			}
		})
	}
}

// A key the file does not set is inserted where the format expects it — the
// case that creates the map, and the one that adds a second binding.
func TestDocSetStringsInsertsAMissingKey(t *testing.T) {
	t.Run("new file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "agent", "keybindings.yml")
		d, err := OpenDoc(path, FormatYAML)
		if err != nil {
			t.Fatal(err)
		}
		if d.Exists() {
			t.Fatal("a file that is not there is not there")
		}
		if err := d.SetStrings([]string{"app.exit"}, []string{"ctrl+q"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(""); err != nil {
			t.Fatal(err)
		}
		got := readBack(t, path)
		if !strings.Contains(got, `app.exit: ["ctrl+q"]`) {
			t.Fatalf("the created file says:\n%s", got)
		}
	})
	t.Run("into an existing map", func(t *testing.T) {
		path := writeTemp(t, "keybindings.yml", "app.exit: ctrl+q\n")
		d, err := OpenDoc(path, FormatYAML)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetStrings([]string{"tui.input.newLine"}, []string{"shift+enter", "ctrl+j"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(d.Revision()); err != nil {
			t.Fatal(err)
		}
		got := readBack(t, path)
		if !strings.Contains(got, "app.exit: ctrl+q") || !strings.Contains(got, `tui.input.newLine: ["shift+enter", "ctrl+j"]`) {
			t.Fatalf("the map does not hold both:\n%s", got)
		}
	})
}

// Reset is "hand the key back to the CLI's own default", so what it must leave
// behind is a file with no trace of the key — not the value the key had before
// PiCode touched it, which the editor never held. Two cases matter: a key that
// was not there at all (the file must come back byte for byte) and a key the
// file set (no orphan block lines may survive).
func TestDocAddThenRemoveLeavesNoTrace(t *testing.T) {
	t.Run("a key the file did not set comes back byte for byte", func(t *testing.T) {
		path := writeTemp(t, "keybindings.yml", "app.exit: ctrl+q\n")
		before := readBack(t, path)
		d, err := OpenDoc(path, FormatYAML)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetStrings([]string{"tui.input.newLine"}, []string{"shift+enter"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Remove("tui.input.newLine"); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(d.Revision()); err != nil {
			t.Fatal(err)
		}
		if got := readBack(t, path); got != before {
			t.Fatalf("the file changed:\n before: %q\n  after: %q", before, got)
		}
	})
	t.Run("a block list leaves no orphan lines", func(t *testing.T) {
		path := writeTemp(t, "keybindings.yml", "tui.editor.cursorUp:\n  - up\n  - ctrl+p\napp.exit: ctrl+q\n")
		d, err := OpenDoc(path, FormatYAML)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetStrings([]string{"tui.editor.cursorUp"}, []string{"alt+x"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Remove("tui.editor.cursorUp"); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(d.Revision()); err != nil {
			t.Fatal(err)
		}
		if got := readBack(t, path); got != "app.exit: ctrl+q\n" {
			t.Fatalf("removing the key must take its block with it, got:\n%s", got)
		}
	})
	t.Run("a list spliced in place", func(t *testing.T) {
		path := writeTemp(t, "keybindings.json", "{\n  \"app.exit\": [\"ctrl+q\"]\n}\n")
		d, err := OpenDoc(path, FormatJSON)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetStrings([]string{"app.exit"}, []string{"ctrl+q", "ctrl+w"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(d.Revision()); err != nil {
			t.Fatal(err)
		}
		if got := readBack(t, path); !strings.Contains(got, `"app.exit": ["ctrl+q", "ctrl+w"]`) {
			t.Fatalf("the list was not replaced in place:\n%s", got)
		}
	})
}

// A key that holds a table is a file PiCode must not rewrite: the CLI may keep
// something there that is not a chord list, and a silent replacement would be
// the worst outcome of an editor that writes other vendors' files.
func TestDocRefusesAShapeItCannotRewrite(t *testing.T) {
	for _, tc := range []struct {
		name, file, key string
		f               Format
	}{
		{"yaml table", "app.exit:\n  nested: true\n", "app.exit", FormatYAML},
		{"json object", "{\n  \"app.exit\": {\"nested\": true}\n}\n", "app.exit", FormatJSON},
		{"yaml number", "app.exit: 3\n", "app.exit", FormatYAML},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, "map", tc.file)
			d, err := OpenDoc(path, tc.f)
			if err != nil {
				t.Fatal(err)
			}
			err = d.SetStrings([]string{tc.key}, []string{"ctrl+q"})
			if !errors.Is(err, ErrShape) {
				t.Fatalf("writing a table must be refused with ErrShape, got %v", err)
			}
			if _, _, err := d.Strings(tc.key); !errors.Is(err, ErrShape) {
				t.Fatalf("reading it as a chord list must be refused too, got %v", err)
			}
			if err := d.Save(d.Revision()); err != nil {
				t.Fatal(err)
			}
			if readBack(t, path) != tc.file {
				t.Fatal("a refused write must not touch the file")
			}
		})
	}
}

// A key that blocked on a stale revision is the whole point of the check: the
// caller's editor is showing a map that is no longer the file's.
func TestDocRefusesAFileThatMoved(t *testing.T) {
	path := writeTemp(t, "keybindings.yml", "app.exit: ctrl+q\n")
	d, err := OpenDoc(path, FormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	rev := d.Revision()
	if err := d.SetStrings([]string{"app.exit"}, []string{"ctrl+w"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("app.exit: alt+q\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.Save(rev); !errors.Is(err, ErrStale) {
		t.Fatalf("a file that moved must be refused, got %v", err)
	}
	if got := readBack(t, path); got != "app.exit: alt+q\n" {
		t.Fatalf("the other writer's file was clobbered: %q", got)
	}
	// Treating the file as new (no revision) writes: that is the explicit
	// create-the-file path the empty document reports.
	if err := d.Save(""); err != nil {
		t.Fatal(err)
	}
}

// An existing file with nothing in it is the state a full reset leaves behind,
// and it must be writable: the revision of an empty file is the hash of its
// zero bytes, never the empty string that means "do not check".
func TestDocWritesIntoAnEmptyFile(t *testing.T) {
	path := writeTemp(t, "keybindings.yml", "")
	d, err := OpenDoc(path, FormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Exists() || d.Revision() == "" {
		t.Fatalf("an empty file exists and has a revision: exists=%v rev=%q", d.Exists(), d.Revision())
	}
	if err := d.SetStrings([]string{"app.exit"}, []string{"ctrl+q"}); err != nil {
		t.Fatal(err)
	}
	if err := d.Save(d.Revision()); err != nil {
		t.Fatalf("an empty file must be writable: %v", err)
	}
	if got := readBack(t, path); !strings.Contains(got, `app.exit: ["ctrl+q"]`) {
		t.Fatalf("the file says:\n%s", got)
	}
}

// A key map is not a TOML document anywhere PiCode writes today; the primitive
// says so instead of shipping an unexercised path.
func TestListLiteralPerFormat(t *testing.T) {
	if lit, err := listLiteral(FormatTOML, []string{"ctrl+q"}); err != nil || lit != `["ctrl+q"]` {
		t.Fatalf("a TOML array of strings is valid TOML: lit=%q err=%v", lit, err)
	}
	if lit, err := listLiteral(FormatYAML, nil); err != nil || lit != "[]" {
		t.Fatalf("an empty list is how a CLI unbinds: lit=%q err=%v", lit, err)
	}
}
