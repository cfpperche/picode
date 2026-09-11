//go:build windows

package main

import (
	"os/exec"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/cfpperche/picode/internal/desktop"
)

// pollEvery is how often the tray asks PiCode whether it is up. The probe is a
// plain HTTP call to a port on this machine, not a wsl.exe spawn, so it is
// cheap enough to run on a timer.
const pollEvery = 5 * time.Second

// diskEvery is how often the tray re-reads the disk. Those numbers move over
// hours, and every read spawns wsl.exe, PowerShell and fsutil — five minutes
// keeps the line current without turning the notification area into a poller.
const diskEvery = 5 * time.Minute

type tray struct {
	app app

	mu            sync.Mutex
	url           string
	bootID        string
	up            bool
	detail        string
	diskTitle     string
	diskWarn      bool
	quitRequested bool

	status  *systray.MenuItem
	disk    *systray.MenuItem
	open    *systray.MenuItem
	restart *systray.MenuItem
	logs    *systray.MenuItem
	quit    *systray.MenuItem

	keepalive *exec.Cmd
}

func runTray(distroFlag, userFlag string) error {
	t := &tray{}
	var resolveErr error

	systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTitle("PiCode")
		systray.SetTooltip("PiCode — starting…")

		t.status = systray.AddMenuItem("Starting…", "")
		t.status.Disable()
		// The one fact a person otherwise leaves PiCode to check in Explorer:
		// what the distro's disk costs Windows, and how much room is left.
		t.disk = systray.AddMenuItem("Disk: reading…", "The distro's disk file, and free space on the volume it lives on")
		t.disk.Disable()
		systray.AddSeparator()
		t.open = systray.AddMenuItem("Open PiCode", "Open PiCode in the browser")
		t.restart = systray.AddMenuItem("Restart PiCode", "Restart the service inside WSL")
		t.logs = systray.AddMenuItem("View logs", "Follow the service journal")
		systray.AddSeparator()
		t.quit = systray.AddMenuItem("Quit", "Stop the tray (PiCode keeps running)")

		t.app, resolveErr = resolve(distroFlag, userFlag)
		if resolveErr != nil {
			t.setStatus(false, resolveErr.Error())
			go t.handleClicks()
			return
		}

		// Hold the distro open. Without this the VM is reclaimed when idle and
		// PiCode goes down while the tray still says it is up. The child is
		// supervised, so it cannot outlive this process even if the tray is
		// force-killed and onExit never runs.
		if cmd, err := startSupervised(desktop.WSLExe, desktop.KeepaliveArgs(t.app.distro)...); cmd != nil {
			t.keepalive = cmd
			_ = err // the keepalive runs either way; only supervision may have failed
		}

		go t.poll()
		go t.pollDisk()
		go t.handleClicks()
	}, func() {
		if t.keepalive != nil && t.keepalive.Process != nil {
			_ = t.keepalive.Process.Kill()
		}
	})
	t.mu.Lock()
	quitRequested := t.quitRequested
	t.mu.Unlock()
	return trayResult(resolveErr, quitRequested)
}

func (t *tray) poll() {
	for {
		t.tick()
		time.Sleep(pollEvery)
	}
}

func (t *tray) pollDisk() {
	for {
		t.diskTick()
		time.Sleep(diskEvery)
	}
}

// diskTick reads both halves of the disk and updates one menu line. A failed
// half is reported as unread, never as a zero: the toolbar is the last place
// that should tell someone their disk is empty.
func (t *tray) diskTick() {
	facts, factsErr := desktop.DistroDisk(t.app.runner, t.app.distro)

	var ptr *desktop.DiskFacts
	var used int64
	if factsErr == nil {
		ptr = &facts
		// Usage is best-effort: with the file size alone the line is still worth
		// reading, it just cannot say how much is being held. When the Windows
		// half failed there is nothing to spend a wsl.exe spawn on.
		used, _ = distroUsed(t.app.runner, t.app.distro, t.app.user)
	}

	title, warn := diskLine(ptr, used, factsErr)

	// The warning is stored, not painted onto the tooltip here: the health
	// probe rewrites the tooltip every five seconds, and whichever of the two
	// timers writes last has to write the same thing.
	t.mu.Lock()
	t.diskTitle = title
	t.diskWarn = warn
	t.mu.Unlock()
	t.disk.SetTitle(title)
	t.refreshTooltip()
}

func (t *tray) tick() {
	t.mu.Lock()
	url := t.url
	t.mu.Unlock()

	if url == "" {
		got, err := desktop.ServerURL(t.app.runner, t.app.distro, t.app.user)
		if err != nil {
			t.setStatus(false, "PiCode has not started yet")
			return
		}
		t.mu.Lock()
		t.url, url = got, got
		t.mu.Unlock()
	}

	bootID, err := desktop.Health(url)
	if err != nil {
		// The port can move inside its range (8445-8455), so a failed probe
		// invalidates the cached address rather than being reported forever.
		t.mu.Lock()
		t.url = ""
		t.mu.Unlock()
		t.setStatus(false, "not answering")
		return
	}

	t.mu.Lock()
	restarted := t.bootID != "" && t.bootID != bootID
	t.bootID = bootID
	t.mu.Unlock()

	detail := url
	if restarted {
		detail = url + " (restarted)"
	}
	t.setStatus(true, detail)
}

func (t *tray) setStatus(up bool, detail string) {
	t.mu.Lock()
	t.up = up
	t.detail = detail
	t.mu.Unlock()

	if up {
		t.status.SetTitle("Running · " + detail)
		t.open.Enable()
		t.restart.Enable()
	} else {
		t.status.SetTitle("Stopped · " + detail)
		t.open.Disable()
		t.restart.Enable()
	}
	t.refreshTooltip()
}

// refreshTooltip is the one writer of the tooltip. Two timers feed it — the
// health probe every five seconds, the disk every five minutes — so the
// composition lives here rather than at the call sites, or the faster timer
// would keep erasing what the slower one wrote (the tray has no balloon left
// to raise: fyne.io/systray v1.12 removed that API).
func (t *tray) refreshTooltip() {
	systray.SetTooltip(t.tooltip())
}

// tooltip is the status line and the disk line together, because the tooltip
// is read at a glance and a person looking for one of them is looking for
// both.
func (t *tray) tooltip() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := "PiCode — " + t.detail
	if t.diskTitle != "" {
		s += " · " + t.diskTitle
	}
	if t.diskWarn {
		s += " — run picode-desktop disk for the two-sided report"
	}
	return s
}

func (t *tray) handleClicks() {
	for {
		select {
		case <-t.open.ClickedCh:
			t.mu.Lock()
			url := t.url
			t.mu.Unlock()
			if url != "" {
				_ = newCmd("rundll32", "url.dll,FileProtocolHandler", url).Start()
			}

		case <-t.restart.ClickedCh:
			go func() {
				_ = t.app.runner.Run(desktop.WSLExe, desktop.WSLArgs(t.app.distro, t.app.user,
					"systemctl", "--user", "restart", "picode")...)
				t.tick()
			}()

		case <-t.logs.ClickedCh:
			// A log window is the one place a console is wanted, so this one
			// deliberately opens Windows Terminal instead of suppressing it.
			args := append([]string{desktop.WSLExe}, desktop.WSLArgs(t.app.distro, t.app.user,
				"journalctl", "--user", "-u", "picode", "-f")...)
			if err := exec.Command("wt.exe", args...).Start(); err != nil {
				_ = exec.Command("cmd", append([]string{"/c", "start", ""}, args...)...).Start()
			}

		case <-t.quit.ClickedCh:
			t.mu.Lock()
			t.quitRequested = true
			t.mu.Unlock()
			systray.Quit()
			return
		}
	}
}
