package store

import (
	"errors"
	"strings"
	"testing"
)

func TestWebappCRUD(t *testing.T) {
	s := openTest(t)
	app, err := s.CreateWebapp(WebappInput{Name: " Example ", URL: "https://example.com/web?a=1", Icon: []byte("PNGDATA"), IconMime: "image/png"})
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
	first, err := s.CreateWebapp(WebappInput{Name: "Example", URL: "https://example.com", Icon: nil, IconMime: ""})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = s.CreateWebapp(WebappInput{Name: "Other", URL: "https://EXAMPLE.com/", Icon: nil, IconMime: ""})
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
		if _, err := s.CreateWebapp(WebappInput{Name: "App", URL: c.url}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: want ErrInvalid, got %v", c.name, err)
		}
	}
	if _, err := s.CreateWebapp(WebappInput{Name: "   ", URL: "https://example.com", Icon: nil, IconMime: ""}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank name: want ErrInvalid, got %v", err)
	}
	if _, err := s.CreateWebapp(WebappInput{Name: "App", URL: "https://example.com/#frag", Icon: nil, IconMime: ""}); err != nil {
		t.Fatalf("fragment url should be accepted: %v", err)
	}
}

func TestWebappGetByURL(t *testing.T) {
	s := openTest(t)
	app, _ := s.CreateWebapp(WebappInput{Name: "Example", URL: "https://example.com/x", Icon: nil, IconMime: ""})
	row, err := s.GetWebappByURL("https://example.com/x")
	if err != nil || row.ID != app.ID {
		t.Fatalf("by url = %+v err=%v", row, err)
	}
	if _, err := s.GetWebappByURL("https://other.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing = %v", err)
	}
}

func TestWebappManifestFieldsRoundTrip(t *testing.T) {
	s := openTest(t)
	app, err := s.CreateWebapp(WebappInput{
		Name: "Zulip", URL: "https://chat.example.com",
		StartURL: "https://chat.example.com/app/?source=pwa", Scope: "https://chat.example.com/app/",
		Display: "Standalone", ThemeColor: "#1D2B53",
		Icon: []byte("PNGDATA"), IconMime: "image/png",
	})
	if err != nil {
		t.Fatalf("create with manifest fields: %v", err)
	}
	if app.StartURL != "https://chat.example.com/app/?source=pwa" || app.Scope != "https://chat.example.com/app/" || app.Display != "standalone" || app.ThemeColor != "#1d2b53" {
		t.Fatalf("created = %+v", app)
	}
	list, _ := s.ListWebapps()
	if list[0].StartURL != app.StartURL || list[0].ThemeColor != "#1d2b53" {
		t.Fatalf("listed = %+v", list[0])
	}
	for _, bad := range []WebappInput{
		{Name: "Bad start", URL: "https://x.com", StartURL: "ftp://x.com"},
		{Name: "Bad scope", URL: "https://x.com", Scope: "https://user:pass@x.com/"},
		{Name: "Bad display", URL: "https://x.com", Display: "kiosk"},
	} {
		if _, err := s.CreateWebapp(bad); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%+v: want ErrInvalid, got %v", bad, err)
		}
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
	deep, err := s.CreateWebapp(WebappInput{Name: "Deep", URL: "https://example.com/#/inbox", Icon: nil, IconMime: ""})
	if err != nil {
		t.Fatalf("deep link: %v", err)
	}
	if deep.URL != "https://example.com/#/inbox" {
		t.Fatalf("fragment must survive: %q", deep.URL)
	}
	six, err := s.CreateWebapp(WebappInput{Name: "Six", URL: "http://[::1]:3000/ui", Icon: nil, IconMime: ""})
	if err != nil {
		t.Fatalf("ipv6 with port: %v", err)
	}
	if six.URL != "http://[::1]:3000/ui" {
		t.Fatalf("ipv6 url = %q", six.URL)
	}
	_, err = s.CreateWebapp(WebappInput{Name: "Long", URL: "https://example.com/" + strings.Repeat("a", maxWebAppURL)})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("overlong url = %v", err)
	}
}

func TestWebappIconAbsentAndMissingRow(t *testing.T) {
	s := openTest(t)
	app, _ := s.CreateWebapp(WebappInput{Name: "Bare", URL: "https://bare.example.com", Icon: nil, IconMime: ""})
	icon, mime, err := s.GetWebappIcon(app.ID)
	if err != nil || icon != nil || mime != "" {
		t.Fatalf("no-icon = %q %q err=%v", icon, mime, err)
	}
	if _, _, err := s.GetWebappIcon("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing row = %v", err)
	}
}
