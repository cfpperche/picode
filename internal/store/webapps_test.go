package store

import (
	"errors"
	"strings"
	"testing"
)

func TestWebappCRUD(t *testing.T) {
	s := openTest(t)
	app, err := s.CreateWebapp(" Example ", "https://example.com/web?a=1", []byte("PNGDATA"), "image/png")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if app.ID == "" || app.Name != "Example" || app.URL != "https://example.com/web?a=1" || !app.HasIcon || app.CreatedAt == "" {
		t.Fatalf("created = %+v", app)
	}
	list, err := s.ListWebapps()
	if err != nil || len(list) != 1 || list[0].ID != app.ID || !list[0].HasIcon {
		t.Fatalf("list = %+v err=%v", list, err)
	}
	renamed, err := s.UpdateWebappName(app.ID, "Renamed App")
	if err != nil || renamed.Name != "Renamed App" {
		t.Fatalf("rename = %+v err=%v", renamed, err)
	}
	icon, mime, err := s.GetWebappIcon(app.ID)
	if err != nil || string(icon) != "PNGDATA" || mime != "image/png" {
		t.Fatalf("icon = %q %q err=%v", icon, mime, err)
	}
	if err := s.DeleteWebapp(app.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.DeleteWebapp(app.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete = %v", err)
	}
	list, _ = s.ListWebapps()
	if len(list) != 0 {
		t.Fatalf("list after delete = %+v", list)
	}
}

func TestWebappDuplicateURLIsConflictWithExisting(t *testing.T) {
	s := openTest(t)
	first, err := s.CreateWebapp("Example", "https://example.com", nil, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = s.CreateWebapp("Other", "https://EXAMPLE.com/", nil, "")
	var dup DuplicateWebappError
	if !errors.As(err, &dup) {
		t.Fatalf("want DuplicateWebappError, got %v", err)
	}
	if dup.Existing.ID != first.ID {
		t.Fatalf("dup.Existing = %+v want %s", dup.Existing, first.ID)
	}
}

func TestWebappValidation(t *testing.T) {
	s := openTest(t)
	cases := []struct {
		name, url string
	}{
		{"scheme refused", "ftp://example.com"},
		{"javascript refused", "javascript:alert(1)"},
		{"credentials refused", "https://user:pass@example.com"},
		{"no host", "https://"},
		{"empty", ""},
		{"empty name", ""},
	}
	for _, c := range cases {
		if _, err := s.CreateWebapp("App", c.url, nil, ""); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: want ErrInvalid, got %v", c.name, err)
		}
	}
	if _, err := s.CreateWebapp("   ", "https://example.com", nil, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank name: want ErrInvalid, got %v", err)
	}
	if _, err := s.CreateWebapp("App", "https://example.com/#frag", nil, ""); err != nil {
		t.Fatalf("fragment url should be accepted: %v", err)
	}
}

func TestWebappGetByURL(t *testing.T) {
	s := openTest(t)
	app, _ := s.CreateWebapp("Example", "https://example.com/x", nil, "")
	row, err := s.GetWebappByURL("https://example.com/x")
	if err != nil || row.ID != app.ID {
		t.Fatalf("by url = %+v err=%v", row, err)
	}
	if _, err := s.GetWebappByURL("https://other.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing = %v", err)
	}
}

func TestWebappURLCanonicalization(t *testing.T) {
	s := openTest(t)
	for raw, want := range map[string]string{
		"127.0.0.1:18999":   "http://127.0.0.1:18999",
		"localhost:3000/ui": "http://localhost:3000/ui",
		"example.com":       "https://example.com",
		"10.0.0.5":          "http://10.0.0.5",
	} {
		got, err := NormalizeWebappURL(raw)
		if err != nil || got != want {
			t.Fatalf("normalize(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	deep, err := s.CreateWebapp("Deep", "https://example.com/#/inbox", nil, "")
	if err != nil {
		t.Fatalf("deep link: %v", err)
	}
	if deep.URL != "https://example.com/#/inbox" {
		t.Fatalf("fragment must survive: %q", deep.URL)
	}
	six, err := s.CreateWebapp("Six", "http://[::1]:3000/ui", nil, "")
	if err != nil {
		t.Fatalf("ipv6 with port: %v", err)
	}
	if six.URL != "http://[::1]:3000/ui" {
		t.Fatalf("ipv6 url = %q", six.URL)
	}
	_, err = s.CreateWebapp("Long", "https://example.com/"+strings.Repeat("a", maxWebAppURL), nil, "")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("overlong url = %v", err)
	}
}

func TestWebappIconAbsentAndMissingRow(t *testing.T) {
	s := openTest(t)
	app, _ := s.CreateWebapp("Bare", "https://bare.example.com", nil, "")
	icon, mime, err := s.GetWebappIcon(app.ID)
	if err != nil || icon != nil || mime != "" {
		t.Fatalf("no-icon = %q %q err=%v", icon, mime, err)
	}
	if _, _, err := s.GetWebappIcon("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing row = %v", err)
	}
}
