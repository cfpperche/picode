package tmux

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// interceptMarker opens every PiCode intercept wrapper on a terminal's PATH
// (ADR-0056), the tmux guard among them (ADR-0138).
var interceptMarker = []byte("# PiCode intercept")

var binaryCache sync.Map // PATH → resolved binary

// Binary is the tmux executable PiCode's own code runs: the first `tmux` on
// PATH that is not a PiCode intercept wrapper. The guard gates what people
// and agents type inside PiCode terminals; PiCode's own calls — a harness or
// the docs fixture ending the private server it started, run from such a
// terminal — are gated by refuseUserServer instead. Without this a fixture
// launched from a PiCode terminal could not kill its own server, deleted its
// directory anyway and left an unreachable tmux server behind (2026-09-22:
// one held the deploy lock for hours). Falls back to "tmux" when nothing
// better is found, so errors read as they always did.
func Binary() string {
	path := os.Getenv("PATH")
	if v, ok := binaryCache.Load(path); ok {
		return v.(string)
	}
	bin := "tmux"
	for _, dir := range filepath.SplitList(path) {
		if dir == "" || !filepath.IsAbs(dir) {
			continue
		}
		p := filepath.Join(dir, "tmux")
		st, err := os.Stat(p)
		if err != nil || st.IsDir() || st.Mode()&0o111 == 0 || isIntercept(p) {
			continue
		}
		bin = p
		break
	}
	binaryCache.Store(path, bin)
	return bin
}

func isIntercept(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, 256))
	return bytes.Contains(b, interceptMarker)
}
