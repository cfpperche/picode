package hookurl

import (
	"net"
	"testing"
)

func TestDestinationPolicy(t *testing.T) {
	for _, tc := range []struct {
		ip    string
		allow bool
	}{
		{"127.0.0.1", true}, {"::1", true}, {"192.168.1.1", true}, {"100.100.1.1", true}, {"8.8.8.8", true},
		{"169.254.169.254", false}, {"::ffff:169.254.169.254", false}, {"fe80::1", false}, {"0.0.0.0", false}, {"::", false}, {"224.0.0.1", false}, {"ff02::1", false},
	} {
		t.Run(tc.ip, func(t *testing.T) {
			if AllowedIP(net.ParseIP(tc.ip)) != tc.allow {
				t.Fatal("address policy")
			}
		})
	}
	for _, raw := range []string{"https://metadata.google.internal/", "http://[fe80::1%25eth0]/", "https://example.com:99999/", "https://u:p@example.com", "file:///tmp/x", "https://"} {
		if Validate(raw) == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
