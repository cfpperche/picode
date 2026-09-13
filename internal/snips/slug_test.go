package snips

import "testing"

func TestSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Review PR", "review-pr"},
		{"  review-pr  ", "review-pr"},
		{"review_pr", "review-pr"},
		{"ABC", "abc"},
		{"a--b", "a-b"},
		{"***", ""},
		{"", ""},
		{"123 Review", "123-review"},
		{"déjà vu", "d-j-vu"},
		{stringsRepeat("a", 80), stringsRepeat("a", MaxSlug)},
	}
	for _, c := range cases {
		if got := Slug(c.in); got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
