package presence

import "testing"

func TestLabel(t *testing.T) {
	if Label("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)") != "iPhone" {
		t.Fatal("iphone")
	}
	if Label("Mozilla/5.0 (Linux; Android 14; Mobile)") != "Android" {
		t.Fatal("android")
	}
	if Label("Mozilla/5.0 (Windows NT 10.0)") != "Windows" {
		t.Fatal("windows")
	}
	if Label("Mozilla/5.0 (X11; Linux x86_64) HeadlessChrome/152.0.7977.64") != "Headless browser" {
		t.Fatal("headless chrome must not read as a Linux machine")
	}
}

func TestPingHostAndRemote(t *testing.T) {
	r := New([]string{"192.168.15.110"})
	host := r.Ping("aaa", "Mozilla/5.0 (Windows NT 10.0)", "127.0.0.1:1234", false, "")
	if !host.Host || !host.Online || host.Name != "Windows" {
		t.Fatalf("host = %+v", host)
	}
	phone := r.Ping("bbb", "Mozilla/5.0 (iPhone)", "100.87.149.83:9999", false, "")
	if phone.Host || phone.Name != "iPhone" || phone.IP != "100.87.149.83" {
		t.Fatalf("phone = %+v", phone)
	}
	list := r.List()
	if len(list) != 2 || !list[0].Host {
		t.Fatalf("list = %+v", list)
	}
}

func TestPingExtension(t *testing.T) {
	r := New(nil)
	d := r.Ping("ext:abc", "Go-http-client/1.1", "127.0.0.1:9", true, "extension")
	if d.Name != "Chrome extension" || d.Kind != "extension" || !d.Host {
		t.Fatalf("%+v", d)
	}
}

func TestStripPort(t *testing.T) {
	if stripPort("192.168.15.110:8445") != "192.168.15.110" {
		t.Fatal(stripPort("192.168.15.110:8445"))
	}
}

func TestAnyHostOnline(t *testing.T) {
	r := New(nil)
	if r.AnyHostOnline() {
		t.Fatal("empty registry is not online")
	}
	r.Ping("phone", "iPhone", "100.64.0.9:1", false, "")
	if r.AnyHostOnline() {
		t.Fatal("a phone on the tailnet is not the host")
	}
	r.Ping("ext", "Chrome", "127.0.0.1:2", true, "extension")
	if r.AnyHostOnline() {
		t.Fatal("the Chrome extension pinging is not a person at the desk")
	}
	r.Ping("desk", "Chrome", "127.0.0.1:3", true, "")
	if !r.AnyHostOnline() {
		t.Fatal("a host browser is online")
	}
}
