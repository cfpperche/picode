package share

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const trustPort = 8470

var (
	trustMu   sync.Mutex
	trustAddr string
)

// EnsureTrustHTTP serves the mkcert CA over HTTP (phones cannot fetch it
// over the HTTPS we are asking them to trust). Safe to call repeatedly.
func EnsureTrustHTTP() string {
	trustMu.Lock()
	defer trustMu.Unlock()
	if trustAddr != "" {
		return trustAddr
	}
	ca, err := caPEM()
	if err != nil {
		log.Printf("share: no mkcert CA: %v", err)
		return ""
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", trustPort))
	if err != nil {
		ln, err = net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			log.Printf("share: trust http: %v", err)
			return ""
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", serveTrustIndex)
	mux.HandleFunc("GET /rootCA.pem", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Header().Set("Content-Disposition", "attachment; filename=picode-rootCA.pem")
		_, _ = w.Write(ca)
	})
	mux.HandleFunc("GET /rootCA.cer", func(w http.ResponseWriter, r *http.Request) {
		der := pemToDER(ca)
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", "attachment; filename=picode-rootCA.cer")
		_, _ = w.Write(der)
	})
	mux.HandleFunc("GET /picode-ca.mobileconfig", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-apple-aspen-config")
		w.Header().Set("Content-Disposition", "attachment; filename=picode-ca.mobileconfig")
		_, _ = w.Write(mobileconfig(pemToDER(ca)))
	})
	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Printf("share: trust http: %v", err)
		}
	}()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	trustAddr = port
	log.Printf("share: trust HTTP on :%s", port)
	tryOpenWindowsPorts(8445, trustPort)
	return port
}

// TrustURL is the phone-facing HTTP page for a given host IP.
func TrustURL(ip, port string) string {
	if ip == "" || port == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%s/", ip, port)
}

func caPEM() ([]byte, error) {
	out, err := exec.Command("mkcert", "-CAROOT").Output()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), "rootCA.pem"))
}

func pemToDER(p []byte) []byte {
	block, _ := pem.Decode(p)
	if block == nil {
		return p
	}
	return block.Bytes
}

func serveTrustIndex(w http.ResponseWriter, r *http.Request) {
	next := r.URL.Query().Get("next")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>PiCode</title>
<style>
 body{font:15px/1.45 -apple-system,sans-serif;margin:24px;max-width:28rem}
 a{display:block;margin:10px 0;padding:12px 14px;border-radius:10px;
   background:#2f6fed;color:#fff;text-decoration:none;text-align:center}
 .sec{background:#eee;color:#111}
 p{color:#444}
</style>
<p>Install this trust profile, enable it, then open PiCode.</p>
<a href="/picode-ca.mobileconfig">iPhone — install profile</a>
<a href="/rootCA.cer">Android — download certificate</a>
<p>iPhone: Settings → Profile Downloaded → Install, then Settings → General → About → Certificate Trust Settings → enable PiCode.</p>
`)
	if next != "" {
		fmt.Fprintf(w, `<a class="sec" href="%s">Open PiCode</a>`, htmlEscape(next))
	}
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return s
}

func mobileconfig(der []byte) []byte {
	b64 := base64.StdEncoding.EncodeToString(der)
	id1, id2 := uuid(), uuid()
	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>PayloadContent</key><array><dict>
<key>PayloadCertificateFileName</key><string>picode-mkcert.cer</string>
<key>PayloadContent</key><data>%s</data>
<key>PayloadDescription</key><string>Trust PiCode local HTTPS</string>
<key>PayloadDisplayName</key><string>PiCode mkcert</string>
<key>PayloadIdentifier</key><string>dev.picode.mkcert</string>
<key>PayloadType</key><string>com.apple.security.root</string>
<key>PayloadUUID</key><string>%s</string>
<key>PayloadVersion</key><integer>1</integer>
</dict></array>
<key>PayloadDisplayName</key><string>PiCode</string>
<key>PayloadIdentifier</key><string>dev.picode.trust</string>
<key>PayloadRemovalDisallowed</key><false/>
<key>PayloadType</key><string>Configuration</string>
<key>PayloadUUID</key><string>%s</string>
<key>PayloadVersion</key><integer>1</integer>
</dict></plist>
`, b64, id1, id2)
	return []byte(body)
}

func uuid() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
