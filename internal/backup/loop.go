package backup

import (
	"context"
	"log"
	"time"
)

// Loop ticks every minute and takes a snapshot when due.
func (e *Engine) Loop(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	e.tick()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.tick()
		}
	}
}

func (e *Engine) tick() {
	if e.Store == nil {
		return
	}
	s, err := LoadSettings(e.Store, e.DataDir)
	if err != nil || !s.Enabled {
		return
	}
	if !Due(s, e.now()) {
		return
	}
	if _, err := e.Snapshot(s.Sessions, s.Secrets, s.Dir); err != nil {
		log.Printf("backup: %v", err)
		return
	}
	if err := Prune(s.Dir, s.KeepDays, e.now()); err != nil {
		log.Printf("backup: prune: %v", err)
	}
}
