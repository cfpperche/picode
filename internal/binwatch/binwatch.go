// Package binwatch reloads picode when the executable on disk is newer
// than this process. Stops serving a yesterday-embed after `go build`.
package binwatch

import (
	"log"
	"os"
	"time"
)

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

// Watch calls reload when the binary on disk changes. reload must
// release the process lock before replacing this process.
func Watch(start Stamp, reload func()) {
	if start.Path == "" || reload == nil {
		return
	}
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for range t.C {
			if Changed(start) {
				log.Printf("picode: %s is newer on disk — reloading", start.Path)
				reload()
				return
			}
		}
	}()
}
