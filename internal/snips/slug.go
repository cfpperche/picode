package snips

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Slug hyphenates like store.newID, not slashres.sanitizeName.
// "Review PR" → "review-pr". Empty after sanitize → "".
func Slug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	dash := false
	for len(s) > 0 {
		r, w := utf8.DecodeRuneInString(s)
		s = s[w:]
		if r <= unicode.MaxASCII && (r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if b.Len() > 0 && !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > MaxSlug {
		out = strings.TrimRight(out[:MaxSlug], "-")
	}
	return out
}
