package provision

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/gateway"
	"github.com/cfpperche/picode/internal/install"
)

// MemberBin is the shared build every member unit runs (installed by
// `picode gateway install`).
const MemberBin = "/usr/local/bin/picode"

// lookupAccount is the account boundary; tests replace it.
var lookupAccount = func(name string) (*user.User, error) { return user.Lookup(name) }

// gatewayHostname reads the box's public name from the gateway config;
// tests replace it.
var gatewayHostname = func() (string, error) {
	cfg, err := gateway.Load(gateway.DefaultConfigPath)
	if err != nil {
		return "", err
	}
	return cfg.Hostname, nil
}

// MemberSteps prepares one person's account and daemon on a shared box
// (ADR-0051), run as root with --user. Every step converges: an account
// that exists, a drop-in that already says the right thing, a daemon
// that already answers — nothing is touched.
func MemberSteps() []Step {
	return []Step{accountStep(), lingerStep(), binaryStep(), memberEnvStep(), memberServiceStep(), memberHealthStep()}
}

// MemberEnv is what a member daemon needs to know before it has a UI:
// loopback only, plain HTTP behind the gateway, everyone pairs, and the
// gateway's name as its public URL.
func MemberEnv(hostname string) map[string]string {
	return map[string]string{
		"PICODE_HOST":       "127.0.0.1",
		"PICODE_INSECURE":   "1",
		"PICODE_AUTH_MODE":  "all",
		"PICODE_PUBLIC_URL": "https://" + hostname,
	}
}

func accountStep() Step {
	return Step{
		ID:    "account",
		Title: "Linux account exists",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			if _, err := lookupAccount(env.User); err == nil {
				return ok("%s exists", env.User)
			}
			return needsFix("no account %q — useradd -m", env.User)
		},
		Fix: func(env Env) error {
			return run("useradd", "-m", "-s", "/bin/bash", env.User)
		},
	}
}

func binaryStep() Step {
	return Step{
		ID:    "binary",
		Title: "shared picode binary",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			if _, err := os.Stat(MemberBin); err != nil {
				return needsFix("%s is missing", MemberBin)
			}
			return ok("%s", MemberBin)
		},
		Fix: func(env Env) error { return install.CopyExe(env.Exe, MemberBin) },
	}
}

func memberEnvStep() Step {
	return Step{
		ID:    "member-env",
		Title: "member daemon environment",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			name, err := gatewayHostname()
			if err != nil {
				return blocked("no gateway config — run `picode gateway install` first")
			}
			have, err := install.ReadEnvDropIn(env.Home)
			if err != nil {
				return needsFix("cannot read the drop-in: %v", err)
			}
			for k, v := range MemberEnv(name) {
				if have[k] != v {
					return needsFix("%s should be %s", k, v)
				}
			}
			return ok("loopback, mode all, public URL https://%s", name)
		},
		Fix: func(env Env) error {
			name, err := gatewayHostname()
			if err != nil {
				return err
			}
			if _, err := install.WriteEnvDropIn(env.Home, MemberEnv(name)); err != nil {
				return err
			}
			// Root wrote into the member's home: hand the files back.
			return run("chown", "-R", env.User+":", filepath.Join(env.Home, ".config"))
		},
	}
}

// memberServiceStep runs the member's own provision pass as the member,
// through their user manager (linger started it). Root cannot ask that
// manager directly, so the check reads what the daemon publishes.
func memberServiceStep() Step {
	return Step{
		ID:    "member-service",
		Title: "member daemon installed and running",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			url, err := serverURL(filepath.Join(env.Home, ".picode"))
			if err != nil {
				return needsFix("no server.json in %s yet", env.Home)
			}
			if err := probeHealth(url); err != nil {
				return needsFix("%s does not answer: %v", url, err)
			}
			return ok("%s answers", url)
		},
		Fix: func(env Env) error {
			u, err := lookupAccount(env.User)
			if err != nil {
				return err
			}
			// The user's pass installs the unit under their own manager;
			// XDG_RUNTIME_DIR is that manager's address. useradd/linger
			// above made sure it exists.
			err = run("runuser", "-u", env.User, "--", "env",
				"HOME="+u.HomeDir, "XDG_RUNTIME_DIR=/run/user/"+u.Uid, "PATH="+strings.TrimSpace(os.Getenv("PATH")),
				MemberBin, "provision")
			if err != nil {
				return fmt.Errorf("provision as %s: %w", env.User, err)
			}
			// The daemon needs a moment to publish server.json.
			for i := 0; i < 10; i++ {
				if _, err := serverURL(filepath.Join(u.HomeDir, ".picode")); err == nil {
					return nil
				}
				time.Sleep(500 * time.Millisecond)
			}
			return nil
		},
	}
}

func memberHealthStep() Step {
	return Step{
		ID:    "member-health",
		Title: "member daemon answers on loopback",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			url, err := serverURL(filepath.Join(env.Home, ".picode"))
			if err != nil {
				return blocked("%v", err)
			}
			if !strings.Contains(url, "127.0.0.1") && !strings.Contains(url, "localhost") {
				return blocked("%s is not on loopback — the member daemon must not face the network", url)
			}
			if err := probeHealth(url); err != nil {
				return blocked("%s: %v", url, err)
			}
			return ok("%s is up, behind the gateway", url)
		},
		Fix: func(Env) error { return fmt.Errorf("the service step above owns this") },
	}
}
