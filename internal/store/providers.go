package store

import "fmt"

// ProviderRefs is how many configured things point at each provider id.
// The providers page shows it before Sign out, so the confirm can name the
// blast radius instead of asking the person to remember it.
type ProviderRefs struct {
	Agents      int `json:"agents"`
	Automations int `json:"automations"`
}

// CountProviderRefs groups agents and automations by their configured
// provider. Rows with no provider (inheriting the default) are not counted
// against any id — they follow whatever the default becomes.
func (s *Store) CountProviderRefs() (map[string]ProviderRefs, error) {
	out := map[string]ProviderRefs{}
	add := func(query string, set func(*ProviderRefs, int)) error {
		rows, err := s.db.Query(query)
		if err != nil {
			return fmt.Errorf("store: count provider refs: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var n int
			if err := rows.Scan(&id, &n); err != nil {
				return fmt.Errorf("store: count provider refs: %w", err)
			}
			cur := out[id]
			set(&cur, n)
			out[id] = cur
		}
		return rows.Err()
	}
	if err := add(
		`SELECT provider, COUNT(1) FROM agents WHERE provider IS NOT NULL AND provider <> '' GROUP BY provider`,
		func(r *ProviderRefs, n int) { r.Agents = n },
	); err != nil {
		return nil, err
	}
	if err := add(
		`SELECT provider, COUNT(1) FROM automations WHERE provider IS NOT NULL AND provider <> '' GROUP BY provider`,
		func(r *ProviderRefs, n int) { r.Automations = n },
	); err != nil {
		return nil, err
	}
	return out, nil
}
