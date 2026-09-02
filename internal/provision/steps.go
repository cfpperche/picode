package provision

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/install"
	"github.com/cfpperche/picode/internal/tlsutil"
)

// BackupSuffix marks the copy provisioning leaves behind before it edits a
// file it did not write.
const BackupSuffix = ".picode.bak"

// renewWithin is how close to expiry a certificate has to be before
// provisioning reissues it. Well short of that and the cert is left alone.
const renewWithin = 30 * 24 * time.Hour

// lookPath is the PATH boundary; tests replace it to pin which branch runs.
var lookPath = exec.LookPath

// Steps is the converging plan, in dependency order: the distro has to boot
// systemd before a user manager can exist, the user manager has to survive
// logout before the unit is worth enabling, and the service has to be up
// before health can mean anything.
func Steps() []Step {
	return []Step{wslConfStep(), systemdStep(), lingerStep(), certStep(), serviceStep(), healthStep(), piStep(), tailnetStep(), reachStep()}
}

// piStep: agents are `pi` processes the unit spawns, so the binary has to
// be on the PATH the unit captured. No fix — installing pi is npm's job
// and the owner's choice of prefix (ADR-0003).
func piStep() Step {
	return Step{
		ID:    "pi",
		Title: "pi is on PATH",
		Scope: ScopeUser,
		Check: func(env Env) State {
			if _, err := lookPath("pi"); err == nil {
				return ok("found")
			}
			return blocked("pi not found — `npm install -g @earendil-works/pi-coding-agent`, then `picode deploy` so the unit sees it")
		},
		Fix: func(Env) error { return fmt.Errorf("install pi with npm, then picode deploy") },
	}
}

// tailnetStep reports whether this box can be reached by its tailnet
// name — the address the guide recommends (ADR-0050). A machine without
// Tailscale is fine (LAN only); one with it installed but down is not.
func tailnetStep() Step {
	return Step{
		ID:    "tailnet",
		Title: "Tailscale is up",
		Scope: ScopeUser,
		Check: func(Env) State {
			if _, err := lookPath("tailscale"); err != nil {
				return ok("no Tailscale — reachable on the LAN only")
			}
			raw, err := output("tailscale", "status", "--json")
			if err != nil {
				return blocked("tailscale is installed but not answering — `tailscale up`")
			}
			var st struct {
				BackendState string `json:"BackendState"`
				Self         struct {
					DNSName      string   `json:"DNSName"`
					TailscaleIPs []string `json:"TailscaleIPs"`
				} `json:"Self"`
			}
			if json.Unmarshal([]byte(raw), &st) != nil || st.BackendState != "Running" {
				return blocked("tailscale is not running (%s) — `tailscale up`", st.BackendState)
			}
			name := strings.TrimSuffix(st.Self.DNSName, ".")
			if name == "" && len(st.Self.TailscaleIPs) > 0 {
				name = st.Self.TailscaleIPs[0]
			}
			return ok("this box is %s on the tailnet", name)
		},
		Fix: func(Env) error { return fmt.Errorf("run `tailscale up` as the owner") },
	}
}

// reachStep is the server-mode question: can another machine reach this
// PiCode at all, and does it know its own public address? Read from
// server.json, so it describes the daemon that is actually running.
func reachStep() Step {
	return Step{
		ID:    "reach",
		Title: "Reachable from other machines",
		Scope: ScopeUser,
		Check: func(env Env) State {
			b, err := os.ReadFile(filepath.Join(env.DataDir, "server.json"))
			if err != nil {
				return blocked("no server.json in %s — PiCode has not started yet", env.DataDir)
			}
			var s struct {
				Bind      string `json:"bind"`
				PublicURL string `json:"publicUrl"`
				Port      int    `json:"port"`
			}
			if json.Unmarshal(b, &s) != nil {
				return blocked("server.json is not readable")
			}
			if s.Bind == "127.0.0.1" || s.Bind == "::1" {
				return blocked("bound to loopback — Preferences → Server → Bind: all interfaces (or PICODE_HOST=0.0.0.0)")
			}
			if s.PublicURL != "" {
				return ok("advertises %s", s.PublicURL)
			}
			if _, err := lookPath("tailscale"); err == nil {
				if ip, err := output("tailscale", "ip", "-4"); err == nil && strings.TrimSpace(ip) != "" {
					return ok("reachable at https://%s:%d — set a public URL so links and clients use one name", strings.TrimSpace(ip), s.Port)
				}
			}
			return ok("reachable on the LAN; no tailnet and no public URL yet")
		},
		Fix: func(Env) error { return fmt.Errorf("set the bind and public URL in Preferences → Server") },
	}
}

// wslConfStep turns systemd on for the distro. It is the only step that writes
// outside the user's home, so it backs the file up and merges by line —
// /etc/wsl.conf is the user's, and it usually carries settings PiCode knows
// nothing about (ADR-0020).
func wslConfStep() Step {
	return Step{
		ID:    "wsl-conf",
		Title: "systemd enabled in " + ConfPath,
		Scope: ScopeRoot,
		Check: func(env Env) State {
			if !env.InWSL {
				return ok("not WSL — no distro config to write")
			}
			content, err := readConf()
			if err != nil {
				return blocked("cannot read %s: %v", ConfPath, err)
			}
			if _, changed := EnsureSystemd(content); !changed {
				return ok("[boot] systemd=true already set")
			}
			return needsFix("[boot] systemd=true is missing")
		},
		Fix: func(env Env) error {
			content, err := readConf()
			if err != nil {
				return err
			}
			merged, changed := EnsureSystemd(content)
			if !changed {
				return nil
			}
			if err := writeBackup(ConfPath, BackupSuffix); err != nil {
				return fmt.Errorf("back up %s: %w", ConfPath, err)
			}
			return writeAtomic(ConfPath, merged, 0o644)
		},
	}
}

// systemdStep has no fix on purpose: once wsl.conf is right, only a distro
// restart can start PID 1, and that costs every running tmux session — so it
// is the owner's call, not the installer's (ADR-0020).
func systemdStep() Step {
	return Step{
		ID:    "systemd",
		Title: "systemd is running",
		Scope: ScopeUser,
		Check: func(env Env) State {
			if _, err := exec.LookPath("systemctl"); err != nil {
				return blocked("systemctl not found — PiCode needs a systemd distro")
			}
			if _, err := os.Stat("/run/systemd/system"); err != nil {
				if env.InWSL {
					return blocked("systemd is not PID 1 — run `wsl --shutdown` on Windows, " +
						"then start the distro again (this ends running tmux sessions)")
				}
				return blocked("systemd is not PID 1")
			}
			return ok("PID 1 is systemd")
		},
		Fix: func(Env) error {
			return fmt.Errorf("only a distro restart can start systemd")
		},
	}
}

// lingerStep is what makes the user unit start with the distro instead of with
// a login. ADR-0020: enabled here, never disabled — linger is shared per-user
// state, so turning it off could break whatever else relies on it.
func lingerStep() Step {
	return Step{
		ID:    "linger",
		Title: "user services start without a login",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			if _, err := os.Stat(lingerPath(env.User)); err == nil {
				return ok("linger is on for %s", env.User)
			}
			return needsFix("linger is off for %s — the unit would wait for a login", env.User)
		},
		Fix: func(env Env) error {
			return run("loginctl", "enable-linger", env.User)
		},
	}
}

// certStep keeps a usable certificate in the data dir. mkcert is preferred
// because a browser trusts it; without mkcert the self-signed pair PiCode
// already knows how to make is enough to serve HTTPS (ADR-0007).
func certStep() Step {
	return Step{
		ID:    "cert",
		Title: "TLS certificate is valid",
		Scope: ScopeUser,
		Check: func(env Env) State {
			path := filepath.Join(env.DataDir, tlsutil.CertFile)
			notAfter, err := certExpiry(path)
			if err != nil {
				return needsFix("no usable certificate in %s", env.DataDir)
			}
			left := time.Until(notAfter)
			if left < renewWithin {
				return needsFix("certificate expires %s", notAfter.Format(time.DateOnly))
			}
			return ok("valid until %s", notAfter.Format(time.DateOnly))
		},
		Fix: func(env Env) error {
			if err := os.MkdirAll(env.DataDir, 0o755); err != nil {
				return err
			}
			if _, err := lookPath("mkcert"); err == nil {
				return issueWithMkcert(env.DataDir)
			}
			// No mkcert: fall back to the self-signed pair. Serves HTTPS, but
			// the browser will warn until scripts/setup-cert.sh runs.
			_, err := tlsutil.Ensure(env.DataDir)
			return err
		},
	}
}

// serviceStep hands the running server to systemd. install.Install stops a
// stray picode first (SIGTERM, then SIGKILL) and never touches the data dir,
// so adoption costs a restart and nothing else (ADR-0018, ADR-0020).
func serviceStep() Step {
	return Step{
		ID:    "service",
		Title: "picode.service installed and enabled",
		Scope: ScopeUser,
		Check: func(env Env) State {
			p := install.ForHome(env.Home)
			if _, err := os.Stat(p.Unit); err != nil {
				return needsFix("no user unit at %s", p.Unit)
			}
			// `systemctl --user` always answers for the *calling* account, so
			// asking it from root reports on root's manager while claiming to
			// describe someone else's. Refusing to answer beats answering
			// wrongly — the pass that runs as the owner reports the truth.
			if env.OnBehalf() {
				return blocked("cannot read %s's services as %s — the pass running as %s reports this",
					env.User, env.Acting, env.User)
			}
			if enabled, _ := output("systemctl", "--user", "is-enabled", install.UnitName); strings.TrimSpace(enabled) != "enabled" {
				return needsFix("unit is present but not enabled")
			}
			if active, _ := output("systemctl", "--user", "is-active", install.UnitName); strings.TrimSpace(active) != "active" {
				return needsFix("unit is enabled but not running")
			}
			return ok("enabled and running")
		},
		Fix: func(env Env) error {
			return install.Install(env.Exe, env.Home, env.PathEnv)
		},
	}
}

// healthStep is the proof. Everything above can succeed while the server still
// fails to bind, so provisioning only claims success once PiCode answers.
func healthStep() Step {
	return Step{
		ID:    "health",
		Title: "PiCode answers /api/health",
		Scope: ScopeUser,
		Check: func(env Env) State {
			url, err := serverURL(env.DataDir)
			if err != nil {
				return blocked("%v", err)
			}
			var last error
			for attempt := 0; attempt < 3; attempt++ {
				if attempt > 0 {
					time.Sleep(time.Second)
				}
				if last = probeHealth(url); last == nil {
					return ok("%s is up", url)
				}
			}
			return blocked("%s did not answer: %v", url, last)
		},
		Fix: func(Env) error {
			return fmt.Errorf("nothing to fix here — the service steps above own this")
		},
	}
}

func readConf() (string, error) {
	b, err := os.ReadFile(ConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// lingerDir is where systemd records which users linger. Reading the file is
// more reliable than asking loginctl, which errors for a user with no session.
var lingerDir = "/var/lib/systemd/linger"

func lingerPath(user string) string {
	return filepath.Join(lingerDir, user)
}

func certExpiry(path string) (time.Time, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return time.Time{}, fmt.Errorf("%s is not PEM", path)
	}
	c, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return c.NotAfter, nil
}

func issueWithMkcert(dataDir string) error {
	args := []string{
		"-cert-file", filepath.Join(dataDir, tlsutil.CertFile),
		"-key-file", filepath.Join(dataDir, tlsutil.KeyFile),
	}
	dns, ips := tlsutil.LocalNames()
	args = append(args, dns...)
	for _, ip := range ips {
		args = append(args, ip.String())
	}
	if err := run("mkcert", args...); err != nil {
		return fmt.Errorf("mkcert: %w", err)
	}
	return nil
}

// serverURL reads the address a running (or last-running) PiCode published.
func serverURL(dataDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dataDir, "server.json"))
	if err != nil {
		return "", fmt.Errorf("no server.json in %s — PiCode has not started yet", dataDir)
	}
	var s struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(b, &s); err != nil || s.URL == "" {
		return "", fmt.Errorf("server.json has no url")
	}
	return s.URL, nil
}

// probeHealth accepts PiCode's own certificate: this is a liveness check
// against a known local process, not a trust decision.
func probeHealth(base string) error {
	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
	}
	res, err := client.Get(strings.TrimSuffix(base, "/") + "/api/health") // health needs no token (ADR-0049)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", res.Status)
	}
	return nil
}
