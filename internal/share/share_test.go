package share

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/cfpperche/picode/internal/tlsutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUsablePhoneIP(t *testing.T) {
	cases := []struct {
		ip string
		ok bool
	}{
		{"127.0.0.1", false},
		{"169.254.1.1", false},
		{"172.17.0.2", false},
		{"10.255.255.254", false},
		{"192.168.15.28", true},
		{"10.0.0.5", true},
		{"100.87.149.83", true},
	}
	for _, c := range cases {
		if got := UsablePhoneIP(net.ParseIP(c.ip)); got != c.ok {
			t.Errorf("UsablePhoneIP(%s) = %v, want %v", c.ip, got, c.ok)
		}
	}
}

func TestDiagnoseInsecureFailsHTTPS(t *testing.T) {
	r := Diagnose(Input{Insecure: true, BindHost: "0.0.0.0", Port: 8445})
	if r.Ready {
		t.Fatal("insecure must not be ready")
	}
	found := false
	for _, c := range r.Checks {
		if c.ID == "https" && !c.OK && c.Action != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing https action: %+v", r.Checks)
	}
}

func TestDetectPhoneOS(t *testing.T) {
	if detectPhoneOS("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)") != "ios" {
		t.Fatal("iphone")
	}
	if detectPhoneOS("Mozilla/5.0 (Linux; Android 14)") != "android" {
		t.Fatal("android")
	}
	if detectPhoneOS("Mozilla/5.0 (Windows NT 10.0)") != "other" {
		t.Fatal("desktop")
	}
}

func TestMissingAny(t *testing.T) {
	have := []string{"localhost", "192.168.15.28"}
	if !missingAny(have, []string{"localhost", "192.168.15.110"}) {
		t.Fatal("should detect new LAN IP")
	}
	if missingAny(have, []string{"localhost"}) {
		t.Fatal("existing name flagged")
	}
}

func TestDiagnoseLoopbackBindFails(t *testing.T) {
	r := Diagnose(Input{Insecure: false, BindHost: "127.0.0.1", Port: 8445})
	for _, c := range r.Checks {
		if c.ID == "bind" && c.OK {
			t.Fatal("loopback bind should fail")
		}
	}
}

func TestPublicURLOnTheTailnetNameIsTrusted(t *testing.T) {
	dir := t.TempDir()
	name := "box.tail1234.ts.net"
	// A leaf for the name, where the daemon keeps the Tailscale one.
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(0, 3, 0), DNSNames: []string{name}}
	der, _ := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	keyDER, _ := x509.MarshalECPrivateKey(key)
	writePEMFile := func(path, typ string, b []byte) {
		f, _ := os.Create(path)
		_ = pem.Encode(f, &pem.Block{Type: typ, Bytes: b})
		_ = f.Close()
	}
	writePEMFile(filepath.Join(dir, tlsutil.TailscaleCertFile), "CERTIFICATE", der)
	writePEMFile(filepath.Join(dir, tlsutil.TailscaleKeyFile), "EC PRIVATE KEY", keyDER)

	rep := Diagnose(Input{DataDir: dir, Port: 8445, BindHost: "0.0.0.0", PublicURL: "https://" + name + ":8445"})
	if rep.URL != "https://"+name+":8445/" || !rep.Trusted {
		t.Fatalf("url %q trusted %v", rep.URL, rep.Trusted)
	}
	var pub *Target
	for i := range rep.Targets {
		if rep.Targets[i].Kind == "public" {
			pub = &rep.Targets[i]
		}
	}
	if pub == nil || !pub.Trusted {
		t.Fatalf("public target %+v", pub)
	}
	// Another public host is not trusted by that leaf.
	rep = Diagnose(Input{DataDir: dir, Port: 8445, BindHost: "0.0.0.0", PublicURL: "https://other.example:8445"})
	if rep.Trusted {
		t.Fatal("a leaf for the tailnet name must not vouch for another host")
	}
}
