package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// deviceLogin runs one CLI's own sign-in command that prints a page and a
// one-time code and then waits for the person to approve it (Muse's
// `muse login`, ADR-0195). It answers with the page and the code as soon as
// the command has printed both, and reports the command's exit afterwards —
// the CLI writes its own credential file; onSuccess files it in the vault.
type deviceLogin struct {
	mu      sync.Mutex
	name    string // the CLI's name, for messages
	running bool
	done    bool
	err     string
	cancel  context.CancelFunc
}

func (d *deviceLogin) start(w http.ResponseWriter, bin string, args []string, env []string, onSuccess func()) {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		writeErr(w, http.StatusConflict, "A "+d.name+" sign-in is already in progress. Finish or cancel it first.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	d.running, d.done, d.err, d.cancel = true, false, "", cancel
	d.mu.Unlock()
	finish := func(errText string) {
		d.mu.Lock()
		d.running, d.done, d.err = false, true, errText
		d.mu.Unlock()
		cancel()
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(), env...)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		finish(err.Error())
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	type found struct{ url, code, last string }
	first := make(chan found, 1)
	go func() {
		sc := bufio.NewScanner(pr)
		var f found
		sent := false
		for sc.Scan() {
			line := strings.TrimSpace(grokANSI.ReplaceAllString(sc.Text(), ""))
			if line != "" {
				f.last = line
			}
			if f.url == "" {
				f.url = grokURL.FindString(line)
			}
			if f.code == "" {
				f.code = grokCode.FindString(line)
			}
			if f.url != "" && f.code != "" && !sent {
				first <- f
				sent = true
			}
		}
		if !sent {
			first <- f
		}
	}()
	exited := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		_ = pw.Close()
		exited <- err
	}()
	var got found
	select {
	case got = <-first:
	case <-time.After(45 * time.Second):
		cancel()
		finish(d.name + " did not show a sign-in page in time.")
		writeErr(w, http.StatusBadGateway, d.name+" did not show a sign-in page in time.")
		return
	}
	if got.url == "" {
		<-exited
		msg := got.last
		if msg == "" {
			msg = d.name + " ended without a sign-in page."
		}
		finish(msg)
		writeErr(w, http.StatusBadGateway, msg)
		return
	}
	go func() {
		err := <-exited
		switch {
		case err == nil:
			if onSuccess != nil {
				onSuccess()
			}
			finish("")
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			finish("The sign-in expired. Start it again.")
		case errors.Is(ctx.Err(), context.Canceled):
			finish("The sign-in was cancelled.")
		default:
			finish(d.name + " could not finish the sign-in.")
		}
	}()
	writeJSON(w, http.StatusOK, map[string]any{"url": got.url, "userCode": got.code})
}

func (d *deviceLogin) status(w http.ResponseWriter, _ *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"pending": d.running, "done": d.done, "error": d.err})
}

func (d *deviceLogin) stop(w http.ResponseWriter, _ *http.Request) {
	d.mu.Lock()
	cancel, running := d.cancel, d.running
	d.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
