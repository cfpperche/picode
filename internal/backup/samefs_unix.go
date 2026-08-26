//go:build !windows

package backup

import (
	"path/filepath"
	"syscall"
)

// SameFS reports whether two paths live on the same device.
func SameFS(a, b string) bool {
	da, oka := fsDev(a)
	db, okb := fsDev(b)
	return oka && okb && da == db
}

func fsDev(p string) (uint64, bool) {
	p = canon(p)
	for p != "" {
		var st syscall.Stat_t
		if err := syscall.Stat(p, &st); err == nil {
			return uint64(st.Dev), true
		}
		next := filepath.Dir(p)
		if next == p {
			break
		}
		p = next
	}
	return 0, false
}
