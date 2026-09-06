package clisession

import (
	"github.com/cfpperche/picode/internal/session"
)

// PISource adapts the pi JSONL index (internal/session, ADR-0005) to the
// multi-CLI listing. Resume stays pi's own flow (the chat picker and the
// resume endpoint), so ResumeArgs stays empty here.
type PISource struct{}

func (PISource) CLI() string { return "pi" }

func (PISource) List(cwd string) ([]Summary, error) {
	// Machine-wide walks the whole sessions root; scoped reads only the
	// shared cwd bucket (ADR-0040). The workspace manage view additionally
	// unions each agent's private dir and ownership data — the pi surface
	// keeps using that richer endpoint; this adapter exists so the registry
	// answers uniformly for every catalog CLI.
	if cwd == "" {
		all, err := session.ListAll()
		if err != nil {
			return nil, err
		}
		return piSummaries(all), nil
	}
	all, err := session.ListDirs(session.Dir(cwd))
	if err != nil {
		return nil, err
	}
	return piSummaries(all), nil
}

// PIDirs summarizes pi JSONL from already-resolved session directories
// (newest first) — the escape hatch the server uses to union the shared
// cwd bucket with each agent's private dir (ADR-0040) for a workspace
// scope, which a plain cwd filter cannot express.
func PIDirs(dirs ...string) ([]Summary, error) {
	all, err := session.ListDirs(dirs...)
	if err != nil {
		return nil, err
	}
	return piSummaries(all), nil
}

func piSummaries(all []session.Summary) []Summary {
	out := make([]Summary, 0, len(all))
	for _, s := range all {
		model := s.Model
		if s.Provider != "" && model != "" {
			model = s.Provider + "/" + model
		}
		out = append(out, Summary{
			CLI:       "pi",
			ID:        s.ID,
			Path:      s.Path,
			Name:      s.Name,
			Cwd:       s.Cwd,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
			Preview:   s.Preview,
			Messages:  s.Messages,
			Size:      s.Size,
			Model:     model,
			Cost:      s.Cost,
		})
	}
	sortNewest(out)
	return out
}
