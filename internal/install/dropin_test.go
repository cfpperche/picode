package install

import (
	"os"
	"strings"
	"testing"
)

func TestEnvDropInWriteMergeQuote(t *testing.T) {
	home := t.TempDir()
	path, err := WriteEnvDropIn(home, map[string]string{"PICODE_DATA": "/srv/picode", "NODE_EXTRA_CA_CERTS": "/home/a b/root CA.pem"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	text := string(b)
	if !strings.Contains(text, "[Service]\n") || !strings.Contains(text, "Environment=PICODE_DATA=/srv/picode\n") ||
		!strings.Contains(text, `Environment="NODE_EXTRA_CA_CERTS=/home/a b/root CA.pem"`) {
		t.Fatalf("drop-in:\n%s", text)
	}
	// Merge: a new key joins, an existing one is replaced, "" removes.
	if _, err := WriteEnvDropIn(home, map[string]string{"PICODE_HOST": "0.0.0.0", "PICODE_DATA": "/data", "NODE_EXTRA_CA_CERTS": ""}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadEnvDropIn(home)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"PICODE_HOST": "0.0.0.0", "PICODE_DATA": "/data"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestEnvDropInMissingIsEmpty(t *testing.T) {
	got, err := ReadEnvDropIn(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestQuoteEnvRoundTrip(t *testing.T) {
	for _, v := range []string{`plain=1`, `K=a b`, `K=say "hi"`, `K=back\\slash`, `K=#hash`} {
		back := parseDropIn("[Service]\nEnvironment=" + quoteEnv(v) + "\n")
		k, val, _ := strings.Cut(v, "=")
		if back[k] != val {
			t.Errorf("%q → %q → %v", v, quoteEnv(v), back)
		}
	}
}

func TestParseEnvFlag(t *testing.T) {
	if k, v, err := ParseEnvFlag("PICODE_DATA=/x"); err != nil || k != "PICODE_DATA" || v != "/x" {
		t.Fatal(k, v, err)
	}
	for _, bad := range []string{"NOEQ", "=v", "a b=c"} {
		if _, _, err := ParseEnvFlag(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
