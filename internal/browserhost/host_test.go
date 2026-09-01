package browserhost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsHostArg(t *testing.T) {
	if !IsHostArg("browser-host") {
		t.Fatal("browser-host")
	}
	if !IsHostArg(ExtensionOrigin) {
		t.Fatal("pinned origin")
	}
	if !IsHostArg("chrome-extension://abc/") {
		t.Fatal("any extension origin")
	}
	if IsHostArg("install") || IsHostArg("") {
		t.Fatal("false positive")
	}
}

func TestFrameRoundTrip(t *testing.T) {
	raw, err := EncodeFrame(Request{Type: "ping", ID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "ping" || got.ID != "1" {
		t.Fatalf("%+v", got)
	}
}

func TestReadRejectsOversize(t *testing.T) {
	var hdr [4]byte
	hdr[0] = 0xff
	hdr[1] = 0xff
	hdr[2] = 0xff
	hdr[3] = 0x7f
	_, err := Read(bytes.NewReader(hdr[:]))
	if err == nil || !strings.Contains(err.Error(), "1 MB") {
		t.Fatalf("got %v", err)
	}
}

func TestServeEOF(t *testing.T) {
	var out bytes.Buffer
	if err := Serve(bytes.NewReader(nil), &out, func(Request) Reply {
		t.Fatal("handler")
		return Reply{}
	}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote %d", out.Len())
	}
}

func TestServeOnePing(t *testing.T) {
	in, err := EncodeFrame(Request{Type: "ping", ID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Serve(bytes.NewReader(in), &out, func(Request) Reply {
		return Reply{OK: true, URL: "https://localhost:8445"}
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := readFrame(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	var reply Reply
	if err := json.Unmarshal(raw, &reply); err != nil {
		t.Fatal(err)
	}
	if !reply.OK || reply.ID != "a" || reply.Type != "ping" || reply.URL == "" {
		t.Fatalf("%+v", reply)
	}
}

func TestWriteHostManifest(t *testing.T) {
	home := t.TempDir()
	exe := filepath.Join(home, "bin", "picode")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := WriteHostManifest(home, exe)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(ChromeHostDir(home), HostName+".json")
	if path != want {
		t.Fatalf("path %s want %s", path, want)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m.Name != HostName || m.Path != exe || m.Type != "stdio" {
		t.Fatalf("%+v", m)
	}
	if len(m.AllowedOrigins) != 1 || m.AllowedOrigins[0] != ExtensionOrigin {
		t.Fatalf("origins %v", m.AllowedOrigins)
	}
	if err := RemoveHostManifest(home); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("still there: %v", err)
	}
}

func TestWriteHostManifestRequiresAbs(t *testing.T) {
	_, err := WriteHostManifest(t.TempDir(), "picode")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWindowsHostPath(t *testing.T) {
	dir := t.TempDir()
	desktop := filepath.Join(dir, "picode-desktop.exe")
	if _, err := WindowsHostPath(desktop); err == nil {
		t.Fatal("missing nmh should fail")
	}
	nmh := filepath.Join(dir, WindowsHostExe)
	if err := os.WriteFile(nmh, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := WindowsHostPath(desktop)
	if err != nil || got != nmh {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestCopyFileSamePath(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(p, p); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "b")
	if err := CopyFile(p, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dst)
	if err != nil || string(b) != "hi" {
		t.Fatalf("%q %v", b, err)
	}
}

func TestWindowsRegistryArgs(t *testing.T) {
	add := WindowsRegistryAddArgs(`C:\PiCode\com.picode.browser.json`)
	if add[0] != "add" || add[len(add)-1] != "/f" {
		t.Fatalf("%v", add)
	}
	if !strings.Contains(add[1], HostName) {
		t.Fatalf("key %s", add[1])
	}
	del := WindowsRegistryDeleteArgs()
	if del[0] != "delete" {
		t.Fatalf("%v", del)
	}
}

func TestClientUnknownType(t *testing.T) {
	c := &Client{Resolve: func() (string, error) { return "http://127.0.0.1", nil }}
	got := c.Handle(Request{Type: "wipe", ID: "x"})
	if got.OK || got.Code != "bad_type" || got.ID != "x" {
		t.Fatalf("%+v", got)
	}
}

func TestClientPingDown(t *testing.T) {
	c := &Client{Resolve: func() (string, error) { return "", fmt.Errorf("nope") }}
	got := c.Handle(Request{Type: "ping"})
	if got.OK || got.Code != "picode_down" {
		t.Fatalf("%+v", got)
	}
}

func TestReadServerURL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PICODE_DATA", dir)
	_, err := ReadServerURL()
	if err == nil {
		t.Fatal("missing file should fail")
	}
	if err := os.WriteFile(filepath.Join(dir, "server.json"), []byte(`{"url":"https://localhost:8446"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	u, err := ReadServerURL()
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://localhost:8446" {
		t.Fatal(u)
	}
}
