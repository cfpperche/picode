package install

import (
	"fmt"
	"strings"
)

const UnitName = "picode.service"

// GatewayUnitName is the system unit of the shared box's front door (ADR-0051).
const GatewayUnitName = "picode-gateway.service"

// GatewayUnitPath is where the system unit lives.
const GatewayUnitPath = "/etc/systemd/system/" + GatewayUnitName

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
