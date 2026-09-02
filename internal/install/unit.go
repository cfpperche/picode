package install

import (
	"fmt"
	"sort"
	"strings"
)

const UnitName = "picode.service"

// GatewayUnitName is the system unit of the shared box's front door (ADR-0051).
const GatewayUnitName = "picode-gateway.service"

// GatewayUnitPath is where the system unit lives.
const GatewayUnitPath = "/etc/systemd/system/" + GatewayUnitName

// ContainerUnitFile is a member's PiCode inside a systemd-nspawn
// container (ADR-0052 D.2): own root filesystem under /var/lib/machines,
// the member's home bound in, the shared binary bound read-only, a
// private user namespace, cgroup limits; host networking on purpose so
// the daemon's loopback port is the host's and the gateway needs no
// change. env is the member environment (MemberEnv).
func ContainerUnitFile(user, rootfs, home, bin string, env map[string]string) string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=PiCode for %s (container)\n", user))
	b.WriteString("After=network-online.target\nWants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	for _, k := range keys {
		b.WriteString("Environment=" + quoteEnv(k+"="+env[k]) + "\n")
	}
	b.WriteString("ExecStart=/usr/bin/systemd-nspawn --quiet --keep-unit --as-pid2")
	b.WriteString(" --directory=" + rootfs)
	b.WriteString(" --bind=" + home)
	b.WriteString(" --bind-ro=" + bin)
	b.WriteString(" --user=" + user)
	b.WriteString(" --private-users=pick --private-users-ownership=map")
	b.WriteString(" --capability=CAP_NET_BIND_SERVICE")
	b.WriteString(" --setenv=HOME=" + home)
	for _, k := range keys {
		b.WriteString(" --setenv=" + k)
	}
	b.WriteString(" " + bin + "\n")
	b.WriteString("CPUQuota=200%\nMemoryMax=4G\nTasksMax=512\n")
	b.WriteString("Restart=on-failure\nRestartSec=3\n")
	b.WriteString("\n[Install]\nWantedBy=multi-user.target\n")
	return b.String()
}

// ContainerUnitPath is the system unit for a member's container.
func ContainerUnitPath(user string) string { return "/etc/systemd/system/picode-" + user + ".service" }

// GatewayUnitFile is the system unit: root (it binds :443, runs
// `tailscale cert`, and reads members' server.json/token), restarts on
// failure, starts after Tailscale is up.
func GatewayUnitFile(execPath, configPath string) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=PiCode gateway (shared box front door)\n")
	b.WriteString("After=network-online.target tailscaled.service\n")
	b.WriteString("Wants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString(fmt.Sprintf("ExecStart=%s gateway --config %s\n", execPath, configPath))
	b.WriteString("Restart=on-failure\nRestartSec=3\n")
	b.WriteString("\n[Install]\nWantedBy=multi-user.target\n")
	return b.String()
}

// UnitFile is the systemd user unit. execPath is the installed binary.
// pathEnv is the PATH snapshot so nvm/`pi` still resolve after install.
func UnitFile(execPath, pathEnv, home string) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=PiCode (browser ADE for Pi agents)\n")
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString(fmt.Sprintf("ExecStart=%s\n", execPath))
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=3\n")
	if home != "" {
		b.WriteString(fmt.Sprintf("Environment=HOME=%s\n", home))
	}
	if pathEnv != "" {
		b.WriteString(fmt.Sprintf("Environment=PATH=%s\n", pathEnv))
	}
	b.WriteString("\n[Install]\n")
	b.WriteString("WantedBy=default.target\n")
	return b.String()
}
