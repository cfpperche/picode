// Package share diagnoses whether a phone can open this PiCode instance
// (HTTPS + reachable address + cert SAN + mkcert CA).
package share

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os/exec"
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

// Target is one candidate URL a phone might use.
type Target struct {
	URL    string `json:"url"`
	Kind   string `json:"kind"` // lan | tailnet
	Addr   string `json:"addr"`
	OnCert bool   `json:"onCert"`
	Reason string `json:"reason,omitempty"`
}

// Report is the payload for GET /api/share.
type Report struct {
	Ready   bool     `json:"ready"`
	URL     string   `json:"url,omitempty"`
	URLs    []string `json:"urls"`
	Targets []Target `json:"targets"`
	Checks  []Check  `json:"checks"`
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
	rep := Report{URLs: []string{}, Targets: []Target{}}
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
	tsOfficial := officialTailscale()

	var anyCovered bool
	for _, a := range addrs {
		kind := "lan"
		if isTailnet(net.ParseIP(a)) {
			kind = "tailnet"
		}
		onCert := !httpsOK || hasSAN(sans, a)
		if onCert {
			anyCovered = true
		}
		t := Target{
			URL:    fmt.Sprintf("%s://%s:%d/", scheme, a, in.Port),
			Kind:   kind,
			Addr:   a,
			OnCert: onCert,
		}
		if !onCert {
			t.Reason = "Certificate does not cover " + a + " — run make cert"
		}
		rep.Targets = append(rep.Targets, t)
		rep.URLs = append(rep.URLs, t.URL)
	}

	rep.Checks = append(rep.Checks, Check{
		ID: "san", OK: anyCovered || !httpsOK, Title: "Certificate covers a current address",
		Action: unless(anyCovered || !httpsOK, "Run make cert (Wi-Fi/LAN IP changed) and restart"),
	})

	mkcertOK := httpsOK && issuerMKCert(issuer)
	rep.Checks = append(rep.Checks, Check{
		ID: "ca", OK: mkcertOK, Title: "Trusted local CA (mkcert)",
		Action: unless(mkcertOK, "Run make cert, then install the CA on the phone"),
	})

	// Prefer a LAN address that is on the cert (same-Wi-Fi phone, no Tailscale),
	// then the official tailscale IP, then any covered target.
	pick := func(pred func(Target) bool) {
		if rep.URL != "" {
			return
		}
		for _, t := range rep.Targets {
			if t.OnCert && pred(t) {
				rep.URL = t.URL
				return
			}
		}
	}
	pick(func(t Target) bool { return t.Kind == "lan" })
	pick(func(t Target) bool { return t.Kind == "tailnet" && t.Addr == tsOfficial })
	pick(func(t Target) bool { return t.Kind == "tailnet" })
	pick(func(Target) bool { return true })

	rep.Ready = httpsOK && bindOK && reachOK && mkcertOK && rep.URL != ""
	return rep
}

func unless(ok bool, action string) string {
	if ok {
		return ""
	}
	return action
}

func officialTailscale() string {
	out, err := exec.Command("tailscale", "ip", "-4").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ReachableIPv4 lists phone-usable IPv4s on real interfaces (not lo/docker).
func ReachableIPv4() []string {
	var out []string
	seen := map[string]bool{}
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		name := strings.ToLower(ifc.Name)
		if name == "lo" || name == "docker0" || strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "veth") {
			continue
		}
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
	var lan, ts []string
	for _, s := range out {
		if isTailnet(net.ParseIP(s)) {
			ts = append(ts, s)
		} else {
			lan = append(lan, s)
		}
	}
	// LAN first: a phone on the same Wi-Fi has no Tailscale by default.
	return append(lan, ts...)
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
	// Docker / typical bridge: 172.16/12.
	if v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31 {
		return false
	}
	// WSL dummy on lo/mirrored networking.
	if s := ip.String(); s == "10.255.255.254" {
		return false
	}
	return true
}

func isTailnet(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
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
