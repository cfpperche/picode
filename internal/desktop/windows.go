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

// TaskName is the scheduled task that starts the tray at logon. The trigger is
// logon and not boot because WSL needs a user session — there is no way to
// bring a distro up before someone signs in (ADR-0020).
const TaskName = "PiCodeDesktop"

// TaskCreateArgs registers the logon task. /rl limited is deliberate: the tray
// must run unelevated, or it cannot talk to Explorer's notification area and
// the browser it opens would inherit administrator rights. Elevation is only
// for install. /f makes re-registering idempotent.
func TaskCreateArgs(exePath string) []string {
	return []string{
		"/create",
		"/tn", TaskName,
		"/tr", `"` + exePath + `" --tray`,
		"/sc", "onlogon",
		"/rl", "limited",
		"/f",
	}
}

// TaskDeleteArgs removes the logon task.
func TaskDeleteArgs() []string {
	return []string{"/delete", "/tn", TaskName, "/f"}
}

// TaskQueryArgs asks whether the logon task exists.
func TaskQueryArgs() []string {
	return []string{"/query", "/tn", TaskName}
}

// KeepaliveArgs holds the distro open. WSL shuts an idle VM down (vmIdleTimeout
// defaults to 60s), and setting that to -1 is reported as unreliable across
// Windows builds — a live child process is the deterministic answer.
func KeepaliveArgs(distro string) []string {
	return WSLArgs(distro, "", "/bin/sleep", "infinity")
}

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
	out, err := r.Output(WSLExe, WSLArgs(distro, user, "cat", "$HOME/.picode/server.json")...)
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
	if _, err := exec.LookPath("wsl.exe"); err == nil {
		return "wsl.exe"
	}
	for _, p := range []string{
		"/mnt/c/Windows/System32/wsl.exe",
		"/mnt/c/Windows/system32/wsl.exe",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "wsl.exe"
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

// caReadCommand streams the mkcert root out of the distro. Reading it through
// wsl.exe means nothing has to guess the Windows account name, which is what
// scripts/setup-cert.sh does when it copies into /mnt/c/Users/<name>.
var caReadCommand = []string{"sh", "-c", `cat "$(mkcert -CAROOT)/rootCA.pem"`}

// ExportCA writes the distro's mkcert root to a file Windows can import.
func ExportCA(r Runner, distro, user string) (string, error) {
	out, err := r.Output(WSLExe, WSLArgs(distro, user, caReadCommand...)...)
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
