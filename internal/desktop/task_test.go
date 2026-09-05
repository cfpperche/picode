package desktop

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf16"
)

func healthyTask() TaskStatus {
	return TaskStatus{
		Schema: 1, Exists: true, Enabled: true, State: 3,
		UserID: "S-1-5-21-123", CurrentUserID: "S-1-5-21-123",
		Interactive: true, Limited: true, Logon: true,
		Executable: `C:\Users\owner\PiCode\picode-desktop.exe`, Arguments: "--tray", ExecutableExists: true,
		ExecutionTimeLimit: "PT0S", MultipleInstances: 2, RestartCount: 3, RestartInterval: "PT1M",
	}
}

func taskJSON(t *testing.T, s TaskStatus) []byte {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTaskPolicyDecisionTable(t *testing.T) {
	tests := []struct {
		name      string
		change    func(*TaskStatus)
		issue     string
		canRepair bool
	}{
		{"healthy stopped", func(*TaskStatus) {}, "", true},
		{"running", func(s *TaskStatus) { s.State = 4; s.LastResult = 0x41301 }, "", true},
		{"disabled choice", func(s *TaskStatus) { s.Enabled = false; s.State = 1 }, "", true},
		{"missing", func(s *TaskStatus) { s.Exists = false }, "not registered", false},
		{"other user", func(s *TaskStatus) { s.UserID = "other" }, "another Windows account", false},
		{"unknown user", func(s *TaskStatus) { s.UserID = "" }, "another Windows account", false},
		{"elevated", func(s *TaskStatus) { s.Limited = false }, "without administrator", false},
		{"noninteractive", func(s *TaskStatus) { s.Interactive = false }, "signed-in account", false},
		{"no enabled logon", func(s *TaskStatus) { s.Logon = false }, "sign-in trigger", false},
		{"trigger limit", func(s *TaskStatus) { s.TriggerLimited = true }, "own execution limit", false},
		{"foreign action", func(s *TaskStatus) { s.Arguments = "install" }, "single tray launch", false},
		{"multiple actions", func(s *TaskStatus) { s.Executable = "" }, "single tray launch", false},
		{"missing exe", func(s *TaskStatus) { s.ExecutableExists = false }, "executable is missing", false},
		{"72 hours", func(s *TaskStatus) { s.ExecutionTimeLimit = "PT72H" }, "PT72H", true},
		{"battery start", func(s *TaskStatus) { s.DisallowStartIfOnBatteries = true }, "battery", true},
		{"battery stop", func(s *TaskStatus) { s.StopIfGoingOnBatteries = true }, "battery", true},
		{"idle start", func(s *TaskStatus) { s.RunOnlyIfIdle = true }, "idle", true},
		{"idle stop", func(s *TaskStatus) { s.StopOnIdleEnd = true }, "idle", true},
		{"network", func(s *TaskStatus) { s.RunOnlyIfNetworkAvailable = true }, "network", true},
		{"duplicate", func(s *TaskStatus) { s.MultipleInstances = 0 }, "duplicate", true},
		{"no launch retry", func(s *TaskStatus) { s.RestartCount = 0 }, "retry", true},
		{"launch retry interval", func(s *TaskStatus) { s.RestartInterval = "PT5M" }, "retry", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := healthyTask()
			tt.change(&s)
			issues := strings.Join(s.PolicyIssues(), "; ")
			if (tt.issue == "" && issues != "") || (tt.issue != "" && !strings.Contains(issues, tt.issue)) {
				t.Fatalf("issues = %q, expected %q", issues, tt.issue)
			}
			if s.CanRepair() != tt.canRepair {
				t.Errorf("CanRepair = %v, want %v", s.CanRepair(), tt.canRepair)
			}
		})
	}
}

func TestInspectTaskBoundary(t *testing.T) {
	tests := []struct {
		name, report string
		runErr       error
		wantErr      bool
		denied       bool
	}{
		{"missing", `{"schema":1,"exists":false}`, nil, false, false},
		{"bad JSON", "Access is denied", nil, true, false},
		{"empty", "", nil, true, false},
		{"unversioned", `{"exists":false}`, nil, true, false},
		{"incomplete", `{"schema":1}`, nil, true, false},
		{"denied", `{"schema":1,"error":"Access is denied","code":-2147024891}`, errors.New("exit 1"), true, true},
		{"other error", `{"schema":1,"error":"RPC unavailable","code":-2147023174}`, errors.New("exit 1"), true, false},
		{"nonzero with JSON", `{"schema":1,"exists":false}`, errors.New("exit 1"), true, false},
		{"runner failure", "", errors.New("powershell unavailable"), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fakeRunner{replies: [][]byte{[]byte(tt.report)}, errs: []error{tt.runErr}}
			_, err := InspectTask(r)
			if (err != nil) != tt.wantErr || TaskAccessDenied(err) != tt.denied {
				t.Fatalf("err = %v, access denied = %v", err, TaskAccessDenied(err))
			}
			if len(r.calls) != 1 || r.calls[0][0] != "powershell.exe" {
				t.Fatalf("unexpected boundary calls: %v", r.calls)
			}
		})
	}
	// Windows commands can return UTF-16 even when a console changes encoding.
	r := &fakeRunner{replies: [][]byte{utf16le(string(taskJSON(t, healthyTask())))}}
	if s, err := InspectTask(r); err != nil || len(s.PolicyIssues()) != 0 {
		t.Fatalf("UTF-16 report = %+v, %v", s, err)
	}
}

func TestTaskArgumentsQuoteDataAndCarryCompletePolicy(t *testing.T) {
	path := `C:\Users\O'Brien\工具 $value\picode-desktop.exe`
	args := taskArgs("install", TaskName, path)
	if len(args) != 4 || args[0] != "-NoProfile" || args[1] != "-NonInteractive" || args[2] != "-EncodedCommand" {
		t.Fatalf("unexpected PowerShell arguments: %v", args[:3])
	}
	b, err := base64.StdEncoding.DecodeString(args[3])
	if err != nil {
		t.Fatal(err)
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	script := string(utf16.Decode(units))
	if !strings.HasSuffix(script, "-Operation 'install' -Name 'PiCodeDesktop' -ExecutablePath 'C:\\Users\\O''Brien\\工具 $value\\picode-desktop.exe'") {
		t.Fatalf("path was not kept literal: %s", script[len(taskScript):])
	}
	for _, policy := range []string{
		"ExecutionTimeLimit = 'PT0S'", "DisallowStartIfOnBatteries = $false",
		"StopIfGoingOnBatteries = $false", "RunOnlyIfIdle = $false",
		"RunOnlyIfNetworkAvailable = $false", "StopOnIdleEnd = $false",
		"RestartCount = 3", "RestartInterval = 'PT1M'", "MultipleInstances = 2",
		"Principal.LogonType = 3", "Principal.RunLevel = 0", "Triggers.Create(9)",
		"$trigger.UserId = $currentSid", "$action.Arguments = '--tray'",
	} {
		if !strings.Contains(script, policy) {
			t.Errorf("complete policy missing %s", policy)
		}
	}
	if len(args[3]) > 30000 {
		t.Fatalf("encoded command is too near Windows command-line limit: %d", len(args[3]))
	}
}

func TestRepairTaskDecisionTable(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*TaskStatus)
		wantErr bool
		calls   int
	}{
		{"already healthy", func(*TaskStatus) {}, false, 1},
		{"healthy disabled", func(s *TaskStatus) { s.Enabled = false }, false, 1},
		{"legacy", func(s *TaskStatus) { s.ExecutionTimeLimit = "PT72H" }, false, 2},
		{"missing", func(s *TaskStatus) { s.Exists = false }, true, 1},
		{"foreign principal", func(s *TaskStatus) { s.UserID = "someone-else" }, true, 1},
		{"elevated", func(s *TaskStatus) { s.Limited = false }, true, 1},
		{"unknown executable", func(s *TaskStatus) { s.ExecutableExists = false }, true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, after := healthyTask(), healthyTask()
			tt.change(&before)
			r := &fakeRunner{replies: [][]byte{taskJSON(t, before), taskJSON(t, after)}}
			_, err := RepairTask(r)
			if (err != nil) != tt.wantErr || len(r.calls) != tt.calls {
				t.Fatalf("err = %v, calls = %d; want err=%v, calls=%d", err, len(r.calls), tt.wantErr, tt.calls)
			}
		})
	}
}

func TestTaskWritesMustVerifyTheReturnedPolicy(t *testing.T) {
	for _, name := range []string{"install", "repair"} {
		t.Run(name, func(t *testing.T) {
			legacy := healthyTask()
			legacy.ExecutionTimeLimit = "PT72H"
			r := &fakeRunner{replies: [][]byte{taskJSON(t, legacy), taskJSON(t, legacy)}}
			var err error
			if name == "install" {
				_, err = InstallTask(r, legacy.Executable)
			} else {
				_, err = RepairTask(r)
			}
			if err == nil || !strings.Contains(err.Error(), "verification failed") {
				t.Fatalf("unverified write accepted: %v", err)
			}
		})
	}
	s := healthyTask()
	s.Enabled = false
	if _, err := InstallTask(&fakeRunner{replies: [][]byte{taskJSON(t, s)}}, s.Executable); err == nil {
		t.Fatal("install claimed success for a disabled task")
	}
}

func TestTaskStateNames(t *testing.T) {
	for state, want := range map[int]string{0: "unknown", 1: "disabled", 2: "queued", 3: "stopped", 4: "running", 99: "unknown"} {
		s := healthyTask()
		s.State = state
		if s.StateName() != want {
			t.Errorf("state %d = %q, want %q", state, s.StateName(), want)
		}
	}
	if (TaskStatus{}).StateName() != "not installed" {
		t.Fatal("missing task was not distinguished from unknown state")
	}
}
