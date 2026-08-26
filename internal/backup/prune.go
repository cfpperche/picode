package backup

import (
	"fmt"
	"os"
	"time"
)

// Prune deletes snapshots older than keepDays, always leaving the newest complete one.
func Prune(dest string, keepDays int, now time.Time) error {
	if keepDays < 1 {
		keepDays = 1
	}
	list, err := List(dest)
	if err != nil {
		return err
	}
	if len(list) <= 1 {
		return nil
	}
	cut := now.Add(-time.Duration(keepDays) * 24 * time.Hour)
	for i, s := range list {
		if i == 0 {
			continue // newest stays
		}
		t, err := time.Parse(time.RFC3339, s.Created)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z", s.Created)
		}
		if err != nil {
			continue
		}
		if t.Before(cut) {
			_ = os.RemoveAll(s.Path)
		}
	}
	return nil
}

// Remove deletes one complete snapshot by id.
func Remove(dest, id string) error {
	if id == "" {
		return fmt.Errorf("backup: snapshot id required")
	}
	list, err := List(dest)
	if err != nil {
		return err
	}
	for _, s := range list {
		if s.ID == id {
			return os.RemoveAll(s.Path)
		}
	}
	return fmt.Errorf("backup: snapshot not found")
}
