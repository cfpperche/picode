package main

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/cfpperche/picode/internal/desktop"
)

// maxStages bounds the loop. Every stage is meant to change what Detect sees;
// one that does not would otherwise spin forever, so it is reported instead.
const maxStages = 8

// bootstrap walks a machine up to the point where `picode provision` can run:
// WSL installed, a WSL 2 distro registered, and a real account to own it. It
// re-reads the machine after each stage rather than trusting a script to have
// worked, so an interrupted install resumes and a finished one does nothing.
//
// It returns false when the machine needs a Windows restart; setup continues
// on its own at the next logon.
func bootstrap(a *app, distroFlag string) (ready bool, err error) {
	var last desktop.Stage

	for i := 0; i < maxStages; i++ {
		state := desktop.Detect(a.runner, distroFlag)
		stage := desktop.NextStage(state)

		if stage == desktop.StageProvision {
			return true, nil
		}
		if stage == last {
			return false, fmt.Errorf("stuck trying to %s", desktop.Describe(stage, distroFlag))
		}
		last = stage

		fmt.Printf("  ...    %s\n", desktop.Describe(stage, distroFlag))

		switch stage {
		case desktop.StageInstallWSL:
			if reboot, err := runMaybeReboot(a, desktop.InstallWSLArgs()); err != nil {
				return false, fmt.Errorf("install WSL: %w", err)
			} else if reboot {
				return false, scheduleResume(a)
			}

		case desktop.StageReboot:
			return false, scheduleResume(a)

		case desktop.StageInstallDistro:
			if reboot, err := runMaybeReboot(a, desktop.InstallDistroArgs(distroFlag)); err != nil {
				return false, fmt.Errorf("install the distribution: %w", err)
			} else if reboot {
				return false, scheduleResume(a)
			}

		case desktop.StageCreateUser:
			if err := createAccount(a, state, distroFlag); err != nil {
				return false, err
			}
		}
	}
	return false, fmt.Errorf("setup did not settle after %d stages", maxStages)
}

// runMaybeReboot separates "this failed" from "this worked, Windows wants a
// restart" — exit code 3010.
func runMaybeReboot(a *app, args []string) (reboot bool, err error) {
	err = a.runner.Run(desktop.WSLExe, args...)
	if desktop.RebootRequired(err) {
		return true, nil
	}
	return false, err
}

// createAccount gives a `--no-launch` distro the user it does not have. The
// name follows the Windows account so the owner keeps their own, and the
// password is left locked: PiCode reaches root through `wsl -u root` and never
// needs sudo, so setting a password here would be an unasked-for decision.
func createAccount(a *app, state desktop.MachineState, distroFlag string) error {
	picked, err := desktop.Pick(state.Distros, distroFlag)
	if err != nil {
		return err
	}
	name := desktop.LinuxUserName(windowsUser())

	if err := a.runner.Run(desktop.WSLExe,
		desktop.WSLArgs(picked.Name, "root", desktop.CreateUserCommand(name)...)...); err != nil {
		return fmt.Errorf("create the account %q: %w", name, err)
	}
	if err := a.runner.Run(desktop.WSLExe,
		desktop.WSLArgs(picked.Name, "root", desktop.SetDefaultUserCommand(name)...)...); err != nil {
		return fmt.Errorf("set %q as the default account: %w", name, err)
	}

	fmt.Printf("  ok     created the Linux account %q\n", name)
	fmt.Printf("         it has no password yet — set one with:\n")
	fmt.Printf("           wsl -d %s -u root passwd %s\n", picked.Name, name)
	return nil
}

// scheduleResume arranges for setup to pick up where it left off. RunOnce
// fires at the next logon and removes itself, so a completed install leaves
// nothing behind.
func scheduleResume(a *app) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := a.runner.Run("reg", desktop.RunOnceArgs(exe)...); err != nil {
		return fmt.Errorf("schedule the resume after restart: %w", err)
	}
	fmt.Println("\nWindows needs to restart to finish installing WSL.")
	fmt.Println("Setup continues on its own the next time you sign in.")
	return nil
}

// windowsUser is the account name to model the Linux one on.
func windowsUser() string {
	if n := strings.TrimSpace(os.Getenv("USERNAME")); n != "" {
		return n
	}
	if u, err := user.Current(); err == nil {
		// A domain account arrives as DOMAIN\name.
		if i := strings.LastIndexAny(u.Username, `\/`); i >= 0 {
			return u.Username[i+1:]
		}
		return u.Username
	}
	return ""
}
