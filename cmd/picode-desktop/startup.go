package main

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/cfpperche/picode/internal/desktop"
)

func runStartupCheck() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("startup-check runs on Windows")
	}
	status, err := desktop.InspectTask(osRunner{})
	if !printStartupStatus(os.Stdout, status, err) {
		return fmt.Errorf("Windows startup needs attention; no changes made")
	}
	return nil
}

func runStartupRepair() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("startup-repair runs on Windows")
	}
	return repairStartup(osRunner{}, elevate, os.Stdout)
}

func repairStartup(r desktop.Runner, requestElevation func() (bool, error), w io.Writer) error {
	status, err := desktop.RepairTask(r)
	if desktop.TaskAccessDenied(err) {
		if relaunched, elevateErr := requestElevation(); elevateErr != nil {
			return elevateErr
		} else if relaunched {
			fmt.Fprintln(w, "Administrator approval requested. Run startup-check afterward to verify the result.")
			return nil
		}
	}
	if err != nil {
		return err
	}
	if status.Backup == "" {
		fmt.Fprintln(w, "Startup policy already matches; nothing changed.")
	} else {
		fmt.Fprintln(w, "Startup policy repaired. No process was started or stopped.")
		fmt.Fprintf(w, "Previous task definition: %s\n", status.Backup)
	}
	printStartupStatus(w, status, nil)
	return nil
}

// printStartupStatus is shared by doctor and the WSL-independent check command.
// Stopped is not a failed installation: the user may have selected Quit.
func printStartupStatus(w io.Writer, s desktop.TaskStatus, err error) bool {
	if err != nil {
		fmt.Fprintf(w, "  FAIL   Windows startup: cannot inspect task: %v\n", err)
		fmt.Fprintln(w, "         Check access to PiCodeDesktop in Windows Task Scheduler.")
		return false
	}
	if !s.Exists {
		fmt.Fprintln(w, "  todo   Windows startup: task is not registered.")
		fmt.Fprintln(w, "         Run picode-desktop install.")
		return false
	}
	fmt.Fprintf(w, "  info   Tray task: %s\n", s.StateName())
	if s.Enabled {
		fmt.Fprintln(w, "  ok     Startup task enabled")
	} else {
		fmt.Fprintln(w, "  todo   Startup task disabled; its enabled/disabled choice was preserved.")
		fmt.Fprintln(w, "         Enable PiCodeDesktop in Windows Task Scheduler if you want startup at sign-in.")
	}
	issues := s.PolicyIssues()
	if len(issues) == 0 {
		fmt.Fprintln(w, "  ok     Startup policy: sign-in, limited user, no time/battery/idle/network limits")
		fmt.Fprintln(w, "  ok     Launch retry policy: 3 retries, 1 minute apart; runtime crash recovery is not guaranteed")
	} else {
		for _, issue := range issues {
			fmt.Fprintf(w, "  todo   %s\n", issue)
		}
		if s.CanRepair() {
			fmt.Fprintln(w, "         Run picode-desktop startup-repair.")
		} else {
			fmt.Fprintln(w, "         Inspect the task registration before running picode-desktop install.")
		}
	}
	fmt.Fprintf(w, "  info   Executable: %s %s\n", s.Executable, s.Arguments)
	fmt.Fprintf(w, "  info   Last task result: 0x%08X (last start: %s)\n", uint32(s.LastResult), s.LastRun)
	return s.Enabled && len(issues) == 0
}
