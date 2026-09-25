package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnsureGeneratesAndLoads(t *testing.T) {
	dir := t.TempDir()

	cert, err := Ensure(dir)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("no certificate bytes")
	}

	// Second call loads from disk (same serial → same cert).
	cert2, err := Ensure(dir)
	if err != nil {
		t.Fatalf("Ensure reload: %v", err)
	}
	c1, _ := x509.ParseCertificate(cert.Certificate[0])
	c2, _ := x509.ParseCertificate(cert2.Certificate[0])
	if c1.SerialNumber.Cmp(c2.SerialNumber) != 0 {
		t.Error("second Ensure regenerated instead of loading")
	}

	// SANs must include localhost.
	found := false
	for _, n := range c1.DNSNames {
		if n == "localhost" {
			found = true
		}
	}
	if !found {
		t.Error("localhost missing from DNS SANs")
	}
}

func TestWarnIfExpiringQuietForFreshCert(t *testing.T) {
	dir := t.TempDir()
	if _, err := Ensure(dir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	// Fresh 10y cert: warning path must not crash (output goes to log).
	WarnIfExpiring(dir, 30*24*time.Hour)

	// Missing files: quiet no-op.
	WarnIfExpiring(filepath.Join(dir, "missing"), time.Hour)
}

func TestGeneratedCertUsableByTLS(t *testing.T) {
	dir := t.TempDir()
	if _, err := Ensure(dir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	// tls.X509KeyPair round-trip via files.
	loaded, err := tls.LoadX509KeyPair(filepath.Join(dir, CertFile), filepath.Join(dir, KeyFile))
	if err != nil {
		t.Fatalf("LoadX509KeyPair: %v", err)
	}
	if len(loaded.Certificate) == 0 {
		t.Error("loaded pair empty")
	}
}

func TestCertNamesFollowsTheFileOnDisk(t *testing.T) {
	dir := t.TempDir()
	names := CertNames(dir)
	if got := names(); len(got) != 0 {
		t.Fatalf("no certificate yet, got %v", got)
	}
	if _, err := Ensure(dir); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(names(), ",")
	if !strings.Contains(got, "localhost") || !strings.Contains(got, "picode.local") {
		t.Fatalf("self-signed names missing: %q", got)
	}
	// A reissued leaf (the cert timer, a Tailscale renewal) is picked up
	// without a restart: a Tailscale leaf appears beside the pair.
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := x509.Certificate{SerialNumber: big.NewInt(7), DNSNames: []string{"Box-1.Tail.ts.net"},
		NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePEM(filepath.Join(dir, TailscaleCertFile), "CERTIFICATE", der); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(names(), ","); !strings.Contains(got, "box-1.tail.ts.net") {
		t.Fatalf("tailscale leaf not picked up (lowercased): %q", got)
	}
}

func TestSkipInterface(t *testing.T) {
	cases := []struct {
		name  string
		flags net.Flags
		skip  bool
	}{
		{"lo", net.FlagUp | net.FlagLoopback, true},
		{"docker0", net.FlagUp, true},
		{"br-8649fff496d0", net.FlagUp, true},
		{"veth1a2b3c", net.FlagUp, true},
		{"eth0", net.FlagUp, false},
		{"eth1", net.FlagUp, false},
		{"tailscale0", net.FlagUp, false},
		{"wlan0", net.FlagUp, false},
		{"bridge0", net.FlagUp, false}, // a host's own LAN bridge is not Docker's
	}
	for _, c := range cases {
		if got := skipInterface(c.name, c.flags); got != c.skip {
			t.Errorf("skipInterface(%q) = %v, want %v", c.name, got, c.skip)
		}
	}
}

// On this machine the result must still hold loopback and nothing from a
// Docker bridge.
func TestLocalNamesKeepsLoopbackAndDropsBridges(t *testing.T) {
	_, ips := LocalNames()
	have := map[string]bool{}
	for _, ip := range ips {
		have[ip.String()] = true
	}
	if !have["127.0.0.1"] || !have["::1"] {
		t.Fatalf("loopback missing: %v", ips)
	}
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if !skipInterface(ifc.Name, ifc.Flags) || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && have[n.IP.String()] {
				t.Errorf("%s from %s is in LocalNames", n.IP, ifc.Name)
			}
		}
	}
}
