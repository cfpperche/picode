package share

import (
	"net"
	"testing"
)

func TestUsablePhoneIP(t *testing.T) {
	cases := []struct {
		ip string
		ok bool
	}{
		{"127.0.0.1", false},
		{"169.254.1.1", false},
		{"172.17.0.2", false},
		{"10.255.255.254", false},
		{"192.168.15.28", true},
		{"10.0.0.5", true},
		{"100.87.149.83", true},
	}
	for _, c := range cases {
		if got := UsablePhoneIP(net.ParseIP(c.ip)); got != c.ok {
			t.Errorf("UsablePhoneIP(%s) = %v, want %v", c.ip, got, c.ok)
		}
	}
}

func TestDiagnoseInsecureFailsHTTPS(t *testing.T) {
	r := Diagnose(Input{Insecure: true, BindHost: "0.0.0.0", Port: 8445})
	if r.Ready {
		t.Fatal("insecure must not be ready")
	}
	found := false
	for _, c := range r.Checks {
		if c.ID == "https" && !c.OK && c.Action != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing https action: %+v", r.Checks)
	}
}

func TestDetectPhoneOS(t *testing.T) {
	if detectPhoneOS("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)") != "ios" {
		t.Fatal("iphone")
	}
	if detectPhoneOS("Mozilla/5.0 (Linux; Android 14)") != "android" {
		t.Fatal("android")
	}
	if detectPhoneOS("Mozilla/5.0 (Windows NT 10.0)") != "other" {
		t.Fatal("desktop")
	}
}

func TestMissingAny(t *testing.T) {
	have := []string{"localhost", "192.168.15.28"}
	if !missingAny(have, []string{"localhost", "192.168.15.110"}) {
		t.Fatal("should detect new LAN IP")
	}
	if missingAny(have, []string{"localhost"}) {
		t.Fatal("existing name flagged")
	}
}

func TestDiagnoseLoopbackBindFails(t *testing.T) {
	r := Diagnose(Input{Insecure: false, BindHost: "127.0.0.1", Port: 8445})
	for _, c := range r.Checks {
		if c.ID == "bind" && c.OK {
			t.Fatal("loopback bind should fail")
		}
	}
}
