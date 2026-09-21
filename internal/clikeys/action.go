package clikeys

// Action is one row of a CLI's key map as the pane renders it: the CLI's own id
// and its own description, the chords it binds out of the box, and — where one
// platform binds something else — the whole binding there rather than an
// addition to it.
//
// It is the envelope's vocabulary (ADR-0174), which is why Pi's catalog is
// reported through it as well; pikeys keeps its own row type because it answers
// `/api/pi-keys` from it too, and the envelope converts rather than joining the
// two packages.
type Action struct {
	ID       string   `json:"id"`
	Group    string   `json:"group"`
	Label    string   `json:"label"`
	Defaults []string `json:"defaults"`
	// Alt is keyed by the CLI's own platform names ("win32", "linux",
	// "darwin" for Omp), never by PiCode's. An empty list is a real answer: the
	// CLI binds nothing there.
	Alt map[string][]string `json:"alt,omitempty"`
}
