package workspace

import (
	"path/filepath"
	"testing"
)

func TestRegistryCRUD(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "workspaces.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	proj := t.TempDir()
	w, err := r.Add("My Project", proj)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if w.ID == "" || w.Path != proj {
		t.Errorf("Add returned %+v", w)
	}

	// Idempotent on same path.
	w2, err := r.Add("My Project", proj)
	if err != nil {
		t.Fatalf("Add duplicate: %v", err)
	}
	if w2.ID != w.ID {
		t.Errorf("duplicate add = new id %q, want %q", w2.ID, w.ID)
	}

	// Registry reloads from disk (persistence).
	r2, err := Open(filepath.Join(dir, "workspaces.json"))
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	list, err := r2.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("List after reload = %d, %v; want 1, nil", len(list), err)
	}
	if list[0].Name != "My Project" {
		t.Errorf("name = %q", list[0].Name)
	}

	got, ok, err := r2.Get(w.ID)
	if err != nil || !ok || got.Path != proj {
		t.Fatalf("Get = %+v, %v, %v", got, ok, err)
	}
	if _, ok, _ := r2.Get("missing"); ok {
		t.Error("Get(missing) found")
	}

	removed, err := r.Remove(w.ID)
	if err != nil || !removed {
		t.Fatalf("Remove = %v, %v", removed, err)
	}
	if removed, err := r.Remove(w.ID); err != nil || removed {
		t.Fatalf("Remove twice = %v, %v", removed, err)
	}
	if list, _ := r2.List(); len(list) != 0 {
		t.Errorf("List after remove = %d, want 0", len(list))
	}
}

func TestAddValidates(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "workspaces.json"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := r.Add("", "/tmp"); err == nil {
		t.Error("empty name accepted")
	}
	if _, err := r.Add("x", "/definitely/not/a/dir"); err == nil {
		t.Error("missing path accepted")
	}
	if _, err := r.Add("x", "/etc/hostname"); err == nil {
		t.Error("file (not dir) accepted")
	}
}
