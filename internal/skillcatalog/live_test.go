package skillcatalog

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveSeeds reads the three seed repositories and skills.sh for real
// (PICODE_SKILLS_LIVE=1): each seed lists skills with a description and an
// install input the Add skill dialog accepts.
func TestLiveSeeds(t *testing.T) {
	if os.Getenv("PICODE_SKILLS_LIVE") != "1" {
		t.Skip("set PICODE_SKILLS_LIVE=1")
	}
	s := NewStore("")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, seed := range Seeds {
		items, err := s.read(ctx, seed)
		if err != nil || len(items) == 0 {
			t.Fatalf("%s: %d items, %v", seed, len(items), err)
		}
		for _, it := range items {
			if it.Description == "" || it.Install == "" {
				t.Fatalf("%s: %+v", seed, it)
			}
		}
		t.Logf("%s: %d skills, e.g. %s (%s)", seed, len(items), items[0].Name, items[0].Install)
	}
	hits, err := s.SearchSkillsSH(ctx, "pdf")
	if err != nil || len(hits) == 0 {
		t.Fatalf("skills.sh: %v", err)
	}
	t.Logf("skills.sh pdf: %d hits, first %s from %s", len(hits), hits[0].Name, hits[0].Source)
}
