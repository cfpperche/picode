package desktop

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"unicode"
)

// DefaultDistro is what a clean machine gets when nothing else is asked for.
const DefaultDistro = "Ubuntu"

// rebootRequired is Windows' ERROR_SUCCESS_REBOOT_REQUIRED. `wsl --install`
// exits with it after enabling the feature: the work succeeded, it just is not
// live until the machine restarts.
const rebootRequired = 3010

// Stage is the next thing a clean machine needs. Deriving it from observed
// state — rather than tracking progress in a file — means an interrupted
// install simply resumes, and a finished one is a no-op.
type Stage string

const (
	StageInstallWSL    Stage = "install-wsl"
	StageReboot        Stage = "reboot"
	StageInstallDistro Stage = "install-distro"
	StageCreateUser    Stage = "create-user"
	StageProvision     Stage = "provision"
)

// MachineState is what could be observed about this Windows machine.
type MachineState struct {
	WSLPresent    bool
	RebootPending bool
	Distros       []Distro
	// DefaultUser is the distro's login account, empty when the distro has
	// only root — which is what `--no-launch` leaves behind.
	DefaultUser string
}

// NextStage decides what to do next. Order is not negotiable: the feature has
// to exist before a distro can be registered, the distro before an account,
// and the account before provisioning, because every step below the account
// runs as that account.
func NextStage(s MachineState) Stage {
	switch {
	case !s.WSLPresent:
		return StageInstallWSL
	case s.RebootPending:
		return StageReboot
	case !hasUsableDistro(s.Distros):
		return StageInstallDistro
	case s.DefaultUser == "" || s.DefaultUser == "root":
		return StageCreateUser
	default:
		return StageProvision
	}
}

func hasUsableDistro(distros []Distro) bool {
	_, err := Pick(distros, "")
	return err == nil
}

// InstallWSLArgs enables WSL without a distribution. Plain `wsl --install`
// would also pull Ubuntu and drop the user into its interactive account setup;
// installing the distro separately later is what keeps the run unattended.
func InstallWSLArgs() []string { return []string{"--install", "--no-distribution"} }

// UpdateWSLArgs refreshes an existing but older WSL.
func UpdateWSLArgs() []string { return []string{"--update"} }

// InstallDistroArgs registers a distribution without launching it. Launching
// is what triggers the username/password prompt, so it is skipped and the
// account is created deliberately afterwards.
func InstallDistroArgs(name string) []string {
	if name == "" {
		name = DefaultDistro
	}
	return []string{"--install", "-d", name, "--no-launch"}
}

// exitCoder is the part of *exec.ExitError this rule needs. Naming it as an
// interface is what makes the rule testable: a POSIX shell truncates an exit
// status to 8 bits (3010 arrives as 194), while Windows returns the full
// 32-bit value — so no real process on this side can produce a 3010 to test
// against.
type exitCoder interface{ ExitCode() int }

// RebootRequired reports whether a command asked for a restart rather than
// failing. Treating 3010 as an error is how an installer reports a step that
// actually succeeded as broken.
func RebootRequired(err error) bool {
	if err == nil {
		return false
	}
	var ec exitCoder
	if errors.As(err, &ec) {
		return ec.ExitCode() == rebootRequired
	}
	return false
}

var _ exitCoder = (*exec.ExitError)(nil)

// RunOnceArgs schedules a resume at the next logon. RunOnce is the right key
// rather than Run: it fires once and deletes itself, so an install that
// finishes never leaves anything behind to fire again.
func RunOnceArgs(exePath string) []string {
	return []string{
		"add", `HKCU\Software\Microsoft\Windows\CurrentVersion\RunOnce`,
		"/v", "PiCodeDesktopSetup",
		"/t", "REG_SZ",
		"/d", `"` + exePath + `" install`,
		"/f",
	}
}

// LinuxUserName turns a Windows account name into a valid Linux one, so the
// owner keeps their name instead of getting something generated. Linux allows
// lowercase letters, digits, underscore and dash, and will not start a name
// with a digit or dash.
func LinuxUserName(windows string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(windows)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		case unicode.IsLetter(r):
			// Fold accented letters rather than dropping them: "joão" should
			// become "joao", not "joo".
			if folded := fold(r); folded != 0 {
				b.WriteRune(folded)
			}
		}
	}
	name := strings.Trim(b.String(), "-_")
	for len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
		name = name[1:]
	}
	if len(name) > 32 {
		name = name[:32]
	}
	if name == "" || name == "root" {
		return "picode"
	}
	return name
}

// folded maps the accented Latin letters a Windows account name is likely to
// carry onto their plain equivalents, so "João" becomes "joao" and not "joo".
var folded = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

func fold(r rune) rune { return folded[r] }

// CreateUserCommand builds the shell run as root inside a fresh distro. It is
// idempotent, and it deliberately leaves the password **locked**: PiCode never
// needs sudo (provisioning reaches root through `wsl -u root`), so setting a
// password — or granting passwordless sudo — would be a security decision the
// installer has no business making on someone's behalf.
func CreateUserCommand(name string) []string {
	script := `set -e
if ! id -u ` + name + ` >/dev/null 2>&1; then
  useradd -m -s /bin/bash ` + name + `
fi
for g in sudo wheel; do
  if getent group "$g" >/dev/null 2>&1; then usermod -aG "$g" ` + name + `; fi
done`
	return []string{"sh", "-c", script}
}

// SetDefaultUserCommand writes [user] default=<name> into the distro's
// wsl.conf. The merge is the same line editor provisioning uses, so a distro
// that already has settings keeps every one of them.
func SetDefaultUserCommand(name string) []string {
	// Done in the distro with a here-doc-free shell so nothing depends on the
	// quoting rules of wsl.exe's argument passing.
	script := `set -e
conf=/etc/wsl.conf
if [ -f "$conf" ] && grep -qi '^[[:space:]]*default[[:space:]]*=' "$conf"; then exit 0; fi
if [ -f "$conf" ] && grep -qi '^[[:space:]]*\[user\]' "$conf"; then
  printf 'default=%s\n' ` + name + ` >> "$conf"
else
  printf '\n[user]\ndefault=%s\n' ` + name + ` >> "$conf"
fi`
	return []string{"sh", "-c", script}
}

// Detect reads what can be observed about this machine. Every probe failing is
// itself information — an absent WSL, an empty distro list, a distro with only
// root — so nothing here is an error; NextStage turns the picture into work.
func Detect(r Runner, preferred string) MachineState {
	var s MachineState

	if _, err := r.Output(WSLExe, "--version"); err != nil {
		// An older WSL has no --version, but --status still answers.
		if _, err := r.Output(WSLExe, "--status"); err != nil {
			return s
		}
	}
	s.WSLPresent = true

	distros, err := ListDistros(r)
	if err != nil {
		return s
	}
	s.Distros = distros

	picked, err := Pick(distros, preferred)
	if err != nil {
		return s
	}
	if user, err := DefaultUser(r, picked.Name); err == nil {
		s.DefaultUser = user
	}
	return s
}

// Describe renders a stage as the sentence the owner needs.
func Describe(stage Stage, distro string) string {
	switch stage {
	case StageInstallWSL:
		return "install WSL (needs a Windows restart)"
	case StageReboot:
		return "restart Windows, then setup resumes on its own"
	case StageInstallDistro:
		if distro == "" {
			distro = DefaultDistro
		}
		return fmt.Sprintf("install the %s distribution", distro)
	case StageCreateUser:
		return "create the Linux account PiCode runs as"
	default:
		return "set PiCode up inside the distribution"
	}
}
