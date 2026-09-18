package mcptool

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveCaptureKeepsTheLastFifty(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	for i := 0; i < captureKeep+5; i++ {
		if _, err := SaveCapture(dir, "term:x/y", "screenshot", "image/png", "iVBORw0KGgo=", base.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "term_x_y"))
	if len(entries) != captureKeep {
		t.Fatalf("kept %d, want %d", len(entries), captureKeep)
	}
	if entries[0].Name() < "20260918-000005" {
		t.Fatalf("the oldest were not the ones dropped: %s", entries[0].Name())
	}
	if p, err := SaveCapture("", "p", "a", "image/png", "x", base); p != "" || err != nil {
		t.Fatal("no dir means no file, no error")
	}
	if _, err := SaveCapture(dir, "p", "a", "image/png", "not base64!", base); err == nil {
		t.Fatal("bad base64 must be reported")
	}
	p, err := SaveCapture(dir, "", "browser", "image/jpeg", "/9j/", base)
	if err != nil || filepath.Base(filepath.Dir(p)) != "anonymous" || filepath.Ext(p) != ".jpg" {
		t.Fatalf("path = %s %v", p, err)
	}
}
