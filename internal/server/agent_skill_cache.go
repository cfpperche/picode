package server

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// agentSkillCacheMinAge spares a copy written moments ago whose agent list
// is still being saved (the cache is written before the list names it).
const agentSkillCacheMinAge = time.Hour

// sweepAgentSkillCache removes the cached agent skills nobody can load
// again (ADR-0196 slice 4, <data>/skills/cache/<digest>/<name>). A digest
// stays while a live agent names it or an exit the person has not
// forgotten does — Bring back (ADR-0205) puts that list back, and its
// folders must still be there.
func sweepAgentSkillCache(deps Deps, minAge time.Duration) (int, error) {
	if deps.Store == nil || deps.DataDir == "" {
		return 0, nil
	}
	root := filepath.Join(deps.DataDir, "skills", "cache")
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	keep := map[string]bool{}
	agents, err := deps.Store.ListAllAgents()
	if err != nil {
		return 0, err
	}
	for _, a := range agents {
		for _, sk := range a.Skills {
			keep[sk.Digest] = true
		}
	}
	if err := deps.Store.EachAgentExit(func(ex store.AgentExit) error {
		if ex.ForgottenAt == nil {
			for _, sk := range ex.Config.Skills {
				keep[sk.Digest] = true
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range ents {
		if !e.IsDir() || keep[e.Name()] {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < minAge {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, e.Name())); err == nil {
			removed++
		}
	}
	return removed, nil
}

// sweepAgentSkillCacheLater runs the sweep off the request path.
func sweepAgentSkillCacheLater(deps Deps) {
	go func() {
		if n, err := sweepAgentSkillCache(deps, agentSkillCacheMinAge); err != nil {
			log.Printf("skills: agent cache sweep: %v", err)
		} else if n > 0 {
			log.Printf("skills: removed %d unused agent skill cop(ies)", n)
		}
	}()
}
