package desktop

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// lxssOutput is `reg query HKCU\…\Lxss /s`, verbatim from a WSL 2.7 machine,
// with the second distro added and the key order left as the registry gave it.
const lxssOutput = `
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Lxss
    DefaultVersion    REG_DWORD    0x2
    DefaultDistribution    REG_SZ    {c2af16cd-4a9b-4d07-94bd-1c82c914c39e}

HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Lxss\{c2af16cd-4a9b-4d07-94bd-1c82c914c39e}
    State    REG_DWORD    0x1
    DistributionName    REG_SZ    Ubuntu
    Version    REG_DWORD    0x2
    BasePath    REG_SZ    C:\Users\cfpp\AppData\Local\wsl\{c2af16cd-4a9b-4d07-94bd-1c82c914c39e}
    VhdFileName    REG_SZ    ext4.vhdx

HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Lxss\{aa11bb22-cc33-dd44-ee55-ff6677889900}
    DistributionName    REG_SZ    Debian
    BasePath    REG_SZ    E:\WSL\Debian
`

func TestParseLxssFindsEveryDistroWithItsOwnFolder(t *testing.T) {
	folders := ParseLxss(utf16le(lxssOutput))
	if len(folders) != 2 {
		t.Fatalf("got %d folders, want 2 (%+v)", len(folders), folders)
	}
	ubuntu := folders[0]
	if ubuntu.Name != "Ubuntu" {
		t.Errorf("name: got %q", ubuntu.Name)
	}
	wantPath := `C:\Users\cfpp\AppData\Local\wsl\{c2af16cd-4a9b-4d07-94bd-1c82c914c39e}`
	if ubuntu.BasePath != wantPath {
		t.Errorf("base path: got %q, want %q", ubuntu.BasePath, wantPath)
	}
	if ubuntu.VHDX != "ext4.vhdx" {
		t.Errorf("vhdx: got %q", ubuntu.VHDX)
	}
	// Debian has no VhdFileName value; the default is what WSL has always
	// used, and guessing a folder path instead would be the real mistake.
	if folders[1].VHDX != "ext4.vhdx" {
		t.Errorf("default vhdx: got %q", folders[1].VHDX)
	}
	// The header key (DefaultVersion, no DistributionName) is not a distro.
	for _, f := range folders {
		if f.Name == "" || f.BasePath == "" {
			t.Errorf("a folder without a name and a path is not a distro: %+v", f)
		}
	}
}

func TestPickFolderMatchesTheNameWSLReports(t *testing.T) {
	folders := []DistroFolder{{Name: "Ubuntu", BasePath: `C:\a`}, {Name: "Debian", BasePath: `E:\b`}}
	got, err := pickFolder(folders, "debian")
	if err != nil || got.BasePath != `E:\b` {
		t.Fatalf("case-insensitive pick: %+v %v", got, err)
	}
	if _, err := pickFolder(folders, "Fedora"); err == nil {
		t.Error("an unknown name must fail rather than pick the first distro")
	}
	// One distro is unambiguous even when the caller names it differently.
	one, err := pickFolder([]DistroFolder{{Name: "Ubuntu", BasePath: `C:\a`}}, "Ubuntu-24.04")
	if err != nil || one.BasePath != `C:\a` {
		t.Errorf("single distro: %+v %v", one, err)
	}
}

func TestParseVolumeJSONSkipsWhateverPrintedFirst(t *testing.T) {
	out := []byte("WARNING: something\r\n{\"len\":233711861760,\"total\":510964789248,\"free\":29251764224}\r\n")
	vol, err := ParseVolumeJSON(out)
	if err != nil {
		t.Fatalf("ParseVolumeJSON: %v", err)
	}
	if vol.Length != 233711861760 || vol.Total != 510964789248 || vol.Free != 29251764224 {
		t.Errorf("got %+v", vol)
	}
	if _, err := ParseVolumeJSON([]byte("Get-Item : Cannot find path")); err == nil {
		t.Error("a failure with no JSON must be an error, not a zero-byte disk")
	}
}

// TestParseSparseRangesReadsNumbersNotWords is the point of the parser: fsutil
// prints in the display language, so "Allocated range", "Offset" and "Length"
// can all be translated — the two hex numbers cannot.
func TestParseSparseRangesReadsNumbersNotWords(t *testing.T) {
	cases := []struct {
		name         string
		out          string
		wantAlloc    int64
		wantIsSparse bool
	}{
		{
			name:         "one extent",
			out:          "Allocated range[1]: Offset: 0x0         Length: 0x6400000\n",
			wantAlloc:    0x6400000,
			wantIsSparse: true,
		},
		{
			name:         "two extents add up",
			out:          "Allocated range[1]: Offset: 0x0 Length: 0x1000\nAllocated range[2]: Offset: 0x2000 Length: 0x3000\n",
			wantAlloc:    0x4000,
			wantIsSparse: true,
		},
		{
			name:         "translated labels",
			out:          "Intervalo alocado[1]: Deslocamento: 0x0   Comprimento: 0x6400000\n",
			wantAlloc:    0x6400000,
			wantIsSparse: true,
		},
		{
			name:         "not sparse",
			out:          "The specified file is NOT sparse\r\n",
			wantAlloc:    0,
			wantIsSparse: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			alloc, sparse := ParseSparseRanges(utf16le(c.out))
			if alloc != c.wantAlloc || sparse != c.wantIsSparse {
				t.Errorf("got (%d, %v), want (%d, %v)", alloc, sparse, c.wantAlloc, c.wantIsSparse)
			}
		})
	}
}

func TestParseWSLVersionSkipsTheLabel(t *testing.T) {
	out := utf16le("WSL version: 2.7.0.0\r\nKernel version: 6.6.114.1-1\r\n")
	if got := ParseWSLVersion(out); got != "2.7.0.0" {
		t.Errorf("got %q, want 2.7.0.0", got)
	}
	translated := utf16le("Versão do WSL: 2.7.0.0\r\nVersão do kernel: 6.6.114.1-1\r\n")
	if got := ParseWSLVersion(translated); got != "2.7.0.0" {
		t.Errorf("translated label: got %q", got)
	}
	if got := ParseWSLVersion(nil); got != "" {
		t.Errorf("no output must not invent a version: got %q", got)
	}
}

func TestHeldIsTheSpaceWindowsCannotSee(t *testing.T) {
	f := DiskFacts{AllocatedBytes: 233_711_861_760} // 217.7 GiB, not sparse
	if got := Held(f, 134_980_124_672); got != 98_731_737_088 {
		t.Errorf("got %d", got)
	}
	// A sparse file that WSL has already given back holds nothing extra.
	if got := Held(DiskFacts{AllocatedBytes: 100, Sparse: true}, 100); got != 0 {
		t.Errorf("sparse and equal: got %d", got)
	}
	// An unmeasured distro must not invent a number.
	if got := Held(f, 0); got != 0 {
		t.Errorf("unknown usage: got %d", got)
	}
}

// TestDistroDiskUsesWSLsOwnBookkeeping drives the whole read with the real
// replies captured from a WSL 2.7 machine: the VHDX path has to come from the
// registry, the allocated size from fsutil, and the capability from the help
// text — nothing may be guessed from a fixed folder name.
func TestDistroDiskUsesWSLsOwnBookkeeping(t *testing.T) {
	r := &fakeRunner{replies: [][]byte{
		utf16le(lxssOutput),
		[]byte(`{"len":233711861760,"total":510964789248,"free":29251764224}`),
		utf16le("The specified file is NOT sparse\r\n"),
		utf16le("WSL version: 2.7.0.0\r\nKernel version: 6.6.114.1-1\r\n"),
		utf16le("    --manage <Distro> <Options...>\n            --set-sparse, -s <true|false>\n"),
	}}
	facts, err := DistroDisk(r, "Ubuntu")
	if err != nil {
		t.Fatalf("DistroDisk: %v", err)
	}
	wantPath := `C:\Users\cfpp\AppData\Local\wsl\{c2af16cd-4a9b-4d07-94bd-1c82c914c39e}\ext4.vhdx`
	if facts.VHDXPath != wantPath {
		t.Errorf("path: got %q, want %q", facts.VHDXPath, wantPath)
	}
	if facts.VHDXBytes != 233711861760 || facts.FreeBytes != 29251764224 {
		t.Errorf("volume: %+v", facts)
	}
	if facts.Sparse {
		t.Error("a file fsutil calls NOT sparse must not be reported as sparse")
	}
	// Not sparse means the allocation needs no second measurement.
	if facts.AllocatedBytes != facts.VHDXBytes {
		t.Errorf("allocated: got %d, want the full length", facts.AllocatedBytes)
	}
	if facts.WSL != "2.7.0.0" || !facts.CanSparse {
		t.Errorf("capability: wsl %q canSparse %v", facts.WSL, facts.CanSparse)
	}

	// The path handed to fsutil and PowerShell is the VHDX, not the folder.
	if got := r.calls[2][len(r.calls[2])-1]; got != wantPath {
		t.Errorf("fsutil was given %q", got)
	}
	if script := strings.Join(r.calls[1], " "); !strings.Contains(script, wantPath) {
		t.Errorf("the volume read did not name the file: %s", script)
	}
}

// TestDistroDiskReadsHelpDespiteTheExitStatus pins what the first live run of
// the merged report exposed: `wsl --help` exits 255 on every machine we have
// seen while printing the whole text to stdout. Dropping the output on a
// non-zero exit made CanSparse false everywhere real — and the report's one
// suggestion never appeared.
func TestDistroDiskReadsHelpDespiteTheExitStatus(t *testing.T) {
	r := &fakeRunner{
		replies: [][]byte{
			utf16le(lxssOutput),
			[]byte(`{"len":233711861760,"total":510964789248,"free":29251764224}`),
			utf16le("The specified file is NOT sparse\r\n"),
			utf16le("WSL version: 2.7.0.0\r\n"),
			utf16le("    --manage <Distro> <Options...>\n            --set-sparse, -s <true|false>\n"),
		},
		errs: []error{nil, nil, nil, nil, errors.New("exit status 255")},
	}
	facts, err := DistroDisk(r, "Ubuntu")
	if err != nil {
		t.Fatalf("DistroDisk: %v", err)
	}
	if !facts.CanSparse {
		t.Error("the help text was on stdout; the exit status must not hide it")
	}
	if facts.WSL != "2.7.0.0" {
		t.Errorf("version: got %q", facts.WSL)
	}
}

func TestDistroDiskReportsAnUnknownDistro(t *testing.T) {
	r := &fakeRunner{replies: [][]byte{
		utf16le(lxssOutput),
		[]byte("WSL version: 2.7.0.0\r\n"),
		utf16le("WSL version: 2.7.0.0\r\n"),
		utf16le("--set-sparse\n"),
	}}
	if _, err := DistroDisk(r, "Fedora"); err == nil {
		t.Fatal("an unknown distro must fail, not fall back to another one")
	} else if !strings.Contains(err.Error(), "Fedora") {
		t.Errorf("the error has to name the distro: %v", err)
	}
}

func TestDiskFactsTimestampIsUTC(t *testing.T) {
	f := DiskFacts{At: time.Now().UTC()}
	if f.At.Location() != time.UTC {
		t.Error("a fact read from another machine is compared to this one's clock; UTC is the only safe stamp")
	}
}
