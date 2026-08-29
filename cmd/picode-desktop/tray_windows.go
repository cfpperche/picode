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

type tray struct {
	app app

	mu     sync.Mutex
	url    string
	bootID string
	up     bool

	status  *systray.MenuItem
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
		systray.SetTitle("PiCode")
		systray.SetTooltip("PiCode — starting…")

		t.status = systray.AddMenuItem("Starting…", "")
		t.status.Disable()
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
		// PiCode goes down while the tray still says it is up.
		if cmd, err := startDetached(desktop.WSLExe, desktop.KeepaliveArgs(t.app.distro)...); err == nil {
			t.keepalive = cmd
		}

		go t.poll()
		go t.handleClicks()
	}, func() {
		if t.keepalive != nil && t.keepalive.Process != nil {
			_ = t.keepalive.Process.Kill()
		}
	})
	return resolveErr
}

func (t *tray) poll() {
	for {
		t.tick()
		time.Sleep(pollEvery)
	}
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
	t.mu.Unlock()

	if up {
		systray.SetTooltip("PiCode — " + detail)
		t.status.SetTitle("Running · " + detail)
		t.open.Enable()
		t.restart.Enable()
	} else {
		systray.SetTooltip("PiCode — " + detail)
		t.status.SetTitle("Stopped · " + detail)
		t.open.Disable()
		t.restart.Enable()
	}
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
			systray.Quit()
			return
		}
	}
}
