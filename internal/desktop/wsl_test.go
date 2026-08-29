package desktop

import (
	"encoding/base64"
	"strings"
	"testing"
)

// realListVerbose is the exact output of `wsl.exe -l -v` on the machine this
// feature was built against: UTF-16LE, no BOM, CRLF. Captured rather than
// written, because the encoding is the whole point of the test.
const realListVerbose = "IAAgAE4AQQBNAEUAIAAgACAAIAAgACAAUwBUAEEAVABFACAAIAAgACAAIAAgACAAIAAgACAAIABWAEUAUgBTAEkATwBOAA0ACgAqACAAVQBiAHUAbgB0AHUAIAAgACAAIABSAHUAbgBuAGkAbgBnACAAIAAgACAAIAAgACAAIAAgADIADQAKAA=="

func realBytes(t *testing.T) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(realListVerbose)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseRealWSLOutput(t *testing.T) {
	got, err := ParseDistros(realBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d distros, want 1: %+v", len(got), got)
	}
	want := Distro{Name: "Ubuntu", State: "Running", Version: 2, Default: true}
	if got[0] != want {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
	if !got[0].Running() {
		t.Error("Running() = false for a Running distro")
	}
}

// Reading that output as UTF-8 is the classic WSL bug; this pins that we do
// not, by showing what the naive reading would have produced.
func TestDecodeWindowsRejectsTheNaiveReading(t *testing.T) {
	raw := realBytes(t)
	if naive := string(raw); !containsNUL(naive) {
		t.Skip("fixture is not UTF-16 after all")
	}
	got := DecodeWindows(raw)
	if containsNUL(got) {
		t.Errorf("decoded text still has interior NULs: %q", got)
	}
	if want := "Ubuntu"; !strings.Contains(got, want) {
		t.Errorf("decoded %q, want it to contain %q", got, want)
	}
}

func containsNUL(s string) bool { return strings.ContainsRune(s, 0) }

func TestDecodeWindows(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"plain UTF-8 passes through", []byte("Ubuntu\r\n"), "Ubuntu\r\n"},
		{"UTF-16LE without a BOM", []byte{'U', 0, 'b', 0, 'u', 0}, "Ubu"},
		{"UTF-16LE with a BOM", []byte{0xFF, 0xFE, 'U', 0, 'b', 0}, "Ub"},
		{"empty", nil, ""},
		{"an odd length is not UTF-16", []byte{'a', 0, 'b'}, "a\x00b"},
		{"UTF-8 with accents is not mistaken for UTF-16", []byte("distribuição"), "distribuição"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecodeWindows(tt.in); got != tt.want {
				t.Errorf("DecodeWindows = %q, want %q", got, tt.want)
			}
		})
	}
}

// utf16 encodes an ASCII string the way wsl.exe would, for table cases.
func utf16le(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		out = append(out, byte(r), byte(r>>8))
	}
	return out
}

func TestParseDistros(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Distro
	}{
		{
			name: "several distros, only one default",
			in: "  NAME              STATE           VERSION\r\n" +
				"* Ubuntu            Running         2\r\n" +
				"  docker-desktop    Stopped         2\r\n" +
				"  Legacy            Stopped         1\r\n",
			want: []Distro{
				{Name: "Ubuntu", State: "Running", Version: 2, Default: true},
				{Name: "docker-desktop", State: "Stopped", Version: 2},
				{Name: "Legacy", State: "Stopped", Version: 1},
			},
		},
		{
			// Read from the right, so a space in the name survives.
			name: "a name with a space",
			in: "  NAME                STATE           VERSION\r\n" +
				"* My Distro          Running         2\r\n",
			want: []Distro{{Name: "My Distro", State: "Running", Version: 2, Default: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDistros(utf16le(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d distros, want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("distro %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseDistrosOnAnEmptyMachine(t *testing.T) {
	for _, in := range []string{"", "  NAME  STATE  VERSION\r\n"} {
		if _, err := ParseDistros(utf16le(in)); err == nil {
			t.Errorf("an empty list (%q) was accepted", in)
		}
	}
}

func TestPick(t *testing.T) {
	ubuntu := Distro{Name: "Ubuntu", State: "Running", Version: 2, Default: true}
	debian := Distro{Name: "Debian", State: "Stopped", Version: 2}
	docker := Distro{Name: "docker-desktop", State: "Running", Version: 2}
	legacy := Distro{Name: "Legacy", State: "Stopped", Version: 1}

	tests := []struct {
		name      string
		distros   []Distro
		preferred string
		want      string
		wantErr   bool
	}{
		{name: "the only one", distros: []Distro{ubuntu}, want: "Ubuntu"},
		{name: "docker-desktop is infrastructure, not a candidate",
			distros: []Distro{ubuntu, docker}, want: "Ubuntu"},
		{name: "the default breaks a tie",
			distros: []Distro{ubuntu, debian}, want: "Ubuntu"},
		{name: "an explicit choice wins over the default",
			distros: []Distro{ubuntu, debian}, preferred: "Debian", want: "Debian"},
		{name: "a name is matched case-insensitively",
			distros: []Distro{ubuntu}, preferred: "ubuntu", want: "Ubuntu"},
		{name: "an unknown name is an error",
			distros: []Distro{ubuntu}, preferred: "Fedora", wantErr: true},
		{name: "WSL 1 is not usable",
			distros: []Distro{legacy}, wantErr: true},
		{name: "asking for a WSL 1 distro by name is an error",
			distros: []Distro{ubuntu, legacy}, preferred: "Legacy", wantErr: true},
		{name: "ambiguous without a default",
			distros: []Distro{debian, {Name: "Arch", Version: 2}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Pick(tt.distros, tt.preferred)
			if tt.wantErr {
				if err == nil {
					t.Errorf("got %+v, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.want {
				t.Errorf("picked %q, want %q", got.Name, tt.want)
			}
		})
	}
}
