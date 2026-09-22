package store

import (
	"errors"
	"testing"
)

func TestRenameWorkspace(t *testing.T) {
	s := openTest(t)
	w, err := s.AddWorkspace("old", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.RenameWorkspace(w.ID, "  New name  ")
	if err != nil || got.Name != "New name" || got.Path != w.Path {
		t.Fatalf("rename = %+v (%v)", got, err)
	}
	if again, _ := s.GetWorkspace(w.ID); again.Name != "New name" {
		t.Fatalf("stored name = %q", again.Name)
	}
	for _, c := range []struct {
		id, name string
		notFound bool
	}{
		{w.ID, "   ", false},
		{"ws_missing", "x", true},
		{FreeWorkspaceID, "x", true},
	} {
		_, err := s.RenameWorkspace(c.id, c.name)
		if err == nil || errors.Is(err, ErrNotFound) != c.notFound {
			t.Errorf("RenameWorkspace(%q, %q) = %v, notFound=%v", c.id, c.name, err, c.notFound)
		}
	}
}

func TestRemoveWorkspaceDropsItsIntegrationDeclaration(t *testing.T) {
	s := openTest(t)
	w, err := s.AddWorkspace("w", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PutIntegrationSettings(w.ID, IntegrationSettingsMutation{FFOnly: true, Checks: []string{"make ci"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RemoveWorkspace(w.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.IntegrationSettingsFor(w.ID); ok {
		t.Fatal("the declaration outlived its workspace")
	}
}
