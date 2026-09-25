package tlsutil

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CertNames returns a function answering the DNS names the served
// certificates cover — the mkcert/self-signed pair and the Tailscale leaf —
// lowercased. The auth gate's Host allowlist includes them (ADR-0215): a
// name the owner issued a certificate for is a name this daemon is meant to
// be reached by, and a name outside them fails the TLS handshake in any
// browser anyway. Files are re-read only when their modification time
// changes, so a reissued certificate (cert timer, Tailscale renewal) is
// picked up without a restart and a request costs two stats.
func CertNames(dataDir string) func() []string {
	paths := []string{filepath.Join(dataDir, CertFile), filepath.Join(dataDir, TailscaleCertFile)}
	var (
		mu    sync.Mutex
		stamp []time.Time
		names []string
	)
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		now := make([]time.Time, len(paths))
		for i, p := range paths {
			if fi, err := os.Stat(p); err == nil {
				now[i] = fi.ModTime()
			}
		}
		if stamp != nil && equalTimes(stamp, now) {
			return names
		}
		stamp = now
		names = names[:0:0]
		for _, p := range paths {
			names = append(names, pemDNSNames(p)...)
		}
		return names
	}
}

// pemDNSNames reads the leaf (first certificate) of a PEM file.
func pemDNSNames(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		leaf, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil
		}
		out := make([]string, 0, len(leaf.DNSNames))
		for _, n := range leaf.DNSNames {
			out = append(out, strings.ToLower(n))
		}
		return out
	}
}

func equalTimes(a, b []time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}
