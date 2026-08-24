package share

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
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
	os := r.URL.Query().Get("os")
	if os == "" {
		os = detectPhoneOS(r.UserAgent())
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(trustBootHTML))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	_, _ = w.Write([]byte(trustPageHTML(os, next)))
}

func detectPhoneOS(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"), strings.Contains(ua, "ipod"):
		return "ios"
	case strings.Contains(ua, "android"):
		return "android"
	default:
		return "other"
	}
}

// First bytes flushed immediately so Safari is not a white void while the
// rest of the wizard (or Tailscale handshake after connect) finishes.
const trustBootHTML = `<!doctype html>
<html lang="en"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<title>PiCode</title>
<style>
#boot{position:fixed;inset:0;display:grid;place-items:center;gap:12px;
 background:#f4f5f7;color:#5b6472;font:15px/1.4 -apple-system,sans-serif;z-index:20}
#boot b{width:28px;height:28px;border:3px solid #dfe3ea;border-top-color:#2f6fed;
 border-radius:50%;animation:spin .7s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
</style></head><body>
<div id="boot"><b></b><span>Opening…</span></div>
`

func trustPageHTML(os, next string) string {
	n := htmlEscape(next)
	q := ""
	if next != "" {
		q = "&next=" + url.QueryEscape(next)
	}
	return `<style>
:root{--bg:#f4f5f7;--card:#fff;--fg:#16181d;--muted:#5b6472;--accent:#2f6fed;--line:#e6e9ef}
*{box-sizing:border-box;margin:0}
body{font:16px/1.45 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:var(--bg);color:var(--fg);min-height:100dvh;padding:28px 20px 40px}
main{max-width:22rem;margin:0 auto}
.brand{font-weight:700;letter-spacing:.02em;margin-bottom:22px}
.steps{display:flex;gap:8px;margin-bottom:22px}
.steps i{flex:1;height:4px;border-radius:99px;background:var(--line)}
.steps i.on{background:var(--accent)}
h1{font-size:1.35rem;margin-bottom:8px}
.lead{color:var(--muted);margin-bottom:22px}
ol{padding-left:1.2rem;color:var(--muted);margin:0 0 22px}
ol li{margin:8px 0}
.btn{display:block;width:100%;text-align:center;padding:14px 16px;border-radius:12px;border:0;font:inherit;font-weight:600;cursor:pointer;text-decoration:none}
.btn-pri{background:var(--accent);color:#fff}
.btn-sec{background:var(--card);color:var(--fg);border:1px solid var(--line);margin-top:10px}
.card{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:14px 16px;margin-bottom:18px}
.card b{display:block;margin-bottom:4px}
.hidden{display:none}
.switch{margin-top:18px;text-align:center}
.switch a{color:var(--muted);font-size:13px}
</style>
<section id="gate" class="hidden">
  <div class="brand">PiCode</div>
  <h1 id="gate-title">Open in Safari</h1>
  <p class="lead" id="gate-lead">This step only works in Safari on iPhone.</p>
  <a class="btn btn-pri" id="gate-go" href="#">Open in Safari</a>
  <p class="lead" id="gate-hint">Or tap Share → Open in Safari.</p>
</section>
<main data-os="` + os + `" id="wiz">
  <div class="brand">PiCode</div>
  <div class="steps" id="bar"><i class="on"></i><i></i><i></i></div>

  <section id="s1">
    <h1>Install trust</h1>
    <p class="lead ios">Safari will ask to allow a profile. That’s the local certificate — one time.</p>
    <p class="lead android">Download the certificate, then install it as a CA.</p>
    <p class="lead other">Choose your phone.</p>
    <a class="btn btn-pri ios" href="/picode-ca.mobileconfig" id="dl-ios">Allow profile</a>
    <p class="lead ios">If nothing happens, tap Share → Open in Safari.</p>
    <a class="btn btn-pri android" href="/rootCA.cer" id="dl-and">Download certificate</a>
    <div class="other">
      <a class="btn btn-pri" href="?os=ios` + q + `">iPhone</a>
      <a class="btn btn-sec" href="?os=android` + q + `">Android</a>
    </div>
    <button class="btn btn-sec" type="button" data-go="2">I installed it</button>
  </section>

  <section id="s2" class="hidden">
    <h1>Turn it on</h1>
    <div class="card ios">
      <b>Settings → General → VPN & Device Management</b>
      tap the PiCode profile → Install
    </div>
    <div class="card ios">
      <b>Settings → General → About → Certificate Trust Settings</b>
      enable <b>PiCode mkcert</b>
    </div>
    <div class="card android">
      <b>Settings → Security → Encryption & credentials</b>
      Install a certificate → CA certificate → the file you downloaded
    </div>
    <button class="btn btn-pri" type="button" data-go="3">It’s enabled</button>
    <button class="btn btn-sec" type="button" data-go="1">Back</button>
  </section>

  <section id="s3" class="hidden">
    <h1>Open PiCode</h1>
    <p class="lead">The padlock should be gone.</p>
    ` + openBtn(n) + `
    <button class="btn btn-sec" type="button" data-go="2">Back</button>
  </section>

  <p class="switch"><a href="?os=ios` + q + `">iPhone</a> · <a href="?os=android` + q + `">Android</a></p>
</main>
<script>
(function(){
  var ua=navigator.userAgent;
  var ios=/iPhone|iPad|iPod/.test(ua);
  var and=/Android/.test(ua);
  var safari=ios&&/Safari/.test(ua)&&!/CriOS|FxiOS|EdgiOS|OPiOS|DuckDuckGo/.test(ua);
  var chrome=and&&/Chrome/.test(ua)&&!/EdgA|OPR|SamsungBrowser|Firefox/.test(ua);
  var boot=document.getElementById("boot"); if(boot) boot.remove();
  if((ios&&!safari)||(and&&!chrome)){
    var gate=document.getElementById("gate");
    var wiz=document.getElementById("wiz");
    gate.classList.remove("hidden");
    if(wiz) wiz.classList.add("hidden");
    if(and){
      document.getElementById("gate-title").textContent="Open in Chrome";
      document.getElementById("gate-lead").textContent="This step works in Chrome on Android.";
      document.getElementById("gate-go").textContent="Open in Chrome";
      document.getElementById("gate-hint").textContent="Or copy the link into Chrome.";
    }
    document.getElementById("gate-go").addEventListener("click", function(e){
      e.preventDefault();
      var here=location.href;
      if(ios){
        location.href=here.replace(/^http:/,"x-safari-http:").replace(/^https:/,"x-safari-https:");
        return;
      }
      var u=new URL(here);
      location.href="intent://"+u.host+u.pathname+u.search+"#Intent;scheme="+u.protocol.replace(":","")+";package=com.android.chrome;S.browser_fallback_url="+encodeURIComponent(here)+";end";
    });
    return;
  }
  var os=document.querySelector("main").dataset.os||"other";
  document.querySelectorAll(".ios,.android,.other").forEach(function(el){
    if(!el.classList.contains(os)) el.classList.add("hidden");
  });
  var bar=document.querySelectorAll("#bar i");
  function show(n){
    [1,2,3].forEach(function(i){
      document.getElementById("s"+i).classList.toggle("hidden", i!==n);
      bar[i-1].classList.toggle("on", i<=n);
    });
    history.replaceState(null,"","#"+n);
  }
  document.querySelectorAll("[data-go]").forEach(function(b){
    b.addEventListener("click", function(){ show(+b.getAttribute("data-go")); });
  });
  var h=+(location.hash||"#1").slice(1); if(h>=1&&h<=3) show(h);
})();
</script>
</body></html>`
}

func openBtn(next string) string {
	if next == "" {
		return `<p class="lead">Return to the QR on your computer when this is done.</p>`
	}
	return `<a class="btn btn-pri" href="` + next + `">Open PiCode</a>`
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
