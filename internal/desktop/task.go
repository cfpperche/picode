package desktop

import (
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// TaskName belongs to the current user's interactive Windows login (ADR-0020).
const TaskName = "PiCodeDesktop"

// The two resident launches the task may point at: the retired Go tray
// (ADR-0020) and the shell (ADR-0142). Anything else is a foreign command
// the task tools must not touch.
const (
	TrayArgs  = "--tray"
	ShellArgs = "--hidden"
)

// ShellExeName is the resident the task moves to.
const ShellExeName = "picode-shell.exe"

// ResidentKind names which resident a task action launches: "tray",
// "shell", or "" for a foreign command. The launch argument decides, not
// the executable name — a renamed copy is still the same resident.
func ResidentKind(args string) string {
	switch args {
	case TrayArgs:
		return "tray"
	case ShellArgs:
		return "shell"
	default:
		return ""
	}
}

// ShellExe finds the shell resident: next to the running tool first (a
// self-contained folder), then the canonical PiCode install folder.
func ShellExe(selfExe, localAppData string) (string, error) {
	candidates := []string{}
	if dir := filepath.Dir(selfExe); dir != "" && dir != "." {
		candidates = append(candidates, filepath.Join(dir, ShellExeName))
	}
	if localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "PiCode", ShellExeName))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("picode-shell.exe was not found next to the tool or in %%LOCALAPPDATA%%\\PiCode — reinstall PiCode Desktop")
}

//go:embed task.ps1
var taskScript string

// TaskStatus distinguishes configured startup from a currently running tray.
// It describes only the Windows boundary; inspecting it never starts WSL.
type TaskStatus struct {
	Schema                     int    `json:"schema"`
	Exists                     bool   `json:"exists"`
	Enabled                    bool   `json:"enabled"`
	State                      int    `json:"state"`
	UserID                     string `json:"userId"`
	CurrentUserID              string `json:"currentUserId"`
	Interactive                bool   `json:"interactive"`
	Limited                    bool   `json:"limited"`
	Logon                      bool   `json:"logon"`
	TriggerLimited             bool   `json:"triggerLimited"`
	Executable                 string `json:"executable"`
	Arguments                  string `json:"arguments"`
	ExecutableExists           bool   `json:"executableExists"`
	ExecutionTimeLimit         string `json:"executionTimeLimit"`
	DisallowStartIfOnBatteries bool   `json:"disallowStartIfOnBatteries"`
	StopIfGoingOnBatteries     bool   `json:"stopIfGoingOnBatteries"`
	RunOnlyIfIdle              bool   `json:"runOnlyIfIdle"`
	StopOnIdleEnd              bool   `json:"stopOnIdleEnd"`
	RunOnlyIfNetworkAvailable  bool   `json:"runOnlyIfNetworkAvailable"`
	MultipleInstances          int    `json:"multipleInstances"`
	RestartCount               int    `json:"restartCount"`
	RestartInterval            string `json:"restartInterval"`
	LastRun                    string `json:"lastRun"`
	LastResult                 int64  `json:"lastResult"`
	Backup                     string `json:"backup"`
}

// TaskError retains the native error so only access denied requests elevation.
type TaskError struct {
	Code    int32
	Message string
	Backup  string
}

func (e *TaskError) Error() string {
	message := fmt.Sprintf("%s (0x%08X)", e.Message, uint32(e.Code))
	if e.Backup != "" {
		message += "; previous definition: " + e.Backup
	}
	return message
}

func TaskAccessDenied(err error) bool {
	var taskErr *TaskError
	return errors.As(err, &taskErr) && uint32(taskErr.Code) == 0x80070005
}

func InspectTask(r Runner) (TaskStatus, error) { return taskOperation(r, "inspect", TaskName, "") }

// InstallTask registers the whole policy in one write, then verifies it.
func InstallTask(r Runner, exe string) (TaskStatus, error) {
	s, err := taskOperation(r, "install", TaskName, exe)
	if err == nil {
		err = s.verifyPolicy()
		if err == nil && !s.Enabled {
			err = fmt.Errorf("startup task was registered but is disabled")
		}
	}
	return s, err
}

// RepairTask preserves the existing action, identity, triggers and opt-out.
// It never launches the resident, provisions WSL, or changes machine-wide
// policy.
func RepairTask(r Runner) (TaskStatus, error) {
	return repairTask(r, TaskName)
}

// RetargetTask moves the task's action to a new resident launch — the
// migration from the Go tray to the shell (ADR-0142) — and converges its
// policy in the same write. It refuses a foreign current action (that is
// install's job, not a silent takeover), an unknown target launch, and a
// target executable that is not an existing absolute file. When the action
// already is the target, it degrades to a policy repair.
func RetargetTask(r Runner, exe, args string) (TaskStatus, error) {
	return retargetTask(r, TaskName, exe, args)
}

func retargetTask(r Runner, name, exe, args string) (TaskStatus, error) {
	if ResidentKind(args) == "" {
		return TaskStatus{}, fmt.Errorf("cannot point startup at %q — the resident launch is --tray or --hidden", args)
	}
	if !filepath.IsAbs(exe) {
		return TaskStatus{}, fmt.Errorf("the resident executable must be an absolute path")
	}
	if st, err := os.Stat(exe); err != nil || st.IsDir() {
		return TaskStatus{}, fmt.Errorf("the resident executable is missing: %s", exe)
	}
	before, err := taskOperation(r, "inspect", name, "")
	if err != nil {
		return before, err
	}
	if !before.CanRetarget() {
		return before, fmt.Errorf("startup task cannot be retargeted safely; inspect its registration before running picode-desktop install")
	}
	if ResidentKind(before.Arguments) == "" {
		return before, fmt.Errorf("startup task runs an unrelated command; inspect it before reinstalling")
	}
	if before.Executable == exe && before.Arguments == args {
		// Already there: a policy repair on the inspected state, not a
		// second look at the task.
		if len(before.PolicyIssues()) == 0 {
			return before, nil
		}
		s, err := taskOperation(r, "repair", name, "")
		if err == nil {
			err = s.verifyPolicy()
		}
		return s, err
	}
	s, err := taskRetargetOperation(r, name, exe, args)
	if err == nil {
		err = s.verifyPolicy()
	}
	return s, err
}

func repairTask(r Runner, name string) (TaskStatus, error) {
	before, err := taskOperation(r, "inspect", name, "")
	if err != nil {
		return before, err
	}
	if len(before.PolicyIssues()) == 0 {
		return before, nil
	}
	if !before.CanRepair() {
		return before, fmt.Errorf("startup task cannot be repaired safely; inspect its registration before running picode-desktop install")
	}
	s, err := taskOperation(r, "repair", name, "")
	if err == nil {
		err = s.verifyPolicy()
	}
	return s, err
}

func taskOperation(r Runner, operation, name, exe string) (TaskStatus, error) {
	return runTaskCall(r, taskArgs(operation, name, exe), operation)
}

func taskRetargetOperation(r Runner, name, exe, args string) (TaskStatus, error) {
	return runTaskCall(r, taskRetargetArgs(name, exe, args), "retarget")
}

func runTaskCall(r Runner, argv []string, operation string) (TaskStatus, error) {
	out, err := r.Output("powershell.exe", argv...)
	var envelope struct {
		TaskStatus
		Exists *bool  `json:"exists"`
		Error  string `json:"error"`
		Code   int32  `json:"code"`
	}
	if parseErr := json.Unmarshal([]byte(strings.TrimSpace(DecodeWindows(out))), &envelope); parseErr != nil {
		if err != nil {
			return TaskStatus{}, fmt.Errorf("%s startup task: %w", operation, err)
		}
		return TaskStatus{}, fmt.Errorf("%s startup task: invalid report: %w", operation, parseErr)
	}
	if envelope.Schema != 1 {
		return TaskStatus{}, fmt.Errorf("%s startup task: unsupported report", operation)
	}
	if envelope.Error != "" {
		return TaskStatus{}, &TaskError{Code: envelope.Code, Message: envelope.Error, Backup: envelope.Backup}
	}
	if err != nil {
		return TaskStatus{}, fmt.Errorf("%s startup task: %w", operation, err)
	}
	if envelope.Exists == nil {
		return TaskStatus{}, fmt.Errorf("%s startup task: incomplete report", operation)
	}
	envelope.TaskStatus.Exists = *envelope.Exists
	return envelope.TaskStatus, nil
}

// CanRepair requires the same identity/shape guarded again at the write boundary.
func (s TaskStatus) CanRepair() bool {
	return s.Exists && s.UserID != "" && s.UserID == s.CurrentUserID &&
		s.Interactive && s.Limited && s.Logon && !s.TriggerLimited &&
		s.Executable != "" && s.ExecutableExists && ResidentKind(s.Arguments) != ""
}

// CanRetarget is the ownership half of CanRepair: the identity and trigger
// shape a retarget preserves, without the action shape it replaces.
func (s TaskStatus) CanRetarget() bool {
	return s.Exists && s.UserID != "" && s.UserID == s.CurrentUserID &&
		s.Interactive && s.Limited && s.Logon && !s.TriggerLimited
}

func taskArgs(operation, name, exe string) []string {
	return taskCallArgs(operation, name, exe, "")
}

func taskRetargetArgs(name, exe, args string) []string {
	return taskCallArgs("retarget", name, exe, args)
}

func taskCallArgs(operation, name, exe, args string) []string {
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "& {\n" + taskScript + "\n} -Operation " + quote(operation) +
		" -Name " + quote(name) + " -ExecutablePath " + quote(exe)
	if args != "" {
		script += " -Arguments " + quote(args)
	}
	units := utf16.Encode([]rune(script))
	encoded := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(encoded[i*2:], unit)
	}
	return []string{"-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(encoded)}
}

// PolicyIssues excludes runtime state and the user's enabled/disabled choice.
func (s TaskStatus) PolicyIssues() []string {
	if !s.Exists {
		return []string{"startup task is not registered"}
	}
	var issues []string
	checks := []struct {
		ok      bool
		message string
	}{
		{s.UserID != "" && s.UserID == s.CurrentUserID, "startup task belongs to another Windows account"},
		{s.Interactive && s.Limited, "startup must use the signed-in account without administrator rights"},
		{s.Logon, "an enabled sign-in trigger is missing"},
		{!s.TriggerLimited, "a startup trigger has its own execution limit"},
		{s.Executable != "" && ResidentKind(s.Arguments) != "", "startup command is not a single resident launch"},
		{s.ExecutableExists, "registered tray executable is missing"},
		{s.ExecutionTimeLimit == "PT0S", "execution limit is " + s.ExecutionTimeLimit + " (expected no limit)"},
		{!s.DisallowStartIfOnBatteries && !s.StopIfGoingOnBatteries, "battery conditions can stop or block the tray"},
		{!s.RunOnlyIfIdle && !s.StopOnIdleEnd, "idle settings differ from the resident-task policy"},
		{!s.RunOnlyIfNetworkAvailable, "a network condition can block the tray"},
		{s.MultipleInstances == 2, "duplicate task launches are not ignored"},
		{s.RestartCount == 3 && s.RestartInterval == "PT1M", "launch retry policy is not three retries one minute apart"},
	}
	for _, check := range checks {
		if !check.ok {
			issues = append(issues, check.message)
		}
	}
	return issues
}

func (s TaskStatus) verifyPolicy() error {
	if issues := s.PolicyIssues(); len(issues) > 0 {
		return fmt.Errorf("startup policy verification failed: %s", strings.Join(issues, "; "))
	}
	return nil
}

func (s TaskStatus) StateName() string {
	if !s.Exists {
		return "not installed"
	}
	switch s.State {
	case 1:
		return "disabled"
	case 2:
		return "queued"
	case 3:
		return "stopped"
	case 4:
		return "running"
	default:
		return "unknown"
	}
}
