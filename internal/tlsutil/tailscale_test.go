package tlsutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeLeaf mints a self-signed leaf for the names into cert/key paths.
func writeLeaf(t *testing.T, certPath, keyPath string, dns []string, ips []net.IP, notAfter time.Time) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "t"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: notAfter,
		DNSNames: dns, IPAddresses: ips, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	if err := writePEM(certPath, "CERTIFICATE", der); err != nil {
		t.Fatal(err)
	}
	if err := writePEM(keyPath, "EC PRIVATE KEY", keyDER); err != nil {
		t.Fatal(err)
	}
}

func TestLiveConfigPicksByServerName(t *testing.T) {
	dir := t.TempDir()
	far := time.Now().AddDate(1, 0, 0)
	writeLeaf(t, filepath.Join(dir, CertFile), filepath.Join(dir, KeyFile), []string{"localhost"}, []net.IP{net.ParseIP("100.64.0.9")}, far)
	writeLeaf(t, filepath.Join(dir, TailscaleCertFile), filepath.Join(dir, TailscaleKeyFile), []string{"box.tail1234.ts.net"}, nil, far)
	cfg := LiveConfig(dir)
	got := func(sni string) string {
		c, err := cfg.GetCertificate(&tls.ClientHelloInfo{ServerName: sni})
		if err != nil {
			t.Fatal(err)
		}
		leaf, _ := x509.ParseCertificate(c.Certificate[0])
		if len(leaf.DNSNames) > 0 {
			return leaf.DNSNames[0]
		}
		return "?"
	}
	if got("box.tail1234.ts.net") != "box.tail1234.ts.net" {
		t.Fatal("tailnet name should get the Tailscale leaf")
	}
	if got("BOX.tail1234.ts.net") != "box.tail1234.ts.net" {
		t.Fatal("hostname match is case-insensitive")
	}
	if got("localhost") != "localhost" || got("") != "localhost" || got("other.example") != "localhost" {
		t.Fatal("everything else gets the local leaf")
	}
	// Without a Tailscale leaf on disk the local one serves the name too.
	_ = os.Remove(filepath.Join(dir, TailscaleCertFile))
	if got("box.tail1234.ts.net") != "localhost" {
		t.Fatal("missing Tailscale leaf must fall back")
	}
}

func TestNeedsTailscaleIssue(t *testing.T) {
	dir := t.TempDir()
	name := "box.tail1234.ts.net"
	if !NeedsTailscaleIssue(dir, name) {
		t.Fatal("missing leaf needs issue")
	}
	if NeedsTailscaleIssue(dir, "") {
		t.Fatal("no name, nothing to issue")
	}
	writeLeaf(t, filepath.Join(dir, TailscaleCertFile), filepath.Join(dir, TailscaleKeyFile), []string{name}, nil, time.Now().AddDate(0, 3, 0))
	if NeedsTailscaleIssue(dir, name) {
		t.Fatal("fresh leaf covering the name needs nothing")
	}
	if !NeedsTailscaleIssue(dir, "other.tail1234.ts.net") {
		t.Fatal("another name needs issue")
	}
	writeLeaf(t, filepath.Join(dir, TailscaleCertFile), filepath.Join(dir, TailscaleKeyFile), []string{name}, nil, time.Now().Add(10*24*time.Hour))
	if !NeedsTailscaleIssue(dir, name) {
		t.Fatal("inside the renewal window needs issue")
	}
}

func TestIssueTailscaleUsesTheCLIAndChecksTheResult(t *testing.T) {
	dir := t.TempDir()
	old := tailscaleCmd
	t.Cleanup(func() { tailscaleCmd = old })
	var gotArgs []string
	tailscaleCmd = func(_ context.Context, args ...string) ([]byte, error) {
		gotArgs = args
		return []byte("Use 'sudo tailscale cert ...'.\nTo not require root, use 'sudo tailscale set --operator=$USER' once."), errors.New("exit 1")
	}
	err := IssueTailscale(context.Background(), dir, "box.tail1234.ts.net")
	if err == nil || err.Error() != "tailscale cert: Use 'sudo tailscale cert ...'." {
		t.Fatalf("err = %v", err)
	}
	if len(gotArgs) != 6 || gotArgs[0] != "cert" || gotArgs[5] != "box.tail1234.ts.net" {
		t.Fatalf("args = %v", gotArgs)
	}
	// A CLI that "succeeds" but writes a leaf for another name is refused.
	tailscaleCmd = func(_ context.Context, args ...string) ([]byte, error) {
		writeLeaf(t, args[2], args[4], []string{"other.ts.net"}, nil, time.Now().AddDate(0, 3, 0))
		return nil, nil
	}
	if err := IssueTailscale(context.Background(), dir, "box.tail1234.ts.net"); err == nil {
		t.Fatal("leaf for another name accepted")
	}
	tailscaleCmd = func(_ context.Context, args ...string) ([]byte, error) {
		writeLeaf(t, args[2], args[4], []string{"box.tail1234.ts.net"}, nil, time.Now().AddDate(0, 3, 0))
		return nil, nil
	}
	if err := IssueTailscale(context.Background(), dir, "box.tail1234.ts.net"); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(filepath.Join(dir, TailscaleKeyFile)); st.Mode().Perm() != 0o600 {
		t.Fatalf("key mode %v", st.Mode().Perm())
	}
}
