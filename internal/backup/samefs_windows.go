//go:build windows

package backup

import (
	"path/filepath"
	"strings"
)

// SameFS reports whether two paths share a drive letter.
func SameFS(a, b string) bool {
	va := strings.ToUpper(filepath.VolumeName(canon(a)))
	vb := strings.ToUpper(filepath.VolumeName(canon(b)))
	return va != "" && va == vb
}
