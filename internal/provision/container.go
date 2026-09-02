package provision

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/install"
)

// A member inside a systemd-nspawn container (ADR-0052 D.2): same
// account, same home, same gateway; a root filesystem of their own, a
// user namespace, dropped capabilities and cgroup limits around their
// daemon. Firecracker/VMs are the next tier, not this one.

// MachinesDir is where nspawn keeps root filesystems.
const MachinesDir = "/var/lib/machines"

// RootfsFor is a member's container root.
func RootfsFor(user string) string { return filepath.Join(MachinesDir, "picode-"+user) }

// hostSuite reads the distro's codename for debootstrap; tests replace it.
var hostSuite = func() string {
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "stable"
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "VERSION_CODENAME="); ok {
			return strings.Trim(v, "\"")
		}
	}
	return "stable"
}

// ContainerSteps prepares a member whose daemon runs in a container.
func ContainerSteps() []Step {
	return []Step{containerToolsStep(), accountStep(), lingerStep(), binaryStep(), rootfsStep(), memberEnvStep(), containerUnitStep(), memberHealthStep()}
}

func containerToolsStep() Step {
	return Step{
		ID:    "container-tools",
		Title: "systemd-nspawn and debootstrap",
		Scope: ScopeRoot,
		Check: func(Env) State {
			for _, bin := range []string{"systemd-nspawn", "debootstrap"} {
				if _, err := lookPath(bin); err != nil {
					return blocked("%s not found — `apt install systemd-container debootstrap`", bin)
				}
			}
			return ok("present")
		},
		Fix: func(Env) error { return fmt.Errorf("install with: apt install systemd-container debootstrap") },
	}
}

// rootfsStep builds the member's root once: a minimal Debian/Ubuntu of
// the host's release, plus what pi needs. Minutes and network the first
// time; a second run finds it and does nothing.
func rootfsStep() Step {
	return Step{
		ID:    "rootfs",
		Title: "container root filesystem",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			root := RootfsFor(env.User)
			if _, err := os.Stat(filepath.Join(root, "usr", "bin", "tmux")); err != nil {
				if _, err := os.Stat(root); err == nil {
					return needsFix("%s exists but is incomplete", root)
				}
				return needsFix("no root at %s — debootstrap %s", root, hostSuite())
			}
			return ok("%s", root)
		},
		Fix: func(env Env) error {
			root := RootfsFor(env.User)
			if err := os.MkdirAll(MachinesDir, 0o755); err != nil {
				return err
			}
			if _, err := os.Stat(filepath.Join(root, "bin")); err != nil {
				if err := run("debootstrap", "--variant=minbase", "--include=ca-certificates,curl,git,tmux,nodejs,npm,procps", hostSuite(), root); err != nil {
					return fmt.Errorf("debootstrap: %w", err)
				}
			}
			// pi inside the container; the account inside mirrors the host's.
			if err := run("systemd-nspawn", "--quiet", "--directory="+root, "--", "npm", "install", "-g", "@earendil-works/pi-coding-agent"); err != nil {
				return fmt.Errorf("pi in the container: %w", err)
			}
			u, err := lookupAccount(env.User)
			if err != nil {
				return err
			}
			_ = run("systemd-nspawn", "--quiet", "--directory="+root, "--", "useradd", "-M", "-u", u.Uid, "-s", "/bin/bash", env.User)
			return nil
		},
	}
}

func containerUnitStep() Step {
	return Step{
		ID:    "container-unit",
		Title: "member container unit",
		Scope: ScopeRoot,
		Check: func(env Env) State {
			path := install.ContainerUnitPath(env.User)
			name, err := gatewayHostname()
			if err != nil {
				return blocked("no gateway config — run `picode gateway install` first")
			}
			want := install.ContainerUnitFile(env.User, RootfsFor(env.User), env.Home, MemberBin, MemberEnv(name))
			have, err := os.ReadFile(path)
			if err != nil || string(have) != want {
				return needsFix("%s is missing or stale", path)
			}
			if active, _ := output("systemctl", "is-active", "picode-"+env.User); strings.TrimSpace(active) != "active" {
				return needsFix("unit present but not running")
			}
			return ok("picode-%s running", env.User)
		},
		Fix: func(env Env) error {
			name, err := gatewayHostname()
			if err != nil {
				return err
			}
			unit := install.ContainerUnitFile(env.User, RootfsFor(env.User), env.Home, MemberBin, MemberEnv(name))
			if err := os.WriteFile(install.ContainerUnitPath(env.User), []byte(unit), 0o644); err != nil {
				return err
			}
			for _, args := range [][]string{{"daemon-reload"}, {"enable", "--now", "picode-" + env.User}} {
				if err := run("systemctl", args...); err != nil {
					return fmt.Errorf("systemctl %s: %w", strings.Join(args, " "), err)
				}
			}
			return nil
		},
	}
}

// RemoveContainer undoes ContainerSteps: unit, root filesystem. The
// account and home (the member's data) stay; that is `userdel -r`.
func RemoveContainer(user string) error {
	_ = run("systemctl", "disable", "--now", "picode-"+user)
	if err := os.Remove(install.ContainerUnitPath(user)); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = run("systemctl", "daemon-reload")
	return os.RemoveAll(RootfsFor(user))
}
