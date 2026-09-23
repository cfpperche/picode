package store

// The reverse of the mission's evidence link. A mission may cite a delivery as
// evidence (Kind "delivery", Value = the delivery id), and the Delivery
// surfaces want the other direction: which objective does this change serve?
// The mission owns the reference and its authority; nothing here writes, and a
// link never transfers integration authority (ADR-0199, ADR-0186).

// MissionLink names an objective a change serves — the part of a mission a
// delivery row needs, not the mission.
type MissionLink struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
}

// MissionsByDelivery inverts the evidence link for one workspace: delivery id →
// the missions that cite it, each mission named once however many of its
// criteria point at the same delivery. The list is paged to a bound above the
// store's own mission cap; `truncated` says so rather than letting the caller
// believe a short answer is a complete one.
func (s *Store) MissionsByDelivery(workspace string) (map[string][]MissionLink, bool, error) {
	out := map[string][]MissionLink{}
	before := int64(0)
	for range 25 {
		missions, err := s.ListMissions(workspace, before)
		if err != nil {
			return nil, false, err
		}
		for _, m := range missions {
			seen := map[string]bool{}
			for _, e := range m.Evidence {
				if e.Kind != "delivery" || e.Value == "" {
					continue
				}
				if seen[e.Value] {
					continue
				}
				seen[e.Value] = true
				out[e.Value] = append(out[e.Value], MissionLink{ID: m.ID, Title: m.Title, State: m.State})
			}
		}
		if len(missions) <= 100 {
			return out, false, nil
		}
		before = missions[len(missions)-1].Sequence
	}
	return out, true, nil
}
