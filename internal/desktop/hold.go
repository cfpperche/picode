package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The distro hold: while a long flow needs the distro stopped (a move, a
// backup, a WSL update), nothing may start it again — not the keepalive, and
// not the shell's own server discovery, which runs `wsl.exe -d <distro>`
// every few seconds when PiCode stops answering. The flow writes this file
// and keeps touching it; the shell (desktop-shell/src/hold.rs) skips both
// while it is fresh. A file, not a flag in the shell, so a flow started from
// a terminal holds the distro exactly like one started from the window, and
// a heartbeat, not a pid, so a killed flow's hold expires by itself.

// HoldFresh is how old the file may be and still hold. The flow touches it
// every HoldBeat; the shell reads the same number.
const (
	HoldBeat  = 30 * time.Second
	HoldFresh = 2 * time.Minute
)

// HoldPath is %LOCALAPPDATA%\PiCode\distro-hold.json — next to the
// installed tools, which the shell already knows.
func HoldPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "PiCode", "distro-hold.json")
}

// HoldDistro writes the hold and keeps it fresh until release is called.
// Holds nest: a second hold keeps the file until both are released.
func HoldDistro(op string) (release func()) {
	holdMu.Lock()
	holdCount++
	first := holdCount == 1
	holdMu.Unlock()

	path := HoldPath()
	write := func() {
		b, _ := json.Marshal(map[string]any{"op": op, "pid": os.Getpid(), "since": time.Now().UTC()})
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, b, 0o644)
	}
	if first {
		write()
	}
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(HoldBeat)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				now := time.Now()
				_ = os.Chtimes(path, now, now)
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			holdMu.Lock()
			holdCount--
			last := holdCount == 0
			holdMu.Unlock()
			if last {
				_ = os.Remove(path)
			}
		})
	}
}

var (
	holdMu    sync.Mutex
	holdCount int
)

// HoldActive reports whether a fresh hold exists (the shell's question, answered
// in Go for the tests and for any Go caller).
func HoldActive() bool {
	st, err := os.Stat(HoldPath())
	return err == nil && time.Since(st.ModTime()) < HoldFresh
}
