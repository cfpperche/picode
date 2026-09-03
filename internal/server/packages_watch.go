package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/pipkg"
)

// pipkgCheck is the scan seam the package watcher runs on; the tests swap
// it so no test ever touches the npm registry.
var pipkgCheck = pipkg.CheckUpdates

// StartPackageUpdatesWatch re-runs the npm update check on a slow ticker
// for the user dir and every registered workspace's project dir, and
// publishes ephemeral packages.updates events (ADR-0048) only when a
// scope's result changes. Scans are serial and network-bound, so the
// cadence is the 30 minutes the browser used to poll — one scan set for
// the whole fleet instead of one per open browser.
func StartPackageUpdatesWatch(ctx context.Context, deps Deps, every time.Duration) {
	if deps.Feed == nil || deps.Store == nil {
		return
	}
	prev := map[string]string{}
	run := func() {
		scopes := []pkgScope{{scope: "user"}}
		workspaces, err := deps.Store.ListWorkspaces()
		if err == nil {
			for _, w := range workspaces {
				if strings.TrimSpace(w.Path) == "" {
					continue
				}
				scopes = append(scopes, pkgScope{scope: "workspace:" + w.ID, dir: w.Path})
			}
		}
		for _, sc := range scopes {
			select {
			case <-ctx.Done():
				return
			default:
			}
			sctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			rep, err := pipkgCheck(sctx, pipkg.UserDir(), sc.dir)
			cancel()
			if err != nil {
				continue // offline / not a project: keep the last known list
			}
			key := updatesFingerprint(rep)
			if prev[sc.scope] == key {
				continue
			}
			prev[sc.scope] = key
			deps.Feed.Ephemeral("packages.updates", map[string]any{"scope": sc.scope, "updates": rep.Updates})
		}
	}
	run()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

type pkgScope struct {
	scope string // "user" or "workspace:<id>" — the client matches on this
	dir   string // project dir; empty for the user scope
}

// updatesFingerprint hashes name|current|latest per row, sorted, so the
// same report in a different order (List walks the filesystem) does not
// publish, and any real change does.
func updatesFingerprint(rep pipkg.UpdateReport) string {
	lines := make([]string, 0, len(rep.Updates))
	for _, u := range rep.Updates {
		lines = append(lines, u.Source+"|"+u.Scope+"|"+u.Current+"|"+u.Latest)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(fmt.Sprint(len(lines), "\x00", strings.Join(lines, "\x00"))))
	return hex.EncodeToString(sum[:])
}
