package automate

import (
	"regexp"
	"testing"

	"github.com/cfpperche/picode/internal/cron"
)

func TestTemplatesAreWellFormed(t *testing.T) {
	slug := regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)
	cats := map[string]bool{CategoryQuality: true, CategoryMaintenance: true, CategoryReporting: true, CategorySecurity: true}
	seen := map[string]bool{}
	list := Templates()
	if len(list) < 6 {
		t.Fatalf("only %d templates", len(list))
	}
	for _, tp := range list {
		if !slug.MatchString(tp.ID) || seen[tp.ID] {
			t.Errorf("%q: bad or duplicate id", tp.ID)
		}
		seen[tp.ID] = true
		if tp.Name == "" || len(tp.Name) > 60 || tp.Description == "" || tp.Prompt == "" {
			t.Errorf("%s: empty or long fields", tp.ID)
		}
		if !cats[tp.Category] {
			t.Errorf("%s: category %q", tp.ID, tp.Category)
		}
		if _, err := cron.Parse(tp.Cron); err != nil {
			t.Errorf("%s: cron: %v", tp.ID, err)
		}
		if tp.MaxCostUSD <= 0 || (tp.MaxRuns == 0) != (tp.MaxRunsWindowMin == 0) {
			t.Errorf("%s: limits %v %d/%d", tp.ID, tp.MaxCostUSD, tp.MaxRuns, tp.MaxRunsWindowMin)
		}
	}
}
