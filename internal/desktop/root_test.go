package desktop

import (
	"encoding/base64"
	"strings"
	"testing"
)

// TestRootArgsNeverUseAShell: every root call goes through --exec, and the
// wsl.conf content travels as one base64 argument — data, never syntax.
func TestRootArgsNeverUseAShell(t *testing.T) {
	a := RootArgs("Ubuntu", "cat", WSLConfPath)
	if strings.Join(a, " ") != "-d Ubuntu -u root --exec cat /etc/wsl.conf" {
		t.Errorf("args = %v", a)
	}
	content := "[boot]\nsystemd=true\n$(rm -rf /)\n"
	w := WriteWSLConfArgs("Ubuntu", content)
	if w[4] != "--exec" || w[5] != "sh" || w[6] != "-c" || w[7] != writeWSLConfScript {
		t.Fatalf("write args = %v", w)
	}
	last := w[len(w)-1]
	dec, err := base64.StdEncoding.DecodeString(last)
	if err != nil || string(dec) != content {
		t.Errorf("content arg decodes to %q (%v)", dec, err)
	}
	if strings.ContainsAny(last, "$`;\n ") {
		t.Errorf("the content argument must be plain base64: %q", last)
	}
}

// TestSystemCachesAreAClosedList: only table ids resolve, and none can be
// mistaken for a person's cache id.
func TestSystemCachesAreAClosedList(t *testing.T) {
	for _, c := range SystemCaches {
		if !strings.HasPrefix(c.ID, "system:") || len(c.Prune) == 0 || len(c.Paths) == 0 {
			t.Errorf("bad row %+v", c)
		}
	}
	for _, id := range []string{"apt", "go-build", "system:../../x", "system:apt; rm"} {
		if _, ok := SystemCacheByID(id); ok {
			t.Errorf("%q resolved", id)
		}
	}
}

func TestMeasureSystemCaches(t *testing.T) {
	r := &stubRunner{out: []byte("123\t/var/cache/apt/archives\n4567\t/var/log/journal\n")}
	got, err := MeasureSystemCaches(r, "Ubuntu")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Bytes != 123 || got[1].Bytes != 4567 {
		t.Errorf("got %+v", got)
	}
	if !strings.Contains(strings.Join(r.last, " "), "-u root --exec sh -c "+measureScript+" picode-measure /var/cache/apt/archives /var/log/journal") {
		t.Errorf("call = %v", r.last)
	}
}

type stubRunner struct {
	out  []byte
	err  error
	last []string
}

func (s *stubRunner) Output(name string, args ...string) ([]byte, error) {
	s.last = append([]string{name}, args...)
	return s.out, s.err
}

func (s *stubRunner) Run(name string, args ...string) error {
	_, err := s.Output(name, args...)
	return err
}
