package tlsutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// A second leaf, issued by Tailscale for this node's MagicDNS name
// (box.tailxxxx.ts.net), signed by a public CA — so a phone on the
// tailnet needs no certificate installed (ADR-0050, B.2). It lives next
// to the mkcert/self-signed pair and is chosen per handshake by the name
// the client asked for (see LiveConfig). Without Tailscale, or when
// issuance fails, nothing here runs and the mkcert path is unchanged.
const (
	TailscaleCertFile = "tailscale-cert.pem"
	TailscaleKeyFile  = "tailscale-key.pem"
)

// RenewWithin is how close to expiry the Tailscale leaf is reissued.
// Let's Encrypt leaves last 90 days; Tailscale itself renews well
// before, so a daily check with this margin never serves a stale one.
const RenewWithin = 30 * 24 * time.Hour

// tailscaleCmd is the process boundary; tests replace it.
var tailscaleCmd = func(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "tailscale", args...).CombinedOutput()
}

// TailscaleState describes the leaf on disk for a given name.
type TailscaleState struct {
	Present  bool
	Covers   bool // the leaf's SANs include name
	NotAfter time.Time
}

// TailscaleLeaf reports what is on disk for name.
func TailscaleLeaf(dataDir, name string) TailscaleState {
	pair, err := tls.LoadX509KeyPair(filepath.Join(dataDir, TailscaleCertFile), filepath.Join(dataDir, TailscaleKeyFile))
	if err != nil || len(pair.Certificate) == 0 {
		return TailscaleState{}
	}
	crt, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return TailscaleState{}
	}
	st := TailscaleState{Present: true, NotAfter: crt.NotAfter}
	st.Covers = name != "" && crt.VerifyHostname(name) == nil
	return st
}

// NeedsTailscaleIssue says whether the leaf for name is missing, for
// another name, or inside the renewal window.
func NeedsTailscaleIssue(dataDir, name string) bool {
	if name == "" {
		return false
	}
	st := TailscaleLeaf(dataDir, name)
	return !st.Present || !st.Covers || time.Until(st.NotAfter) < RenewWithin
}

// IssueTailscale asks tailscaled for a certificate for name and writes
// the pair (key 0600). The error carries tailscale's own words — "sudo
// tailscale set --operator=$USER" is the usual first one.
func IssueTailscale(ctx context.Context, dataDir, name string) error {
	if name == "" {
		return fmt.Errorf("tailscale: no MagicDNS name")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	certPath := filepath.Join(dataDir, TailscaleCertFile)
	keyPath := filepath.Join(dataDir, TailscaleKeyFile)
	out, err := tailscaleCmd(ctx, "cert", "--cert-file", certPath, "--key-file", keyPath, name)
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("tailscale cert: %s", firstLine(msg))
	}
	_ = os.Chmod(keyPath, 0o600)
	if st := TailscaleLeaf(dataDir, name); !st.Present || !st.Covers {
		return fmt.Errorf("tailscale cert wrote a certificate that does not cover %s", name)
	}
	return nil
}

// KeepTailscaleCert issues the leaf when needed and re-checks every
// interval until ctx ends. nameFn is asked each time: a node can join a
// tailnet after PiCode started. Failures are logged, never fatal — the
// mkcert path keeps serving.
func KeepTailscaleCert(ctx context.Context, dataDir string, nameFn func() string, every time.Duration) {
	if _, err := exec.LookPath("tailscale"); err != nil {
		return
	}
	lastErr := ""
	check := func() {
		name := nameFn()
		if name == "" || !NeedsTailscaleIssue(dataDir, name) {
			return
		}
		cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := IssueTailscale(cctx, dataDir, name); err != nil {
			if err.Error() != lastErr { // say it once, not once a day
				log.Printf("tls: %v — the tailnet name is served with the local certificate until this works", err)
				lastErr = err.Error()
			}
			return
		}
		lastErr = ""
		log.Printf("tls: Tailscale certificate for %s ready (%s)", name, TailscaleLeaf(dataDir, name).NotAfter.Format("2006-01-02"))
	}
	check()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			check()
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
