package version

import (
	"strings"
	"testing"
)

func TestClientValueRoundTrips(t *testing.T) {
	c, ok := ParseClient(ClientValue("picode-mcp"))
	if !ok || c.Kind != "picode-mcp" || c.Protocol != APIProtocol || c.Build != Build() {
		t.Fatalf("round trip: %+v %v", c, ok)
	}
}

func TestParseClient(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		want Client
	}{
		{"", false, Client{}},
		{"garbage", false, Client{}},
		{"/1.0", false, Client{}},
		{"picode-mcp/0.7.0+abc; protocol=3", true, Client{"picode-mcp", "0.7.0+abc", 3}},
		{" shell/0.1.0 ;Protocol= 2 ", true, Client{"shell", "0.1.0", 2}},
		{"picode-cli/0.7.0", true, Client{"picode-cli", "0.7.0", 0}}, // no protocol: the oldest
		{"picode-cli/0.7.0; protocol=x", false, Client{}},
		{"picode-cli/0.7.0; protocol=-1", false, Client{}},
	}
	for _, c := range cases {
		got, ok := ParseClient(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseClient(%q) = %+v %v, want %+v %v", c.in, got, ok, c.want, c.ok)
		}
	}
	if !strings.Contains(ClientValue("x"), "; protocol=") {
		t.Fatal("ClientValue must carry the protocol")
	}
}
