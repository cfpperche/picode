package communication

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LocalTrust exports public CA material for this local server only. It verifies
// the leaf's issuer and hostname before adding anything to a child trust bundle.
// No private key is read, and certificate verification is never disabled.
func LocalTrust(data, endpoint string, extras []string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" {
		return "", nil
	}
	if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		return "", errors.New("automatic setup requires a local endpoint")
	}
	raw, err := os.ReadFile(filepath.Join(data, "cert.pem"))
	if err != nil {
		return "", errors.New("could not read the local server certificate")
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return "", errors.New("invalid local server certificate")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	roots := x509.NewCertPool()
	ca := raw
	// Self-signed installations use their leaf as the trust anchor; mkcert uses
	// its public root. The bounded CLI query is read-only and preserves CAROOT.
	if leaf.CheckSignature(leaf.SignatureAlgorithm, leaf.RawTBSCertificate, leaf.Signature) != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		out, e := exec.CommandContext(ctx, "mkcert", "-CAROOT").Output()
		if e != nil {
			return "", errors.New("local CA unavailable; install mkcert or configure a trusted server certificate")
		}
		ca, e = os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), "rootCA.pem"))
		if e != nil {
			return "", errors.New("could not read the local CA certificate")
		}
	}
	if !roots.AppendCertsFromPEM(ca) {
		return "", errors.New("invalid local CA certificate")
	}
	if _, err = leaf.Verify(x509.VerifyOptions{DNSName: u.Hostname(), Roots: roots}); err != nil {
		return "", errors.New("local CA does not verify this server certificate")
	}
	bundle := string(ca) + "\n"
	seen := map[string]bool{}
	for _, path := range extras {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		b, e := os.ReadFile(path)
		if e != nil {
			return "", errors.New("could not preserve an existing client CA bundle")
		}
		if !x509.NewCertPool().AppendCertsFromPEM(b) {
			return "", errors.New("invalid existing client CA bundle")
		}
		bundle += string(b) + "\n"
	}
	return bundle, nil
}
