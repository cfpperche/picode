package pipkg

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

// PiPackage is the npm package behind the `pi` command.
const PiPackage = "@earendil-works/pi-coding-agent"

// piRegistryBase is the registry seam for tests.
var piRegistryBase = npmRegistry

var (
	piCheckMu    sync.Mutex
	piCheckCache PiUpdateInfo
	piCheckAt    time.Time
)

// PiUpdateInfo is the answer to "is pi behind the registry?".
type PiUpdateInfo struct {
	Current  string `json:"current"`
	Latest   string `json:"latest,omitempty"`
	Outdated bool   `json:"outdated"`
}

// PiUpdateCheck compares the installed pi version with the npm registry.
// The check rides on GET /api/system, so it is cached for six hours.
// A registry hiccup falls back to the last known answer rather than
// flashing a stale-free dot.
func PiUpdateCheck(ctx context.Context, current string) PiUpdateInfo {
	if current == "" {
		return PiUpdateInfo{}
	}
	piCheckMu.Lock()
	fresh := time.Since(piCheckAt) < 6*time.Hour && piCheckCache.Current == current
	cached := piCheckCache
	piCheckMu.Unlock()
	if fresh {
		return cached
	}
	latest, err := npmLatest(ctx, nil, piRegistryBase, PiPackage)
	if err != nil || latest == "" {
		if !piCheckAt.IsZero() && piCheckCache.Current == current {
			return cached // registry hiccup: stale answer for THIS version beats none
		}
		return PiUpdateInfo{Current: current}
	}
	info := PiUpdateInfo{Current: current, Latest: latest, Outdated: Newer(latest, current)}
	piCheckMu.Lock()
	piCheckCache = info
	piCheckAt = time.Now()
	piCheckMu.Unlock()
	return info
}

// ResetPiUpdateCache drops the cached check (after a self-update).
func ResetPiUpdateCache() {
	piCheckMu.Lock()
	piCheckAt = time.Time{}
	piCheckCache = PiUpdateInfo{}
	piCheckMu.Unlock()
}

// UpdatePiSelf runs pi's own self-update and returns its combined output.
func UpdatePiSelf(ctx context.Context, piCmd string) (string, error) {
	if piCmd == "" {
		piCmd = "pi"
	}
	cmd := exec.CommandContext(ctx, piCmd, "update", "--self")
	out, err := cmd.CombinedOutput()
	return string(out), err
}
