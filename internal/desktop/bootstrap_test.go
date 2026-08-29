package desktop

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// The stage machine is the whole clean-machine path. Every row is a state a
// real machine passes through, and getting the order wrong means installing a
// distro before WSL exists, or provisioning into an account that is not there.
func TestNextStage(t *testing.T) {
	ubuntu := Distro{Name: "Ubuntu", State: "Stopped", Version: 2, Default: true}
	legacy := Distro{Name: "Old", State: "Stopped", Version: 1, Default: true}

	tests := []struct {
		name  string
		state MachineState
		want  Stage
	}{
		{
			name:  "a clean machine starts by installing WSL",
			state: MachineState{},
			want:  StageInstallWSL,
		},
		{
			name:  "a pending restart outranks everything below it",
			state: MachineState{WSLPresent: true, RebootPending: true, Distros: []Distro{ubuntu}, DefaultUser: "goat"},
			want:  StageReboot,
		},
		{
			name:  "WSL without a distro installs one",
			state: MachineState{WSLPresent: true},
			want:  StageInstallDistro,
		},
		{
			name:  "a WSL 1 distro does not count",
			state: MachineState{WSLPresent: true, Distros: []Distro{legacy}},
			want:  StageInstallDistro,
		},
		{
			// This is exactly what `--install --no-launch` leaves behind.
			name:  "a distro with only root needs an account",
			state: MachineState{WSLPresent: true, Distros: []Distro{ubuntu}},
			want:  StageCreateUser,
		},
		{
			name:  "a distro logging in as root needs an account too",
			state: MachineState{WSLPresent: true, Distros: []Distro{ubuntu}, DefaultUser: "root"},
			want:  StageCreateUser,
		},
		{
			name:  "a ready machine goes straight to provisioning",
			state: MachineState{WSLPresent: true, Distros: []Distro{ubuntu}, DefaultUser: "goat"},
			want:  StageProvision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NextStage(tt.state); got != tt.want {
				t.Errorf("NextStage = %q, want %q", got, tt.want)
			}
		})
	}
}

// 3010 means the step worked and Windows wants a restart. Reporting it as a
// failure would abort an install that actually succeeded.
func TestRebootRequired(t *testing.T) {
	if RebootRequired(nil) {
		t.Error("no error should not mean a reboot")
	}
	if RebootRequired(errors.New("some other failure")) {
		t.Error("a plain error is not a reboot request")
	}

	if !RebootRequired(exitErr(3010)) {
		t.Error("exit 3010 not recognised as a reboot request")
	}
	if RebootRequired(exitErr(1)) {
		t.Error("exit 1 was read as a reboot request")
	}
	// A POSIX shell truncates the status to 8 bits, so 3010 arrives as 194.
	// Windows does not — this pins that the rule matches the real value and
	// not the truncated one.
	if RebootRequired(exitErr(3010 % 256)) {
		t.Error("the truncated status 194 must not count as a reboot request")
	}
	if got := exec.Command("sh", "-c", "exit 3010").Run(); RebootRequired(got) {
		t.Errorf("a shell cannot report 3010 (it gave %v); the test must not rely on it", got)
	}
	// The check has to survive wrapping, because the runner folds stderr in.
	if !RebootRequired(fmt.Errorf("wsl --install: %w", exitErr(3010))) {
		t.Error("a wrapped 3010 was not recognised")
	}
}

// fakeExit stands in for *exec.ExitError, which cannot be constructed with an
// arbitrary code from outside os/exec.
type fakeExit int

func (f fakeExit) Error() string { return fmt.Sprintf("exit status %d", int(f)) }
func (f fakeExit) ExitCode() int { return int(f) }

func exitErr(code int) error { return fakeExit(code) }

// Plain `wsl --install` would also pull a distro and open its interactive
// account setup, which is the one thing the unattended path cannot survive.
func TestInstallArgsAvoidTheInteractiveSetup(t *testing.T) {
	got := strings.Join(InstallWSLArgs(), " ")
	if !strings.Contains(got, "--no-distribution") {
		t.Errorf("InstallWSLArgs = %q, want --no-distribution", got)
	}

	got = strings.Join(InstallDistroArgs("Ubuntu"), " ")
	for _, want := range []string{"--install", "-d Ubuntu", "--no-launch"} {
		if !strings.Contains(got, want) {
			t.Errorf("InstallDistroArgs = %q, missing %q", got, want)
		}
	}
	if def := strings.Join(InstallDistroArgs(""), " "); !strings.Contains(def, DefaultDistro) {
		t.Errorf("InstallDistroArgs(\"\") = %q, want the default distro", def)
	}
}

// RunOnce, not Run: an install that finishes must not leave something behind
// that fires again at every logon.
func TestRunOnceArgs(t *testing.T) {
	got := strings.Join(RunOnceArgs(`C:\PiCode\picode-desktop.exe`), " ")
	if !strings.Contains(got, `RunOnce`) || strings.Contains(got, `CurrentVersion\Run `) {
		t.Errorf("RunOnceArgs = %q, want the RunOnce key", got)
	}
	if !strings.Contains(got, "install") {
		t.Errorf("RunOnceArgs = %q, want it to resume the install", got)
	}
	if !strings.Contains(got, "/f") {
		t.Errorf("RunOnceArgs = %q, want /f so a retry overwrites", got)
	}
}

func TestLinuxUserName(t *testing.T) {
	tests := []struct {
		windows string
		want    string
	}{
		{"goat", "goat"},
		{"Carlos", "carlos"},
		{"Carlos Perche", "carlos-perche"},
		{"João", "joao"},
		{"MARIA.SILVA", "maria-silva"},
		{"user@domain", "userdomain"},
		{"123abc", "abc"},
		{"", "picode"},
		{"!!!", "picode"},
		{"root", "picode"},
		{"-weird-", "weird"},
		{strings.Repeat("a", 40), strings.Repeat("a", 32)},
	}
	for _, tt := range tests {
		t.Run(tt.windows, func(t *testing.T) {
			got := LinuxUserName(tt.windows)
			if got != tt.want {
				t.Errorf("LinuxUserName(%q) = %q, want %q", tt.windows, got, tt.want)
			}
			if got != "" && !validLinuxName(got) {
				t.Errorf("LinuxUserName(%q) = %q, which Linux would reject", tt.windows, got)
			}
		})
	}
}

func validLinuxName(s string) bool {
	if s == "" || len(s) > 32 {
		return false
	}
	if s[0] >= '0' && s[0] <= '9' || s[0] == '-' {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// Creating the account twice must not fail, because the installer resumes.
func TestCreateUserCommandIsIdempotentAndLeavesThePasswordLocked(t *testing.T) {
	script := strings.Join(CreateUserCommand("goat"), " ")
	if !strings.Contains(script, "id -u goat") {
		t.Error("the account is created without checking whether it exists")
	}
	// A password or passwordless sudo would be a security decision the
	// installer has no standing to make.
	for _, forbidden := range []string{"passwd", "NOPASSWD", "chpasswd"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("the script touches %q — the password must stay locked", forbidden)
		}
	}
	if !strings.Contains(script, "usermod -aG") {
		t.Error("the account is never added to a sudo group")
	}
}

func TestSetDefaultUserCommandKeepsAnExistingSetting(t *testing.T) {
	script := strings.Join(SetDefaultUserCommand("goat"), " ")
	if !strings.Contains(script, "exit 0") {
		t.Error("an existing default= is not detected, so wsl.conf would gain a second one")
	}
	if !strings.Contains(script, "[[:space:]]*\\[user\\]") {
		t.Error("an existing [user] section is not detected")
	}
}

func TestDetectOnAMachineWithoutWSL(t *testing.T) {
	r := &fakeRunner{errs: []error{errors.New("not found"), errors.New("not found")}}
	got := Detect(r, "")
	if got.WSLPresent {
		t.Error("WSL reported as present when both probes failed")
	}
	if NextStage(got) != StageInstallWSL {
		t.Errorf("NextStage = %q, want %q", NextStage(got), StageInstallWSL)
	}
}

func TestDetectOnAMachineWithWSLButNoDistro(t *testing.T) {
	r := &fakeRunner{
		replies: [][]byte{utf16le("WSL version: 2.0.0\n"), nil},
		errs:    []error{nil, errors.New("no distributions")},
	}
	got := Detect(r, "")
	if !got.WSLPresent {
		t.Error("WSL not detected despite --version answering")
	}
	if NextStage(got) != StageInstallDistro {
		t.Errorf("NextStage = %q, want %q", NextStage(got), StageInstallDistro)
	}
}

func TestDescribeCoversEveryStage(t *testing.T) {
	for _, s := range []Stage{StageInstallWSL, StageReboot, StageInstallDistro, StageCreateUser, StageProvision} {
		if got := Describe(s, ""); got == "" || strings.Contains(got, string(s)) {
			t.Errorf("Describe(%q) = %q, want a sentence rather than the stage name", s, got)
		}
	}
}
