//go:build windows

package main

import (
	"os/exec"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/hostfs"
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
	compacting    bool
	quitRequested bool

	status  *systray.MenuItem
	disk    *systray.MenuItem
	compact *systray.MenuItem
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
		// The action the number exists for. It stops the distro, so it asks in a
		// dialog, and it stays disabled until there is something to give back.
		t.compact = systray.AddMenuItem("No held space to give back", "Return the space the distro has freed to C:")
		t.compact.Disable()
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

	t.mu.Lock()
	t.diskTitle = title
	t.diskWarn = warn
	t.mu.Unlock()
	t.disk.SetTitle(title)
	t.refreshCompactItem(ptr, used)
	t.refreshTooltip()
}

// refreshCompactItem turns the held number into the one action this tray
// offers. It is disabled on purpose when there is nothing to give back, when
// someone is mid-compact, or when this WSL build cannot convert the file — a
// grey line that says why beats an error after the click.
func (t *tray) refreshCompactItem(ptr *desktop.DiskFacts, used int64) {
	t.mu.Lock()
	compacting := t.compacting
	t.mu.Unlock()

	switch {
	case compacting:
		t.compact.SetTitle("Compacting…")
		t.compact.Disable()
	case ptr == nil:
		t.compact.SetTitle("Disk not read — nothing to offer")
		t.compact.Disable()
	case !ptr.CanSparse:
		t.compact.SetTitle("Compact from an admin terminal — WSL " + ptr.WSL + " cannot convert")
		t.compact.Disable()
	case desktop.Held(*ptr, used) < heldWorthTelling:
		t.compact.SetTitle("No held space to give back")
		t.compact.Disable()
	default:
		t.compact.SetTitle("Give back ≈" + hostfs.Bytes(desktop.Held(*ptr, used)) + "…")
		t.compact.Enable()
	}
}

// beginCompact is the re-entry guard: one compact at a time, however many
// times the menu item is clicked while one runs.
func (t *tray) beginCompact() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.compacting {
		return false
	}
	t.compacting = true
	return true
}

func (t *tray) endCompact() {
	t.mu.Lock()
	t.compacting = false
	t.mu.Unlock()
}

// serverURL is the address the readiness question goes to, from the cache the
// health probe filled or straight from the distro.
func (t *tray) serverURL() string {
	t.mu.Lock()
	url := t.url
	t.mu.Unlock()
	if url != "" {
		return url
	}
	got, err := desktop.ServerURL(t.app.runner, t.app.distro, t.app.user)
	if err != nil {
		return ""
	}
	return got
}

// compactFlow is what the menu item runs. Every refusal before the compact is
// a dialog that says why; the compact itself is the one thing this tray does
// that ends sessions, so the dialog asking about it names that in plain words.
func (t *tray) compactFlow() {
	if !t.beginCompact() {
		return
	}
	defer t.endCompact()

	url := t.serverURL()
	if url == "" {
		alert("PiCode is not answering, so the tray cannot check whether agents are working.\n\nOpen PiCode, wait for it to come up, and try again.", "PiCode")
		return
	}
	ready, busy, err := desktop.DeployReady(url)
	if err != nil {
		alert("Could not ask PiCode whether agents are working:\n"+err.Error(), "PiCode")
		return
	}
	if !ready {
		alert("Someone is still working:\n\n  "+strings.Join(busy, "\n  ")+"\n\nAsk them to finish, then give the space back.", "PiCode")
		return
	}

	facts, err := desktop.DistroDisk(t.app.runner, t.app.distro)
	if err != nil {
		alert("Could not read the disk:\n"+err.Error(), "PiCode")
		return
	}
	used, _ := distroUsed(t.app.runner, t.app.distro, t.app.user)
	held := desktop.Held(facts, used)
	if held < heldWorthTelling {
		alert("Nothing is held: "+t.app.distro+" has given back what it freed.", "PiCode")
		return
	}
	if !facts.CanSparse {
		alert("This WSL ("+facts.WSL+") cannot convert the file to sparse.\n\nRun picode-desktop disk-compact --yes from an administrator terminal instead.", "PiCode")
		return
	}

	if !confirm("Stopping "+t.app.distro+" returns ≈"+hostfs.Bytes(held)+" to C:.\n\nEverything inside it ends now — agents, terminals and tmux sessions do not come back.\n\nContinue?", "Give back disk space") {
		return
	}

	method, _ := desktop.PlanCompact(facts)
	res, err := desktop.Compact(t.app.runner, t.app.distro, t.app.user, method, facts.VHDXPath, facts.AllocatedBytes, func(s string) {
		t.mu.Lock()
		t.diskTitle = "Disk: " + s
		t.mu.Unlock()
		t.disk.SetTitle("Disk: " + s)
	})

	// wsl --terminate took the keepalive's child down with the distro. The
	// distro is starting again; without this it would idle out from under it
	// sixty seconds later.
	t.restartKeepalive()
	t.diskTick()

	if err != nil {
		alert("The compact failed and "+t.app.distro+" was started again:\n"+err.Error(), "PiCode")
		return
	}
	alert("Before: "+hostfs.Bytes(res.BeforeBytes)+" on disk\nAfter: "+hostfs.Bytes(res.AfterBytes)+"\nBack on C: ≈"+hostfs.Bytes(res.Returned())+"\n\n"+t.app.distro+" is starting again; its sessions do not come back.", "PiCode")
}

// restartKeepalive re-arms the child that holds the distro open. The old one
// is already dead by then — wsl --terminate ends it — and nothing else
// restarts it.
func (t *tray) restartKeepalive() {
	if t.keepalive != nil && t.keepalive.Process != nil {
		_ = t.keepalive.Process.Kill()
	}
	if cmd, err := startSupervised(desktop.WSLExe, desktop.KeepaliveArgs(t.app.distro)...); cmd != nil {
		t.mu.Lock()
		t.keepalive = cmd
		t.mu.Unlock()
		_ = err // the keepalive runs either way; only supervision may have failed
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

		case <-t.compact.ClickedCh:
			go t.compactFlow()

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
