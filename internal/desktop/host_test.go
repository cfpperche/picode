package desktop

import "testing"

func TestParseMeminfo(t *testing.T) {
	m, err := ParseMeminfo("MemTotal:       32729120 kB\nMemFree: 1 kB\nMemAvailable:   18654340 kB\nCached:         14089964 kB\nSwapCached: 5 kB\nSwapTotal:       8388608 kB\nSwapFree:        2220540 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if m.TotalBytes != 32729120*1024 || m.AvailableBytes != 18654340*1024 || m.CachedBytes != 14089964*1024 ||
		m.SwapTotalBytes != 8388608*1024 || m.SwapFreeBytes != 2220540*1024 {
		t.Errorf("got %+v", m)
	}
	if _, err := ParseMeminfo("nothing here"); err == nil {
		t.Error("a text without MemTotal must be an error, not an empty machine")
	}
}

func TestParseHostMemory(t *testing.T) {
	h, err := parseHostMemory("banner\r\n{\"total\":68438863872,\"freeKiB\":26151704,\"vm\":21808197632}\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if h.TotalBytes != 68438863872 || h.FreeBytes != 26151704*1024 || h.VMWorkingSetBytes != 21808197632 {
		t.Errorf("got %+v", h)
	}
	if _, err := parseHostMemory(`{"total":0}`); err == nil {
		t.Error("zero total must be an error")
	}
}

// TestParseWSLVersionsByPosition: the labels are translated, the order is not.
func TestParseWSLVersionsByPosition(t *testing.T) {
	en := "WSL version: 2.7.0.0\r\nKernel version: 6.6.114.1-1\r\nWSLg version: 1.0.71\r\nMSRDC version: 1.2.6676\r\nDirect3D version: 1.611.1-81528511\r\nDXCore version: 10.0.26100.1-240331-1435.ge-release\r\nWindows version: 10.0.26200.9278\r\n"
	pt := "Versão do WSL: 2.7.0.0\r\nVersão do kernel: 6.6.114.1-1\r\nVersão do WSLg: 1.0.71\r\nVersão do MSRDC: 1.2.6676\r\nVersão do Direct3D: 1.611.1\r\nVersão do DXCore: 10.0.26100.1\r\nVersão do Windows: 10.0.26200.9278\r\n"
	for name, text := range map[string]string{"en": en, "pt": pt} {
		v, err := ParseWSLVersions(text)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if v.WSL != "2.7.0.0" || v.Kernel != "6.6.114.1-1" || v.WSLg != "1.0.71" || v.Windows != "10.0.26200.9278" {
			t.Errorf("%s: got %+v", name, v)
		}
	}
	if _, err := ParseWSLVersions("Invalid command line option: --version"); err == nil {
		t.Error("an inbox WSL's refusal must be an error")
	}
	if _, err := ParseWSLVersions("Usage: wsl.exe [Argument]\nArguments: see below\nOptions: --exec\n"); err == nil {
		t.Error("usage text is not a version")
	}
}

func TestNewerVersion(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"2.7.1", "2.7.0.0", true},
		{"v2.8.0", "2.7.0.0", true},
		{"2.7.0", "2.7.0.0", false},
		{"2.6.3", "2.7.0.0", false},
		{"2.7.0-pre", "2.7.0.0", false}, // unparsable → never nags
		{"", "2.7.0.0", false},
	}
	for _, c := range cases {
		if got := NewerVersion(c.latest, c.current); got != c.want {
			t.Errorf("NewerVersion(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}
