package snips

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseGolden(t *testing.T) {
	cases := []struct {
		body string
		want []Placeholder
	}{
		{"", nil},
		{"just text", nil},
		{"{{name}}", []Placeholder{{Name: "name"}}},
		{"{{ name }}", []Placeholder{{Name: "name"}}},
		{"{{name=foo}}", []Placeholder{{Name: "name", Default: "foo", Optional: true}}},
		{"{{n=}}", []Placeholder{{Name: "n", Optional: true}}},
		{"{{name=foo}bar}}", []Placeholder{{Name: "name", Default: "foo}bar", Optional: true}}},
		{"{{name=foo}}bar}}", []Placeholder{{Name: "name", Default: "foo", Optional: true}}},
		{"hello {{a}} and {{b=x}}", []Placeholder{
			{Name: "a"},
			{Name: "b", Default: "x", Optional: true},
		}},
		{"{{x}}{{x=ignored}}", []Placeholder{{Name: "x"}}}, // first-wins
		{"{{x=first}}{{x=second}}", []Placeholder{{Name: "x", Default: "first", Optional: true}}},
		{"{{{{name}}}}", nil},
		{"{{{{", nil},
		{"}}}}", nil},
		{"{{x={{{{}}", []Placeholder{{Name: "x", Default: "{{", Optional: true}}},
		{"{{cwd}} {{branch}}", []Placeholder{{Name: "cwd"}, {Name: "branch"}}},
		{"before !`rm` after {{x}}", []Placeholder{{Name: "x"}}},
		{"{{env=dev}}", []Placeholder{{Name: "env", Default: "dev", Optional: true}}},
	}
	for _, c := range cases {
		got, err := Parse(c.body)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.body, err)
			continue
		}
		if got == nil {
			got = []Placeholder{}
		}
		want := c.want
		if want == nil {
			want = []Placeholder{}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Parse(%q) = %#v, want %#v", c.body, got, want)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, body := range []string{
		"{{",
		"{{x",
		"{{x}",
		"{{1abc}}",
		"{{Not Valid}}",
		"{{#if}}",
		"{{-x}}",
		"{{x=unterminated",
	} {
		if _, err := Parse(body); err == nil {
			t.Errorf("Parse(%q) accepted", body)
		}
	}
	long := "{{" + strings.Repeat("a", MaxName+1) + "}}"
	if _, err := Parse(long); err == nil {
		t.Error("accepted overlong name")
	}
	var many strings.Builder
	for i := 0; i < MaxPlaceholders+1; i++ {
		many.WriteString("{{n")
		many.WriteString(strings.Repeat("x", 1))
		many.WriteByte('0' + byte(i/10))
		many.WriteByte('0' + byte(i%10))
		many.WriteString("}}")
	}
	if _, err := Parse(many.String()); err == nil {
		t.Error("accepted too many placeholders")
	}
}

func TestParseDoesNotUnescapeStoredShape(t *testing.T) {
	body := "see {{{{name}}}} here"
	got, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("escaped braces became placeholder: %#v", got)
	}
}
