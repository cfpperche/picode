package desktop

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TaskDeleteArgs removes the logon task.
func TaskDeleteArgs() []string {
	return []string{"/delete", "/tn", TaskName, "/f"}
}

// TaskRunArgs starts the logon task's resident now. It is the only
// supported way to (re)launch the resident from a shell: a backgrounded exe
// dies with the shell and strands the distro (scripts/desktop-swap.sh).
func TaskRunArgs() []string {
	return []string{"/run", "/tn", TaskName}
}

// The distro keepalive (`wsl.exe -d <distro> -- /bin/sleep infinity`,
// supervised so it cannot outlive its owner) lives in the shell resident
// since ADR-0142; the Go side keeps no copy of that argv.

// CACountArgs counts mkcert roots already trusted by this machine. Import is
// gated on this so a logon does not re-import the CA every time.
func CACountArgs() []string {
	return []string{"-NoProfile", "-NonInteractive", "-Command",
		`(Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -like '*mkcert*').Count`}
}

// CAImportArgs trusts the mkcert root for the whole machine. Unlike
// scripts/setup-cert.sh — which has to bounce through `Start-Process -Verb
// RunAs` from inside WSL and raise a UAC prompt — this runs during install,
// where PiCode Desktop is already elevated.
func CAImportArgs(certPath string) []string {
	return []string{"-NoProfile", "-NonInteractive", "-Command",
		`Import-Certificate -FilePath "` + certPath + `" -CertStoreLocation Cert:\LocalMachine\Root`}
}

// CATrusted reads the output of CACountArgs.
func CATrusted(out []byte) bool {
	text := strings.TrimSpace(DecodeWindows(out))
	return text != "" && text != "0"
}

// ServerURL asks the distro where PiCode is listening. The port is a range
// (8445-8455), so it is read from server.json rather than assumed; the answer
// is cached by the caller and polled over HTTP afterwards, which is far
// cheaper than spawning wsl.exe on a timer.
func ServerURL(r Runner, distro, user string) (string, error) {
	// Through a shell for the `~`: no `$` crosses the wsl.exe boundary
	// intact, so `$HOME` is not spelled even though it happens to resolve.
	out, err := r.Output(WSLExe, WSLArgs(distro, user, "sh", "-c", "cat ~/.picode/server.json")...)
	if err != nil {
		return "", fmt.Errorf("read server.json: %w", err)
	}
	text := DecodeWindows(out)
	start := strings.Index(text, "{")
	if start < 0 {
		return "", fmt.Errorf("server.json is not JSON — has PiCode started?")
	}
	var s struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(text[start:]), &s); err != nil {
		return "", fmt.Errorf("server.json: %w", err)
	}
	if s.URL == "" {
		return "", fmt.Errorf("server.json has no url")
	}
	return s.URL, nil
}

// ResolveWSLExe finds the launcher. On Windows it is on PATH; from inside a
// distro configured with appendWindowsPath=false it is not, so the mount is
// probed — that is how `doctor` runs during development.
func ResolveWSLExe() string {
	return ResolveWindowsTool("wsl.exe", "wsl.exe")
}

// WSLExe itself lives in drive.go, next to the Runner it is called through.

// The Windows console programs the desktop reads the machine with. Like WSLExe
// they are names on Windows and mounts from inside a distro, so the same code
// runs in the tray and in a shell during development.
var (
	RegExe        = "reg"
	FsutilExe     = "fsutil"
	PowerShellExe = "powershell"
)

// ResolveWindowsTools points those three at programs this process can run.
func ResolveWindowsTools() {
	RegExe = ResolveWindowsTool("reg", "reg.exe")
	FsutilExe = ResolveWindowsTool("fsutil", "fsutil.exe")
	PowerShellExe = ResolveWindowsTool("powershell", "WindowsPowerShell/v1.0/powershell.exe")
}

// DeployReady asks the running server whether anyone is mid-turn — the same
// interlock `picode deploy` consults before restarting the daemon, because
// stopping the distro ends the same work. Loopback needs no session
// (internal/auth exempts it), which is exactly why the tray is allowed to ask.
func DeployReady(base string) (ready bool, busy []string, err error) {
	res, err := healthClient.Get(strings.TrimSuffix(base, "/") + "/api/deploy/readiness")
	if err != nil {
		return false, nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("status %s", res.Status)
	}
	var body struct {
		Ready bool `json:"ready"`
		Busy  []struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"busy"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return false, nil, err
	}
	for _, b := range body.Busy {
		name := b.Name
		if name == "" {
			name = b.ID
		}
		busy = append(busy, b.Kind+" "+name)
	}
	return body.Ready, busy, nil
}

// DescribeBusy names the callers the interlock found, for a refusal message.
func DescribeBusy(busy []string) string {
	if len(busy) == 0 {
		return "someone"
	}
	return strings.Join(busy, "; ")
}

// ResolveWindowsTool finds one of them: on PATH on Windows, under System32
// from inside a distro that left appendWindowsPath alone.
func ResolveWindowsTool(name, underSystem32 string) string {
	if _, err := exec.LookPath(name); err == nil {
		return name
	}
	for _, root := range []string{"/mnt/c/Windows/System32", "/mnt/c/Windows/system32"} {
		p := root + "/" + underSystem32
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return name
}

// DefaultUser asks the distro which account it logs in as, so the owner never
// has to spell out something WSL already knows.
func DefaultUser(r Runner, distro string) (string, error) {
	out, err := r.Output(WSLExe, WSLArgs(distro, "", "whoami")...)
	if err != nil {
		return "", fmt.Errorf("read the distro's default user: %w", err)
	}
	name := strings.TrimSpace(DecodeWindows(out))
	if name == "" || strings.ContainsAny(name, " \t") {
		return "", fmt.Errorf("could not read the distro's default user (got %q)", name)
	}
	if name == "root" {
		return "", fmt.Errorf("the distro logs in as root — pass --user to name the account PiCode should use")
	}
	return name, nil
}

// ExportCA writes the distro's mkcert root to a file Windows can import.
// Reading it through wsl.exe means nothing has to guess the Windows account
// name, which is what scripts/setup-cert.sh does when it copies into
// /mnt/c/Users/<name>. Two calls on purpose: substitution never survives
// the wsl.exe boundary, so the CAROOT is read back and joined here.
func ExportCA(r Runner, distro, user string) (string, error) {
	caOut, err := r.Output(WSLExe, WSLArgs(distro, user, "mkcert", "-CAROOT")...)
	if err != nil {
		return "", fmt.Errorf("read the mkcert root (is mkcert installed in the distro?): %w", err)
	}
	caroot := strings.TrimSpace(DecodeWindows(caOut))
	if caroot == "" || strings.ContainsAny(caroot, " \t\r\n") {
		return "", fmt.Errorf("mkcert -CAROOT answered %q", caroot)
	}
	out, err := r.Output(WSLExe, WSLArgs(distro, user, "cat", caroot+"/rootCA.pem")...)
	if err != nil {
		return "", fmt.Errorf("read the mkcert root (is mkcert installed in the distro?): %w", err)
	}
	pem := DecodeWindows(out)
	if !strings.Contains(pem, "BEGIN CERTIFICATE") {
		return "", fmt.Errorf("no mkcert certificate authority in the distro")
	}
	path := filepath.Join(os.TempDir(), "picode-mkcert-rootCA.cer")
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// healthClient accepts PiCode's own certificate. This is a liveness check
// against a known local process, not a trust decision — the browser still gets
// a properly trusted cert through the mkcert CA.
var healthClient = &http.Client{
	Timeout:   3 * time.Second,
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
}

// Health reports whether PiCode answers, and its boot id. A changed boot id
// means the server restarted since the last poll.
func Health(base string) (bootID string, err error) {
	res, err := healthClient.Get(strings.TrimSuffix(base, "/") + "/api/health")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s", res.Status)
	}
	var body struct {
		Status string `json:"status"`
		BootID string `json:"bootId"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.BootID, nil
}
