// The Pi pane's JSON, derived from the unified model (ADR-0176 slice 1c). The
// legacy routes answer from the driver so there is one source of truth, while
// the bytes the pane parses stay exactly what `pipkg` answered before: these
// mappers are pure, so the two can be compared in a test instead of trusted.
package pkgs

import "github.com/cfpperche/picode/internal/pipkg"

// Legacy is GET /api/packages as the Pi pane reads it: Scope keeps the CLI's
// own word (Row.Vendor), and Row.Enabled is the inverse of the settings file's
// `filtered` flag — stored, not loaded.
func (r Report) Legacy() pipkg.Report {
	out := pipkg.Report{
		Packages:     make([]pipkg.Pkg, 0, len(r.Rows)),
		Capabilities: pipkg.Capabilities{WebSearch: r.Capabilities["webSearch"]},
		Gallery:      r.Gallery,
		Isolated:     r.Isolated,
	}
	for _, row := range r.Rows {
		out.Packages = append(out.Packages, pipkg.Pkg{
			Source:        row.Source,
			Scope:         row.Vendor,
			Kind:          row.Kind,
			Filtered:      !row.Enabled,
			InstalledPath: row.InstalledPath,
			ConfigKind:    row.ConfigKind,
		})
	}
	return out
}

// LegacyUpdates is GET /api/packages/updates: the rows whose catalog moved
// ahead, each with the source to update, the scope it lives in and the pair
// the pane shows. A row with no catalog version is not behind and is dropped.
func LegacyUpdates(rows []Row) pipkg.UpdateReport {
	out := pipkg.UpdateReport{Updates: make([]pipkg.Available, 0, len(rows))}
	for _, row := range rows {
		if row.Behind == "" {
			continue
		}
		out.Updates = append(out.Updates, pipkg.Available{
			Source:  row.Source,
			Scope:   row.Vendor,
			Current: row.Version,
			Latest:  row.Behind,
		})
	}
	return out
}
