package desktop

import (
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

// TaskName belongs to the current user's interactive Windows login (ADR-0020).
const TaskName = "PiCodeDesktop"

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
// It never launches the tray, provisions WSL, or changes machine-wide policy.
func RepairTask(r Runner) (TaskStatus, error) {
	return repairTask(r, TaskName)
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
	out, err := r.Output("powershell.exe", taskArgs(operation, name, exe)...)
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
		s.Executable != "" && s.ExecutableExists && s.Arguments == "--tray"
}

func taskArgs(operation, name, exe string) []string {
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "& {\n" + taskScript + "\n} -Operation " + quote(operation) +
		" -Name " + quote(name) + " -ExecutablePath " + quote(exe)
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
		{s.Executable != "" && s.Arguments == "--tray", "startup command is not a single tray launch"},
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
