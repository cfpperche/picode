package snips

import (
	"strings"
	"testing"
	"time"
)

func TestExpandGolden(t *testing.T) {
	cases := []struct {
		body   string
		values map[string]string
		ctx    map[string]string
		want   string
		miss   []string
	}{
		{"hello", nil, nil, "hello", nil},
		{"{{{{name}}}}", nil, nil, "{{name}}", nil},
		{"{{x={{{{}}", nil, nil, "{{", nil},
		{"{{name=foo}}bar}}", nil, nil, "foobar}}", nil},
		{"{{name=foo}bar}}", nil, nil, "foo}bar", nil},
		{"Create {{name}}", map[string]string{"name": "Button"}, nil, "Create Button", nil},
		{"{{a}} {{b}}", map[string]string{"a": "1"}, nil, "1 ", []string{"b"}},
		{"{{a}}{{a}}", map[string]string{"a": "z"}, nil, "zz", nil},
		{"{{n=}}", map[string]string{"n": ""}, nil, "", nil},
		{"{{n=def}}", map[string]string{"n": ""}, nil, "def", nil},
		{"{{n=def}}", map[string]string{"n": "hit"}, nil, "hit", nil},
		{"{{req}}", map[string]string{"req": ""}, nil, "", []string{"req"}},
		{"x{{cwd}}y", map[string]string{"cwd": "/evil"}, map[string]string{"cwd": "/ok"}, "x/oky", nil},
		{"{{branch}}", map[string]string{"branch": "from-values"}, map[string]string{}, "", nil},
		{"{{workspace}}", nil, nil, "", nil},
		{"keep {{x}}", map[string]string{"x": "a{{y}}b"}, nil, "keep a{{y}}b", nil},
		{"!`rm -rf /` and {{x}}", map[string]string{"x": "ok"}, nil, "!`rm -rf /` and ok", nil},
		{"echo \"{{msg}}\"", map[string]string{"msg": "hi"}, nil, "echo \"hi\"", nil},
	}
	for _, c := range cases {
		got, miss, err := Expand(c.body, c.values, c.ctx)
		if err != nil {
			t.Errorf("Expand(%q): %v", c.body, err)
			continue
		}
		if got != c.want {
			t.Errorf("Expand(%q) = %q, want %q", c.body, got, c.want)
		}
		if strings.Join(miss, ",") != strings.Join(c.miss, ",") {
			t.Errorf("Expand(%q) missing %q, want %q", c.body, miss, c.miss)
		}
	}
}

func TestExpandDate(t *testing.T) {
	got, miss, err := Expand("on {{date}}", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("date listed missing: %v", miss)
	}
	want := "on " + time.Now().UTC().Format("2006-01-02")
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	got, _, err = Expand("on {{date}}", nil, map[string]string{"date": "1999-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "on 1999-01-01" {
		t.Errorf("ctx date: %q", got)
	}
}

func TestExpandRejectsBadBody(t *testing.T) {
	if _, _, err := Expand("{{x", nil, nil); err == nil {
		t.Error("expected error")
	}
}
