package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/desktop"
)

// placesReport is where the distro's disk file lives and where it could go:
// every ready drive with the answer for the default folder, so the window
// shows the reason next to a drive instead of offering one that cannot fit.
type placesReport struct {
	Distro string             `json:"distro"`
	Disk   *desktop.DiskFacts `json:"disk,omitempty"`
	Folder string             `json:"folder"`
	// BackupFolder is the default for Back up here.
	BackupFolder string `json:"backupFolder"`
	// CanMove / CanBackup: this WSL knows the commands (an older build
	// does not, and would fail only after the distro was stopped).
	CanMove   bool         `json:"canMove"`
	CanBackup bool         `json:"canBackup"`
	Drives    []placeDrive `json:"drives"`
	Error     string       `json:"error,omitempty"`
}

type placeDrive struct {
	desktop.WinVolume
	Check desktop.PlaceCheck `json:"check"`
}

// defaultFolder is where a moved distro goes unless the person names
// another folder; backups get their own, so a backup never lands in the
// folder a later move would need empty.
func defaultFolder(distro string) string { return `WSL\` + distro }

func defaultBackupFolder() string { return `WSL\Backups` }

func runPlaces(distroFlag, userFlag string) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	a.runner = timedRunner{d: 30 * time.Second}
	rep := placesReport{Distro: a.distro, Folder: defaultFolder(a.distro), BackupFolder: defaultBackupFolder(), Drives: []placeDrive{}}
	facts, err := desktop.DistroDisk(a.runner, a.distro)
	if err != nil {
		rep.Error = err.Error()
		return json.NewEncoder(os.Stdout).Encode(rep)
	}
	rep.Disk = &facts
	rep.CanMove, rep.CanBackup = facts.CanMove, facts.CanExportVHD
	vols, err := desktop.ListVolumes(a.runner)
	if err != nil {
		rep.Error = err.Error()
		return json.NewEncoder(os.Stdout).Encode(rep)
	}
	// The per-drive answer is about the drive (fixed, NTFS, room); the
	// folder is the person's to type and is checked again when they act.
	for _, v := range vols {
		c := desktop.CheckPlace(vols, facts, v.Letter, rep.Folder)
		if c.SameDisk {
			c.OK, c.Reason = true, ""
		}
		rep.Drives = append(rep.Drives, placeDrive{WinVolume: v, Check: c})
	}
	return json.NewEncoder(os.Stdout).Encode(rep)
}

// runRelocate moves the disk file (backup=false) or writes a copy of it as
// one .vhdx (backup=true). Order: check the place (nothing stops if it
// cannot fit), the interlock, --yes, stop the distro, create the folder,
// the copy (bounded at 4 h), start the distro again whatever happened.
func runRelocate(distroFlag, userFlag, drive, folder string, backup, yes, unreachable, asJSON bool) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	a.runner = timedRunner{d: 30 * time.Second}
	out := wslOutcome{Distro: a.distro}
	emit := func() error {
		if asJSON {
			return json.NewEncoder(os.Stdout).Encode(out)
		}
		switch {
		case out.Refused != "":
			return fmt.Errorf("not run — %s", out.Refused)
		case out.Error != "":
			return fmt.Errorf("%s", out.Error)
		default:
			fmt.Println(out.Note)
		}
		return nil
	}
	say := func(s string) {
		if asJSON {
			fmt.Println(progressLine(s))
			return
		}
		fmt.Println("  " + s)
	}

	say("checking the destination")
	if folder == "" {
		folder = defaultFolder(a.distro)
		if backup {
			folder = defaultBackupFolder()
		}
	}
	facts, err := desktop.DistroDisk(a.runner, a.distro)
	if err != nil {
		out.Error = err.Error()
		return emit()
	}
	if backup && !facts.CanExportVHD {
		out.Refused = "this WSL cannot write a .vhdx backup — update WSL on the System tab first"
		return emit()
	}
	if !backup && !facts.CanMove {
		out.Refused = "this WSL cannot move a distro — update WSL on the System tab first"
		return emit()
	}
	vols, err := desktop.ListVolumes(a.runner)
	if err != nil {
		out.Error = err.Error()
		return emit()
	}
	place := desktop.CheckPlace(vols, facts, drive, folder)
	if !place.OK {
		out.Refused = place.Reason
		return emit()
	}
	// A move needs its folder absent or empty; checked now, not after the
	// sessions ended. The path passed CheckPlace, so it quotes safely.
	if !backup {
		count, _ := a.runner.Output(desktop.PowerShellExe, "-NoProfile", "-NonInteractive", "-Command",
			"(Get-ChildItem -Force -LiteralPath '"+place.Path+"' -ErrorAction SilentlyContinue | Measure-Object).Count")
		if n := strings.TrimSpace(desktop.DecodeWindows(count)); n != "" && n != "0" {
			out.Refused = place.Path + " already has files in it — pick an empty or new folder"
			return emit()
		}
	}

	say("checking that no agent is working")
	if refusal, failure := idleInterlock(a, false, asJSON); refusal != "" && !(unreachable && refusal == unreachableRefusal) {
		out.Refused = refusal
		return emit()
	} else if failure != "" {
		out.Error = failure
		return emit()
	}
	if !yes {
		out.Note = "Re-run with --yes to stop " + a.distro + " and copy its disk now."
		return emit()
	}

	target := place.Path
	if backup {
		target = place.Path + `\` + a.distro + "-" + time.Now().Format("2006-01-02-150405") + ".vhdx"
	}
	// No deadline on the copy: killing wsl.exe would not stop the copy in
	// the WSL service, and starting the distro on a disk still being moved
	// is the one outcome worse than waiting. The hold keeps the shell's
	// keepalive and discovery from starting it meanwhile.
	release := desktop.HoldDistro(map[bool]string{true: "backup", false: "move"}[backup])
	defer release()
	relocateFlow(a, osRunner{}, backup, place.Path, target, say, &out)
	return emit()
}

// relocateFlow is the part that stops the distro; its order is the promise,
// pinned by a test. The distro starts again whatever failed.
func relocateFlow(a app, long desktop.Runner, backup bool, folder, target string, say func(string), out *wslOutcome) {
	defer keepaliveTaskOff(a.runner)()

	out.Stopped = true
	say("stopping " + a.distro + " — every session inside ends")
	if err := a.runner.Run(desktop.WSLExe, desktop.TerminateArgs(a.distro)...); err != nil {
		out.Error = "stop " + a.distro + ": " + err.Error()
	}
	// A backup needs its folder; a move creates its own (and is not
	// promised to accept one that already exists). The path passed
	// CheckPlace — drive letter and plain segments, no quote can be in it —
	// so it is safe inside single quotes.
	if out.Error == "" && backup {
		if err := a.runner.Run(desktop.PowerShellExe, "-NoProfile", "-NonInteractive", "-Command",
			"New-Item -ItemType Directory -Force -Path '"+folder+"' | Out-Null"); err != nil {
			out.Error = "create " + folder + ": " + err.Error()
		}
	}
	if out.Error == "" {
		if backup {
			say("copying the disk to " + target + " — this takes a while")
			if err := long.Run(desktop.WSLExe, desktop.ExportArgs(a.distro, target)...); err != nil {
				out.Error = "back up to " + target + ": " + err.Error()
			} else {
				out.Copied = true
				out.Note = "Backed up to " + target + "."
			}
		} else {
			say("moving the disk to " + folder + " — this takes a while")
			if err := long.Run(desktop.WSLExe, desktop.MoveArgs(a.distro, folder)...); err != nil {
				out.Error = "move to " + folder + ": " + err.Error()
			} else {
				out.Copied = true
				out.Note = "Moved to " + folder + "."
			}
		}
	}
	say("starting " + a.distro + " again")
	if err := a.runner.Run(desktop.WSLExe, desktop.StartDistroArgs(a.distro, a.user)...); err != nil && out.Error == "" {
		out.Error = "start " + a.distro + ": " + err.Error()
	}
	out.Done = out.Error == ""
}

// keepaliveTaskOff disables the PiCodeDistro task and returns the function
// that re-enables and runs it. The task restarts on failure within a minute,
// so any flow that stops the distro (move, backup, compact) turns it off for
// the work and back on whatever happened; a machine without the task
// answers an error, which is fine.
func keepaliveTaskOff(r desktop.Runner) (restore func()) {
	_ = r.Run(desktop.SchtasksExe, desktop.TaskArgs("/disable")...)
	return func() {
		_ = r.Run(desktop.SchtasksExe, desktop.TaskArgs("/enable")...)
		_ = r.Run(desktop.SchtasksExe, desktop.TaskArgs("/run")...)
	}
}
