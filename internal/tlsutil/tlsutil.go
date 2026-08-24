// Package tlsutil provides PiCode's TLS bootstrap: load data-dir certs when
// present (mkcert-issued by scripts/setup-cert.sh), else generate a
// self-signed certificate covering localhost + every local IPv4
// (tailnet included). Mirrors the agentdeck-proven pattern (ADR-0007).
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertFile / KeyFile are the conventional names inside the data dir; the
// mkcert script writes the same paths.
const (
	CertFile = "cert.pem"
	KeyFile  = "key.pem"
)

// Ensure returns a usable certificate pair, loading data/cert.pem when it
// exists or generating a self-signed one otherwise.
func Ensure(dataDir string) (tls.Certificate, error) {
	certPath := filepath.Join(dataDir, CertFile)
	keyPath := filepath.Join(dataDir, KeyFile)
	if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		return cert, nil
	}
	return generate(certPath, keyPath)
}

func generate(certPath, keyPath string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "picode"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	tmpl.IPAddresses = append(tmpl.IPAddresses, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))
	tmpl.DNSNames = []string{"localhost", "picode.local"}
	seen := map[string]bool{}
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.To4() == nil || seen[ip.String()] {
				continue
			}
			// Skip link-local and docker bridges — not user-facing routes.
			if ip.IsLinkLocalUnicast() {
				continue
			}
			seen[ip.String()] = true
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		return tls.Certificate{}, err
	}
	if err := writePEM(certPath, "CERTIFICATE", der); err != nil {
		return tls.Certificate{}, err
	}
	if err := writePEM(keyPath, "EC PRIVATE KEY", keyDER); err != nil {
		return tls.Certificate{}, err
	}
	log.Printf("tlsutil: generated self-signed certificate: %s", certPath)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

// WarnIfExpiring logs loudly when the leaf cert is close to expiry.
func WarnIfExpiring(dataDir string, within time.Duration) {
	cert, err := tls.LoadX509KeyPair(filepath.Join(dataDir, CertFile), filepath.Join(dataDir, KeyFile))
	if err != nil || len(cert.Certificate) == 0 {
		return
	}
	crt, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return
	}
	if time.Until(crt.NotAfter) < within {
		log.Printf("WARNING: TLS certificate expires in %.0f days (%s) — run scripts/setup-cert.sh",
			time.Until(crt.NotAfter).Hours()/24, crt.NotAfter.Format("2006-01-02"))
	}
}

// LiveConfig reloads cert.pem/key.pem on every handshake so a renewed
// mkcert file is picked up without rebuilding or restarting.
func LiveConfig(dataDir string) *tls.Config {
	certPath := filepath.Join(dataDir, CertFile)
	keyPath := filepath.Join(dataDir, KeyFile)
	return &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			c, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err != nil {
				return nil, err
			}
			return &c, nil
		},
	}
}

func writePEM(path, blockType string, der []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: der})
}
