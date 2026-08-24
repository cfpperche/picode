package share

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// tryOpenWindowsPorts adds inbound allow rules via powershell (WSL host).
// Best-effort: no admin → log and continue. Phones need these ports on LAN.
func tryOpenWindowsPorts(ports ...int) {
	pwsh := windowsPowerShell()
	if pwsh == "" {
		return
	}
	var ps []string
	for _, p := range ports {
		ps = append(ps, fmt.Sprintf("%d", p))
	}
	list := strings.Join(ps, ",")
	script := fmt.Sprintf(
		`$n='PiCode LAN'; if (-not (Get-NetFirewallRule -DisplayName $n -ErrorAction SilentlyContinue)) { New-NetFirewallRule -DisplayName $n -Direction Inbound -Action Allow -Protocol TCP -LocalPort %s | Out-Null }`,
		list,
	)
	out, err := exec.Command(pwsh, "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		log.Printf("share: windows firewall (need Admin once): %v %s", err, strings.TrimSpace(string(out)))
		return
	}
	log.Printf("share: windows firewall allows %s", list)
}

func windowsPowerShell() string {
	for _, c := range []string{"powershell.exe", "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"} {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}
