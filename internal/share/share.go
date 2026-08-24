// Package share diagnoses whether a phone can open this PiCode instance
// (HTTPS + reachable address + cert SAN + mkcert CA).
package share

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/tlsutil"
)

// Check is one readiness row. Action is set only when OK is false.
type Check struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Title  string `json:"title"`
	Action string `json:"action,omitempty"`
}

// Report is the payload for GET /api/share.
type Report struct {
	Ready  bool     `json:"ready"`
	URL    string   `json:"url,omitempty"`
	URLs   []string `json:"urls"`
	Checks []Check  `json:"checks"`
}

// Input is live server state needed to diagnose.
type Input struct {
	Insecure bool
	BindHost string
	Port     int
	DataDir  string
}

// Diagnose returns readiness for a phone-scannable URL.
func Diagnose(in Input) Report {
	rep := Report{URLs: []string{}}
	scheme := "https"
	if in.Insecure {
		scheme = "http"
	}

	httpsOK := !in.Insecure
	rep.Checks = append(rep.Checks, Check{
		ID: "https", OK: httpsOK, Title: "HTTPS",
		Action: unless(httpsOK, "Restart without PICODE_INSECURE=1"),
	})

	bindOK := in.BindHost == "" || in.BindHost == "0.0.0.0" || in.BindHost == "::"
	if ip := net.ParseIP(in.BindHost); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
		bindOK = true
	}
	rep.Checks = append(rep.Checks, Check{
		ID: "bind", OK: bindOK, Title: "Reachable bind",
		Action: unless(bindOK, "Bind 0.0.0.0 so other devices can connect"),
	})

	addrs := ReachableIPv4()
	reachOK := len(addrs) > 0
	rep.Checks = append(rep.Checks, Check{
		ID: "address", OK: reachOK, Title: "Phone-reachable address",
		Action: unless(reachOK, "Join Tailscale or a LAN"),
	})

	sans, issuer := certInfo(in.DataDir)
	coverOK := false
	if httpsOK {
		for _, a := range addrs {
			if hasSAN(sans, a) {
				coverOK = true
				break
			}
		}
	} else {
		coverOK = reachOK // HTTP: no cert to check
	}
	rep.Checks = append(rep.Checks, Check{
		ID: "san", OK: coverOK, Title: "Certificate covers that address",
		Action: unless(coverOK, "Run make cert and restart"),
	})

	mkcertOK := httpsOK && issuerMKCert(issuer)
	rep.Checks = append(rep.Checks, Check{
		ID: "ca", OK: mkcertOK, Title: "Trusted local CA (mkcert)",
		Action: unless(mkcertOK, "Run make cert, then install the CA on the phone"),
	})

	for _, a := range addrs {
		rep.URLs = append(rep.URLs, fmt.Sprintf("%s://%s:%d/", scheme, a, in.Port))
	}
	// Prefer tailnet, then first LAN.
	for _, a := range addrs {
		if isTailnet(net.ParseIP(a)) {
			rep.URL = fmt.Sprintf("%s://%s:%d/", scheme, a, in.Port)
			break
		}
	}
	if rep.URL == "" && len(rep.URLs) > 0 {
		rep.URL = rep.URLs[0]
	}

	rep.Ready = httpsOK && bindOK && reachOK && coverOK && mkcertOK
	return rep
}

func unless(ok bool, action string) string {
	if ok {
		return ""
	}
	return action
}

// ReachableIPv4 lists non-loopback, non-link-local, non-docker IPv4s.
func ReachableIPv4() []string {
	var out []string
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
			if ip == nil || ip.To4() == nil {
				continue
			}
			s := ip.String()
			if seen[s] || !UsablePhoneIP(ip) {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
	}
	// Tailnet first.
	var ts, rest []string
	for _, s := range out {
		if isTailnet(net.ParseIP(s)) {
			ts = append(ts, s)
		} else {
			rest = append(rest, s)
		}
	}
	return append(ts, rest...)
}

// UsablePhoneIP reports whether a phone on LAN/tailnet could route here.
func UsablePhoneIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return false
	}
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	// Docker / typical bridge: 172.16/12 (also RFC1918 — but 172.16-31 is
	// the range setup-cert.sh skips). Keep 10/8 and 192.168/16.
	if v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31 {
		return false
	}
	return true
}

func isTailnet(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	// 100.64.0.0/10
	return v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
}

func certInfo(dataDir string) (sans []string, issuer string) {
	if dataDir == "" {
		return nil, ""
	}
	pair, err := tls.LoadX509KeyPair(filepath.Join(dataDir, tlsutil.CertFile), filepath.Join(dataDir, tlsutil.KeyFile))
	if err != nil || len(pair.Certificate) == 0 {
		return nil, ""
	}
	crt, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, ""
	}
	for _, n := range crt.DNSNames {
		sans = append(sans, n)
	}
	for _, ip := range crt.IPAddresses {
		sans = append(sans, ip.String())
	}
	return sans, crt.Issuer.CommonName + " " + strings.Join(crt.Issuer.Organization, " ")
}

func issuerMKCert(issuer string) bool {
	return strings.Contains(strings.ToLower(issuer), "mkcert")
}

func hasSAN(sans []string, addr string) bool {
	for _, s := range sans {
		if s == addr {
			return true
		}
	}
	return false
}

// DataDirExists is a tiny helper for tests.
func DataDirExists(dir string) bool {
	st, err := os.Stat(dir)
	return err == nil && st.IsDir()
}
