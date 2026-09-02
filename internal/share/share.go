// Package share diagnoses whether a phone can open this PiCode instance
// (HTTPS + reachable address + cert SAN + mkcert CA).
package share

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
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
	URL     string `json:"url"`
	Kind    string `json:"kind"` // lan | tailnet
	Addr    string `json:"addr"`
	OnCert  bool   `json:"onCert"`
	Trusted bool   `json:"trusted"` // a public chain: the phone installs nothing (Tailscale leaf, B.2)
	Reason  string `json:"reason,omitempty"`
	Note    string `json:"note,omitempty"` // what the phone needs for this address to work
}

// Report is the payload for GET /api/share.
type Report struct {
	Ready     bool     `json:"ready"`
	Trusted   bool     `json:"trusted"` // the picked URL needs no certificate on the phone
	URL       string   `json:"url,omitempty"`
	URLs      []string `json:"urls"`
	Targets   []Target `json:"targets"`
	Checks    []Check  `json:"checks"`
	TrustURL  string   `json:"trustUrl,omitempty"`
	TrustPort string   `json:"trustPort,omitempty"`
}

// Input is live server state needed to diagnose.
type Input struct {
	Insecure  bool
	BindHost  string
	Port      int
	DataDir   string
	PublicURL string // configured origin (ADR-0050); listed first when set
}

// SyncCert reissues the mkcert leaf when current phone-usable IPs are
// missing from the SAN list. No-op if mkcert is absent or already in sync.
// The running server reloads the files on the next handshake (tlsutil.LiveConfig).
func SyncCert(dataDir string, extra ...string) {
	if dataDir == "" {
		return
	}
	if _, err := exec.LookPath("mkcert"); err != nil {
		return
	}
	want := DesiredNames()
	for _, e := range extra {
		if e = strings.TrimSpace(e); e != "" && !hasSAN(want, e) {
			want = append(want, e)
		}
	}
	sans, issuer := certInfo(dataDir)
	if !issuerMKCert(issuer) {
		return
	}
	if !missingAny(sans, want) {
		return
	}
	certPath := filepath.Join(dataDir, tlsutil.CertFile)
	keyPath := filepath.Join(dataDir, tlsutil.KeyFile)
	args := append([]string{"-cert-file", certPath, "-key-file", keyPath}, want...)
	out, err := exec.Command("mkcert", args...).CombinedOutput()
	if err != nil {
		log.Printf("share: mkcert renew: %v (%s)", err, strings.TrimSpace(string(out)))
		return
	}
	log.Printf("share: renewed certificate SANs: %s", strings.Join(want, " "))
}

// DesiredNames is localhost + picode.local + current phone-usable IPs.
func DesiredNames() []string {
	names := []string{"localhost", "picode.local"}
	seen := map[string]bool{"localhost": true, "picode.local": true}
	for _, a := range ReachableIPv4() {
		if seen[a] {
			continue
		}
		seen[a] = true
		names = append(names, a)
	}
	return names
}

func missingAny(have, want []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	for _, w := range want {
		if !set[w] {
			return true
		}
	}
	return false
}

// Diagnose returns readiness for a phone-scannable URL.
func Diagnose(in Input) Report {
	pubHost := ""
	if u, err := url.Parse(in.PublicURL); err == nil && u.Host != "" {
		pubHost = u.Hostname()
	}
	SyncCert(in.DataDir, pubHost)
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
		ID: "address", OK: reachOK, Title: "An address the phone can use",
		Action: unless(reachOK, "Join Tailscale (works on any network) or the same Wi-Fi as this machine"),
	})

	sans, issuer := certInfo(in.DataDir)
	tsOfficial := officialTailscale()

	var anyCovered bool
	// The tailnet name, served with the Tailscale-issued leaf (B.2): the
	// one address a phone opens with nothing installed.
	if name := MagicDNSName(); name != "" && !in.Insecure {
		st := tlsutil.TailscaleLeaf(in.DataDir, name)
		t := Target{URL: fmt.Sprintf("https://%s:%d/", name, in.Port), Kind: "tailnet", Addr: name, OnCert: st.Present && st.Covers, Trusted: st.Present && st.Covers}
		if t.OnCert {
			t.Note = "Any network, with Tailscale on the phone — nothing to install"
			anyCovered = true
		} else {
			t.Reason = "Tailscale certificate not issued yet — `sudo tailscale set --operator=$USER` once, then `picode provision`"
		}
		rep.Targets = append(rep.Targets, t)
		rep.URLs = append(rep.URLs, t.URL)
	}
	if pub := strings.TrimRight(in.PublicURL, "/"); pub != "" {
		host := pub
		if u, err := url.Parse(pub); err == nil && u.Host != "" {
			host = u.Host
		}
		t := Target{URL: pub + "/", Kind: "public", Addr: host, OnCert: true, Note: "The address you configured for this server"}
		// A public URL on the tailnet name is served with the Tailscale
		// leaf: nothing to install on the phone.
		if u, err := url.Parse(pub); err == nil && !in.Insecure {
			if st := tlsutil.TailscaleLeaf(in.DataDir, u.Hostname()); st.Present && st.Covers {
				t.Trusted = true
				t.Note = "The address you configured — nothing to install"
			}
		}
		rep.Targets = append(rep.Targets, t)
		rep.URLs = append(rep.URLs, pub+"/")
		anyCovered = true
	}
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
		if t.Kind == "tailnet" {
			t.Note = "Any network, with Tailscale on the phone"
		} else {
			t.Note = "Same Wi-Fi as this machine; on Windows the firewall rule must exist"
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
		Action: unless(mkcertOK, "Run make cert, then install the CA on the phone — or use the tailnet name, which needs none"),
	})

	// Prefer the official tailscale IP: it works from any network and needs
	// no firewall rule. A LAN address only works on the same Wi-Fi, behind
	// the Windows firewall rule, and (on WSL) with mirrored networking —
	// three things the phone cannot see failing.
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
	pick(func(t Target) bool { return t.Kind == "public" })
	pick(func(t Target) bool { return t.Trusted })
	pick(func(t Target) bool { return t.Kind == "tailnet" && t.Addr == tsOfficial })
	pick(func(t Target) bool { return t.Kind == "lan" })
	pick(func(t Target) bool { return t.Kind == "tailnet" })
	pick(func(Target) bool { return true })

	for _, t := range rep.Targets {
		if t.URL == rep.URL && t.Trusted {
			rep.Trusted = true // the same URL may be listed twice (public + tailnet name)
		}
	}
	rep.Ready = httpsOK && bindOK && reachOK && (mkcertOK || rep.Trusted) && rep.URL != ""
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

// TailscaleIPv4 is this node's tailnet address, "" without Tailscale.
func TailscaleIPv4() string { return officialTailscale() }

// MagicDNSName is this node's tailnet name (box.tailxxxx.ts.net), ""
// when Tailscale is absent, down, or MagicDNS is off.
func MagicDNSName() string {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return ""
	}
	var st struct {
		BackendState string `json:"BackendState"`
		Self         struct {
			DNSName string `json:"DNSName"`
		} `json:"Self"`
	}
	if json.Unmarshal(out, &st) != nil || st.BackendState != "Running" {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(st.Self.DNSName), ".")
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
	official := officialTailscale()
	var lan, ts []string
	for _, s := range out {
		if isTailnet(net.ParseIP(s)) {
			// Only THIS machine's tailscale IP. Mirrored Windows Tailscale
			// (another node) shows up on eth* and is not where picode listens.
			if official != "" && s == official {
				ts = append(ts, s)
			}
			continue
		}
		lan = append(lan, s)
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
