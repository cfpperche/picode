package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
)

// ExitSkillSide counts one side of a comparison: the exits that loaded a
// skill, or the recorded exits that did not.
type ExitSkillSide struct {
	Total      int `json:"total"`
	Resolved   int `json:"resolved"`
	Partial    int `json:"partial"`
	Unresolved int `json:"unresolved"`
	Trial      int `json:"trial"`
	Unanswered int `json:"unanswered"`
	// Cost is summed over the exits whose sessions were measured.
	Cost         float64 `json:"cost"`
	CostMeasured int     `json:"costMeasured"`
}

func (s *ExitSkillSide) add(outcome string, cost *ExitCost) {
	s.Total++
	switch outcome {
	case ExitResolved:
		s.Resolved++
	case ExitPartial:
		s.Partial++
	case ExitUnresolved:
		s.Unresolved++
	case ExitTrial:
		s.Trial++
	default:
		s.Unanswered++
	}
	if cost != nil {
		s.Cost += cost.Cost
		s.CostMeasured++
	}
}

// ExitSkillRow compares, for one skill, the exits that loaded it with the
// exits that recorded a skill set without it (ADR-0196 slice 6). Versions
// counts the distinct contents seen, so a skill edited mid-way shows it.
type ExitSkillRow struct {
	Name     string        `json:"name"`
	Versions int           `json:"versions"`
	With     ExitSkillSide `json:"with"`
	Without  ExitSkillSide `json:"without"`
}

// ExitSkillStats is the comparison over a filter. Recorded exits carry a
// skill set; Unrecorded ones (before slice 6, or a CLI PiCode could not read)
// are on neither side of any row.
type ExitSkillStats struct {
	Recorded   int            `json:"recorded"`
	Unrecorded int            `json:"unrecorded"`
	Rows       []ExitSkillRow `json:"rows"`
}

// AgentExitSkillStats compares outcomes with and without each skill over the
// exits the filter selects; f.Limit and f.Before are ignored.
func (s *Store) AgentExitSkillStats(f ExitFilter) (ExitSkillStats, error) {
	f.Before, f.Limit = "", 0
	where, args := f.where()
	rows, err := s.db.Query(`SELECT outcome, config, cost FROM agent_exits`+where, args...)
	if err != nil {
		return ExitSkillStats{}, fmt.Errorf("store: exit skills: %w", err)
	}
	defer rows.Close()
	type exit struct {
		outcome string
		cost    *ExitCost
		names   map[string]bool
	}
	var recorded []exit
	digests := map[string]map[string]bool{}
	out := ExitSkillStats{Rows: []ExitSkillRow{}}
	for rows.Next() {
		var outcome, cfg string
		var cst sql.NullString
		if err := rows.Scan(&outcome, &cfg, &cst); err != nil {
			return ExitSkillStats{}, fmt.Errorf("store: exit skills: %w", err)
		}
		var c ExitConfig
		if json.Unmarshal([]byte(cfg), &c) != nil || c.Loaded == nil {
			out.Unrecorded++
			continue
		}
		ex := exit{outcome: outcome, names: map[string]bool{}}
		if cst.Valid && cst.String != "" {
			var cost ExitCost
			if json.Unmarshal([]byte(cst.String), &cost) == nil {
				ex.cost = &cost
			}
		}
		for _, sk := range c.Loaded.Skills {
			ex.names[sk.Name] = true
			if digests[sk.Name] == nil {
				digests[sk.Name] = map[string]bool{}
			}
			if sk.Digest != "" {
				digests[sk.Name][sk.Digest] = true
			}
		}
		recorded = append(recorded, ex)
	}
	if err := rows.Err(); err != nil {
		return ExitSkillStats{}, err
	}
	out.Recorded = len(recorded)
	for name, ds := range digests {
		row := ExitSkillRow{Name: name, Versions: len(ds)}
		for _, ex := range recorded {
			if ex.names[name] {
				row.With.add(ex.outcome, ex.cost)
			} else {
				row.Without.add(ex.outcome, ex.cost)
			}
		}
		out.Rows = append(out.Rows, row)
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		if out.Rows[i].With.Total != out.Rows[j].With.Total {
			return out.Rows[i].With.Total > out.Rows[j].With.Total
		}
		return out.Rows[i].Name < out.Rows[j].Name
	})
	return out, nil
}
