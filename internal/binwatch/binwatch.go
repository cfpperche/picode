// Package binwatch reloads picode when the executable on disk is newer
// than this process. Stops serving a yesterday-embed after `go build`
// in a foreground run. A systemd-supervised daemon must not re-exec:
// `picode deploy` copies the binary then restarts the unit, and a
// same-PID Exec swallows the SIGTERM systemd already sent.
package binwatch

import (
	"context"
	"log"
	"os"
	"time"
)

const defaultPoll = 2 * time.Second

// Stamp is the executable identity at process start.
type Stamp struct {
	Path  string
	Mtime time.Time
	Size  int64
}

// Capture records the running binary.
func Capture() (Stamp, error) {
	path, err := os.Executable()
	if err != nil {
		return Stamp{}, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return Stamp{}, err
	}
	return Stamp{Path: path, Mtime: st.ModTime(), Size: st.Size()}, nil
}

// Changed reports whether path now differs from start.
func Changed(start Stamp) bool {
	st, err := os.Stat(start.Path)
	if err != nil {
		return false
	}
	return st.Size() != start.Size || st.ModTime().After(start.Mtime.Add(time.Second))
}

// Supervised reports whether systemd owns this process.
func Supervised() bool {
	return os.Getenv("INVOCATION_ID") != ""
}

// Watch calls reload when the binary on disk changes. reload must
// release the process lock before replacing this process. It is a no-op
// under systemd. Cancel ctx to stop watching (SIGTERM).
func Watch(ctx context.Context, start Stamp, reload func()) {
	if Supervised() {
		if start.Path != "" {
			log.Printf("picode: systemd owns this process — not auto-reloading %s", start.Path)
		}
		return
	}
	watch(ctx, defaultPoll, start, reload)
}

func watch(ctx context.Context, every time.Duration, start Stamp, reload func()) {
	if start.Path == "" || reload == nil || every <= 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if !Changed(start) {
					continue
				}
				if ctx.Err() != nil {
					return
				}
				log.Printf("picode: %s is newer on disk — reloading", start.Path)
				reload()
				return
			}
		}
	}()
}
