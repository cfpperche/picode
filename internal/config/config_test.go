package config

import "testing"

func TestParsePort(t *testing.T) {
	cases := []struct {
		in      string
		want    PortConfig
		wantErr bool
	}{
		{"8445", PortConfig{8445, 8445}, false},
		{" 8445 ", PortConfig{8445, 8445}, false},
		{"8445-8460", PortConfig{8445, 8460}, false},
		{"", PortConfig{}, true},
		{"abc", PortConfig{}, true},
		{"8445-abc", PortConfig{}, true},
	}
	for _, c := range cases {
		got, err := ParsePort(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParsePort(%q) = %+v, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePort(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParsePort(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := (PortConfig{8445, 8446}).Validate(); err != nil {
		t.Errorf("valid range rejected: %v", err)
	}
	if err := (PortConfig{1, 200}).Validate(); err == nil {
		t.Error("wide range accepted")
	}
	if err := (PortConfig{80, 80}).Validate(); err != nil {
		t.Errorf("specific port rejected: %v", err)
	}
}

func TestPortString(t *testing.T) {
	if got := (PortConfig{8445, 8445}).String(); got != "8445" {
		t.Errorf("String() = %q", got)
	}
	if got := (PortConfig{8445, 8460}).String(); got != "8445-8460" {
		t.Errorf("String() = %q", got)
	}
}

func TestResolvePrecedence(t *testing.T) {
	t.Setenv("PICODE_HOST", "")
	t.Setenv("PICODE_PORT", "9000")
	t.Setenv("PICODE_DATA", "")
	t.Setenv("PICODE_INSECURE", "")

	// No DB setting → env wins.
	cfg, err := Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.Port.Min != 9000 || cfg.Port.Max != 9000 {
		t.Errorf("env port = %+v, want 9000", cfg.Port)
	}

	// DB setting wins over env.
	cfg, err = Resolve(func(key string) (string, bool, error) {
		if key == PortSettingKey {
			return "9100-9110", true, nil
		}
		return "", false, nil
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.Port.Min != 9100 || cfg.Port.Max != 9110 {
		t.Errorf("db port = %+v, want 9100-9110", cfg.Port)
	}

	// Nothing set → default range.
	t.Setenv("PICODE_PORT", "")
	cfg, err = Resolve(func(string) (string, bool, error) { return "", false, nil })
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.Port.String() != DefaultPortRange {
		t.Errorf("default port = %s, want %s", cfg.Port, DefaultPortRange)
	}
}

func TestValidateHost(t *testing.T) {
	for _, ok := range []string{"0.0.0.0", "127.0.0.1", " 100.87.149.83 ", "::"} {
		if _, err := ValidateHost(ok); err != nil {
			t.Errorf("ValidateHost(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{"", "localhost", "box.tail.ts.net", "0.0.0.0:8445"} {
		if _, err := ValidateHost(bad); err == nil {
			t.Errorf("ValidateHost(%q) accepted", bad)
		}
	}
}

func TestValidatePublicURL(t *testing.T) {
	cases := []struct {
		in       string
		insecure bool
		want     string
		wantErr  bool
	}{
		{"", false, "", false},
		{"https://box.tail.ts.net:8445", false, "https://box.tail.ts.net:8445", false},
		{"https://BOX.tail.ts.net:8445/", false, "https://box.tail.ts.net:8445", false},
		{"http://box:8445", false, "", true},
		{"http://box:8445", true, "http://box:8445", false},
		{"https://box:8445/app", false, "", true},
		{"https://user:pw@box", false, "", true},
		{"box:8445", false, "", true},
		{"ftp://box", false, "", true},
	}
	for _, c := range cases {
		got, err := ValidatePublicURL(c.in, c.insecure)
		if c.wantErr != (err != nil) || got != c.want {
			t.Errorf("ValidatePublicURL(%q, %v) = %q, %v; want %q, err=%v", c.in, c.insecure, got, err, c.want, c.wantErr)
		}
	}
}

func TestResolveHostPrecedence(t *testing.T) {
	t.Setenv("PICODE_HOST", "127.0.0.1")
	t.Setenv("PICODE_PORT", "")
	db := map[string]string{}
	get := func(k string) (string, bool, error) { v, ok := db[k]; return v, ok, nil }
	cfg, err := Resolve(get)
	if err != nil || cfg.Host != "127.0.0.1" || cfg.PublicURL != "" {
		t.Fatalf("env host: %+v %v", cfg, err)
	}
	db[HostSettingKey] = "100.64.0.9"
	db[PublicURLSettingKey] = "https://box.tail.ts.net:8445"
	cfg, _ = Resolve(get)
	if cfg.Host != "100.64.0.9" || cfg.PublicURL != "https://box.tail.ts.net:8445" {
		t.Fatalf("db wins: %+v", cfg)
	}
	db[HostSettingKey] = "not-an-ip"
	if cfg, _ = Resolve(get); cfg.Host != "127.0.0.1" {
		t.Fatalf("bad db host should fall back to env: %+v", cfg)
	}
}
