package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/desktop"
)

// hostReport is the Management window's System tab: memory from both sides
// and which WSL this is. Like the disk report, each half keeps its own error
// — a missing half is named, never shown as zero.
type hostReport struct {
	Distro       string               `json:"distro"`
	Windows      *desktop.HostMemory  `json:"windows,omitempty"`
	WindowsError string               `json:"windowsError,omitempty"`
	VM           *desktop.Meminfo     `json:"vm,omitempty"`
	VMError      string               `json:"vmError,omitempty"`
	Versions     *desktop.WSLVersions `json:"versions,omitempty"`
	VersionError string               `json:"versionError,omitempty"`
	// Latest is WSL's newest release on GitHub, when that answered; the
	// window offers Update only when it is newer than Versions.WSL.
	Latest      string    `json:"latest,omitempty"`
	LatestError string    `json:"latestError,omitempty"`
	Newer       bool      `json:"newer,omitempty"`
	At          time.Time `json:"at"`
}

// latestWSLURL is Microsoft's own release feed for WSL. Unauthenticated,
// one call per window open — far under the API's hourly allowance.
var latestWSLURL = "https://api.github.com/repos/microsoft/WSL/releases/latest"

func runHost(distroFlag, userFlag string) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	// Every read is bounded: a hung WSL leaves the tab with an error, not
	// with skeletons forever.
	a.runner = timedRunner{d: 30 * time.Second}
	return json.NewEncoder(os.Stdout).Encode(collectHost(a, latestWSL))
}

func collectHost(a app, latest func() (string, error)) hostReport {
	rep := hostReport{Distro: a.distro, At: time.Now().UTC()}
	// The distro first: reading it is what wakes an idle VM, so the Windows
	// half, read second, sees the VM the distro half described.
	if m, err := desktop.ReadDistroMemory(a.runner, a.distro, a.user); err != nil {
		rep.VMError = err.Error()
	} else {
		rep.VM = &m
	}
	if w, err := desktop.ReadHostMemory(a.runner); err != nil {
		rep.WindowsError = err.Error()
	} else {
		rep.Windows = &w
	}
	if v, err := desktop.ReadWSLVersion(a.runner); err != nil {
		rep.VersionError = err.Error()
	} else {
		rep.Versions = &v
	}
	if tag, err := latest(); err != nil {
		rep.LatestError = err.Error()
	} else {
		rep.Latest = tag
		rep.Newer = rep.Versions != nil && desktop.NewerVersion(tag, rep.Versions.WSL)
	}
	return rep
}

// latestWSL reads the newest release tag. Bounded: a slow network must not
// hold the System tab.
func latestWSL() (string, error) {
	client := &http.Client{Timeout: 6 * time.Second}
	req, err := http.NewRequest(http.MethodGet, latestWSLURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "picode-desktop")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var rel struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return "", fmt.Errorf("release JSON: %w", err)
	}
	if strings.TrimSpace(rel.Tag) == "" {
		return "", fmt.Errorf("release has no tag")
	}
	return strings.TrimSpace(rel.Tag), nil
}

// wslOutcome is the last line of wsl-restart and wsl-update, the same shape
// the compact uses so the window reads all three the same way.
type wslOutcome struct {
	Distro  string `json:"distro"`
	Refused string `json:"refused,omitempty"`
	Note    string `json:"note,omitempty"`
	Error   string `json:"error,omitempty"`
	// Stopped: WSL was shut down and every session in it ended.
	Stopped bool `json:"stopped,omitempty"`
	Done    bool `json:"done,omitempty"`
}

// runWSLRestart stops the whole WSL VM and starts the distro again — the
// only way a .wslconfig change takes effect. update runs `wsl --update`
// first, which restarts the WSL service by itself. Both end every session,
// so both ask the interlock first and want --yes.
//
// unreachable is the Management window's recovery door: when PiCode itself
// does not answer (a wedged VM, a dead daemon), nobody can say whether
// anyone is working, and the person may still choose to restart. It never
// overrides an answer — a daemon that says "busy" still refuses.
func runWSLRestart(distroFlag, userFlag string, update, yes, force, unreachable, asJSON bool) error {
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
		case out.Note != "":
			fmt.Println(out.Note)
		default:
			fmt.Println(a.distro + " is running again; its sessions do not come back.")
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

	say("checking that no agent is working")
	if refusal, failure := idleInterlock(a, force, asJSON); refusal != "" && !(unreachable && refusal == unreachableRefusal) {
		out.Refused = refusal
		return emit()
	} else if failure != "" {
		out.Error = failure
		return emit()
	}
	if !yes {
		out.Note = "Re-run with --yes to stop WSL now."
		return emit()
	}

	var upd desktop.Runner
	if update {
		// The installer may wait on a UAC prompt; 15 minutes, then it is
		// an error and nothing is shut down.
		upd = timedRunner{d: 15 * time.Minute}
	}
	a.runner = timedRunner{d: 2 * time.Minute}
	restartFlow(a, upd, say, &out)
	return emit()
}

// restartFlow is the part after the interlock and the confirmation: the
// order is the promise. upd, when set, runs `wsl --update` first; an update
// that fails — a declined UAC prompt is the common case — shuts nothing
// down, since a restart would end every session for nothing. The distro is
// started again whatever failed, the way the compact restarts it after a
// failed conversion: starting a running distro is a no-op, and a stopped
// WSL is the one outcome worse than a failed update.
func restartFlow(a app, upd desktop.Runner, say func(string), out *wslOutcome) {
	if upd != nil {
		say("updating WSL — Windows may ask for permission")
		if err := upd.Run(desktop.WSLExe, desktop.UpdateArgs()...); err != nil {
			out.Error = "wsl --update: " + err.Error()
			_ = a.runner.Run(desktop.WSLExe, desktop.StartDistroArgs(a.distro, a.user)...)
			return
		}
	}
	out.Stopped = true
	say("stopping WSL — every session inside ends")
	if err := a.runner.Run(desktop.WSLExe, desktop.ShutdownArgs()...); err != nil && out.Error == "" {
		out.Error = "wsl --shutdown: " + err.Error()
	}
	say("starting " + a.distro + " again")
	if err := a.runner.Run(desktop.WSLExe, desktop.StartDistroArgs(a.distro, a.user)...); err != nil && out.Error == "" {
		out.Error = "start " + a.distro + ": " + err.Error()
	}
	out.Done = out.Error == ""
}
