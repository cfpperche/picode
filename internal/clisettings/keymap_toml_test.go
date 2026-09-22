package clisettings

import (
	"strings"
	"testing"
)

// The nested TOML case, which is what a `[tui.keymap.<context>.<action>]` map
// needs (docs/plans/keyboard-pane.md, P3): the path names one element per level,
// the insert has to create the table when the file has none, and an existing
// value has to be replaced without disturbing the table around it.

func TestDocPathsInTOML(t *testing.T) {
	t.Run("inserts into a table the file does not have", func(t *testing.T) {
		path := writeTemp(t, "config.toml", "model = \"gpt-5\"\n")
		d, err := OpenDoc(path, FormatTOML)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetStrings([]string{"tui", "keymap", "composer", "submit"}, []string{"ctrl+m"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(d.Revision()); err != nil {
			t.Fatal(err)
		}
		got := readBack(t, path)
		if !strings.Contains(got, "[tui.keymap.composer]") || !strings.Contains(got, `submit = ["ctrl+m"]`) {
			t.Fatalf("the table was not created:\n%s", got)
		}
		if !strings.Contains(got, `model = "gpt-5"`) {
			t.Fatalf("the file lost a key PiCode does not manage:\n%s", got)
		}
		values, found, err := d.Strings("tui", "keymap", "composer", "submit")
		if err != nil || !found || strings.Join(values, ",") != "ctrl+m" {
			t.Fatalf("read back found=%v values=%v err=%v", found, values, err)
		}
	})
	t.Run("splices a value inside a table that exists", func(t *testing.T) {
		path := writeTemp(t, "config.toml", "[tui.keymap.global]\nsubmit = [\"enter\"]\nopen = [\"ctrl+o\"]\n")
		d, err := OpenDoc(path, FormatTOML)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetStrings([]string{"tui", "keymap", "global", "submit"}, []string{"enter", "ctrl+m"}); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(d.Revision()); err != nil {
			t.Fatal(err)
		}
		got := readBack(t, path)
		if !strings.Contains(got, `submit = ["enter", "ctrl+m"]`) || !strings.Contains(got, `open = ["ctrl+o"]`) {
			t.Fatalf("the sibling row did not survive:\n%s", got)
		}
	})
	t.Run("removing the last key takes the table with it", func(t *testing.T) {
		path := writeTemp(t, "config.toml", "model = \"gpt-5\"\n\n[tui.keymap.global]\nsubmit = [\"enter\"]\n")
		before := `model = "gpt-5"` + "\n"
		d, err := OpenDoc(path, FormatTOML)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.Remove("tui", "keymap", "global", "submit"); err != nil {
			t.Fatal(err)
		}
		if err := d.Save(d.Revision()); err != nil {
			t.Fatal(err)
		}
		got := readBack(t, path)
		if strings.Contains(got, "[tui.keymap.global]") {
			t.Fatalf("an emptied table must go with its last key:\n%s", got)
		}
		if !strings.Contains(got, before) {
			t.Fatalf("the rest of the file changed:\n%q", got)
		}
	})
	// A hand-written multi-line array is the case a single line span cannot
	// address. What matters is that the result parses: either the value is
	// replaced, or the write is refused — never a half-removed array in a file
	// the CLI reads.
	t.Run("a multi-line array is replaced or refused, never half-removed", func(t *testing.T) {
		path := writeTemp(t, "config.toml", "[tui.keymap.global]\nsubmit = [\n  \"enter\",\n  \"ctrl+m\",\n]\n")
		d, err := OpenDoc(path, FormatTOML)
		if err != nil {
			t.Fatal(err)
		}
		err = d.SetStrings([]string{"tui", "keymap", "global", "submit"}, []string{"alt+m"})
		if err == nil {
			if saveErr := d.Save(d.Revision()); saveErr != nil {
				t.Fatalf("a splice that cannot be written must be refused at the splice: %v", saveErr)
			}
			got := readBack(t, path)
			if strings.Contains(got, "enter") {
				t.Fatalf("the old chords are still in the file:\n%s", got)
			}
			again, err := OpenDoc(path, FormatTOML)
			if err != nil {
				t.Fatal(err)
			}
			values, found, err := again.Strings("tui", "keymap", "global", "submit")
			if err != nil || !found || strings.Join(values, ",") != "alt+m" {
				t.Fatalf("read back found=%v values=%v err=%v", found, values, err)
			}
			return
		}
		if !strings.Contains(err.Error(), "submit") {
			t.Fatalf("a refusal must name the row: %v", err)
		}
		if got := readBack(t, path); !strings.Contains(got, "enter") {
			t.Fatalf("a refused write must leave the file alone:\n%s", got)
		}
	})
}
