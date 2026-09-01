package cron

import (
	"testing"
	"time"
)

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseRejects(t *testing.T) {
	for _, expr := range []string{
		"", "* * * *", "* * * * * *", "60 * * * *", "* 24 * * *", "* * 0 * *", "* * * 13 *",
		"* * * * 8", "a * * * *", "*/0 * * * *", "5-1 * * * *", ",* * * * *", "1,,2 * * * *",
		"MON * * * *", "* * L * *",
	} {
		if _, err := Parse(expr); err == nil {
			t.Errorf("Parse(%q) accepted", expr)
		}
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		expr string
		when string
		want bool
	}{
		{"*/5 * * * *", "2026-09-01 10:05", true},
		{"*/5 * * * *", "2026-09-01 10:07", false},
		{"0 * * * *", "2026-09-01 10:00", true},
		{"7 * * * *", "2026-09-01 10:07", true},
		{"0 9 * * *", "2026-09-01 09:00", true},
		{"0 9 * * *", "2026-09-01 21:00", false},
		{"0 9 * * 1-5", "2026-09-01 09:00", true},  // Tuesday
		{"0 9 * * 1-5", "2026-09-05 09:00", false}, // Saturday
		{"0 9 * * 7", "2026-09-06 09:00", true},    // Sunday as 7
		{"0 9 * * 0", "2026-09-06 09:00", true},
		{"30 14 15 3 *", "2026-03-15 14:30", true},
		{"30 14 15 3 *", "2026-04-15 14:30", false},
		{"0 0 1 * 1", "2026-09-01 00:00", true}, // dom matches (1st), dow Tuesday
		{"0 0 1 * 1", "2026-09-07 00:00", true}, // dow matches (Monday), dom 7th
		{"0 0 1 * 1", "2026-09-08 00:00", false},
		{"1,15,30 * * * *", "2026-09-01 10:15", true},
		{"1-5/2 * * * *", "2026-09-01 10:03", true},
		{"1-5/2 * * * *", "2026-09-01 10:04", false},
	}
	for _, c := range cases {
		s, err := Parse(c.expr)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.expr, err)
		}
		if got := s.Matches(at(c.when)); got != c.want {
			t.Errorf("%q at %s = %v, want %v", c.expr, c.when, got, c.want)
		}
	}
	if (Schedule{}).Matches(at("2026-09-01 10:00")) {
		t.Fatal("zero schedule must match nothing")
	}
}

func TestNext(t *testing.T) {
	cases := []struct{ expr, from, want string }{
		{"*/5 * * * *", "2026-09-01 10:05", "2026-09-01 10:10"},
		{"0 9 * * *", "2026-09-01 09:00", "2026-09-02 09:00"},
		{"0 9 * * 1-5", "2026-09-04 09:30", "2026-09-07 09:00"}, // Fri → Mon
		{"30 14 15 3 *", "2026-09-01 00:00", "2027-03-15 14:30"},
		{"0 0 31 * *", "2026-09-01 00:00", "2026-10-31 00:00"},
	}
	for _, c := range cases {
		s, _ := Parse(c.expr)
		got, ok := s.Next(at(c.from))
		if !ok || !got.Equal(at(c.want)) {
			t.Errorf("%q next after %s = %v (%v), want %s", c.expr, c.from, got, ok, c.want)
		}
	}
	s, _ := Parse("0 0 30 2 *")
	if _, ok := s.Next(at("2026-09-01 00:00")); ok {
		t.Fatal("Feb 30 must never fire")
	}
	d, _ := Parse("0 9 * * *")
	if iv := d.Interval(at("2026-09-01 00:00")); iv != 24*time.Hour {
		t.Fatalf("daily interval = %v", iv)
	}
}
