// Package termopts is PiCode's layer over the tmux option space (ADR-0024).
//
// Two tiers. FEATURED options carry curated metadata — label, help, effect,
// warning — and include everything PiCode itself used to force in code:
// making a force into a default with the warning beside it is what lets the
// user's override win without a hardcoded exception. EVERYTHING ELSE comes
// from the live catalog (internal/tmux.OptionCatalog): the full option space
// of the tmux actually installed, validated by tmux itself at set-option
// time rather than by a whitelist here.
package termopts

import "fmt"

// GlobalScope is the store key holding the defaults every terminal inherits.
// Terminal overrides are stored under the terminal's own id.
const GlobalScope = "global"

// When a change to a flag takes hold.
const (
	EffectLive     = "live"      // the running session picks it up, no reattach
	EffectNewPanes = "new-panes" // existing panes keep what they have
	EffectServer   = "server"    // one value for the whole tmux server, at once
)

// Flag is one curated option.
type Flag struct {
	Key     string   `json:"key"`
	Scope   string   `json:"scope"` // tmux.ScopeServer/Session/Window
	Label   string   `json:"label"`
	Help    string   `json:"help"`
	Effect  string   `json:"effect"`
	Default string   `json:"default"` // PiCode's default, applied to owned sessions
	Values  []string `json:"values"`
	Danger  string   `json:"danger,omitempty"` // consequence of leaving the default, shown beside the control
}

var flags = []Flag{{
	Key:     "mouse",
	Scope:   "session",
	Label:   "Mouse belongs to the terminal",
	Effect:  EffectLive,
	Default: "on",
	Values:  []string{"on", "off"},
	Help: "On, the wheel scrolls this terminal's history — the only thing " +
		"that scrolls a program that ignores the mouse. Off, dragging selects " +
		"text the way it does anywhere else. Shift and drag always selects.",
}, {
	Key:     "status",
	Scope:   "session",
	Label:   "tmux status bar",
	Effect:  EffectLive,
	Default: "off",
	Values:  []string{"off", "on"},
	Help: "tmux's own status line at the bottom of the terminal. PiCode " +
		"turns it off because the web terminal draws its own chrome.",
	Danger: "On draws a bar inside the terminal view.",
}, {
	Key:     "allow-passthrough",
	Scope:   "window",
	Label:   "Escape-sequence passthrough",
	Effect:  EffectLive,
	Default: "on",
	Values:  []string{"on", "off"},
	Help: "Lets a program inside the terminal talk to the browser terminal " +
		"directly. Copying to your clipboard from inside a TUI rides on this.",
	Danger: "Off breaks copy-to-clipboard from inside the terminal.",
}, {
	Key:     "extended-keys",
	Scope:   "server",
	Label:   "Extended keys",
	Effect:  EffectServer,
	Default: "on",
	Values:  []string{"on", "off"},
	Help: "Keeps modified keys like Shift+Enter distinguishable from plain " +
		"ones. One value for every tmux session on this machine.",
	Danger: "Off collapses Shift+Enter into Enter in Pi.",
}, {
	Key:     "extended-keys-format",
	Scope:   "server",
	Label:   "Key encoding",
	Effect:  EffectServer,
	Default: "xterm",
	Values:  []string{"xterm", "csi-u"},
	Help: "How modified keys are encoded for the programs inside. Pi expects " +
		"xterm (modifyOtherKeys); some other TUIs prefer csi-u.",
	Danger: "One value for every tmux session on this machine, PiCode's and yours alike.",
}}

// Dangers for options PiCode does not curate but must not hide: the panel
// shows the whole catalog, and these are the entries that can take the
// surface down with them. Labelled, never filtered.
var dangers = map[string]string{
	"destroy-unattached": "On, closing the browser tab kills the session and whatever runs in it.",
	"exit-empty":         "Governs when the whole tmux server exits — every session on this machine.",
	"exit-unattached":    "On, the tmux server exits when nothing is attached — every session dies.",
	"detach-on-destroy":  "Changes what happens to your attachment when a session is killed.",
	"default-terminal":   "Programs decide colours and keys by this. Most values break both.",
	"set-clipboard":      "Governs how programs reach your clipboard; 'external' is what the browser terminal expects.",
	"default-shell":      "Every new terminal starts this instead of your shell.",
	"default-command":    "Every new terminal runs this instead of a login shell.",
	"lock-after-time":    "Locks the terminal after idle time; there is no unlock UI in the browser.",
	"lock-command":       "Runs on lock; a broken value wedges the pane.",
}

// DangerFor returns the warning for an option, curated or not.
func DangerFor(key string) string {
	if f, ok := Find(key); ok {
		return f.Danger
	}
	return dangers[key]
}

// Flags returns the curated registry (copied; handlers hand it to encoders).
func Flags() []Flag {
	out := make([]Flag, len(flags))
	copy(out, flags)
	return out
}

// Find returns the curated flag with this key.
func Find(key string) (Flag, bool) {
	for _, f := range flags {
		if f.Key == key {
			return f, true
		}
	}
	return Flag{}, false
}

func (f Flag) accepts(value string) bool {
	for _, v := range f.Values {
		if v == value {
			return true
		}
	}
	return false
}

// Defaults is what an owned session gets when nobody has set anything —
// including the options PiCode used to force in code, which live here now so
// a user override wins over them.
func Defaults() map[string]string {
	out := make(map[string]string, len(flags))
	for _, f := range flags {
		out[f.Key] = f.Default
	}
	return out
}

// Resolve layers maps left to right, later winning, over the curated
// defaults. Curated keys are enum-checked (a stored value the flag no longer
// takes falls back rather than reaching tmux); everything else passes through
// untouched — those keys were validated against the live catalog when they
// were stored, and tmux is the final validator at set-option time.
func Resolve(layers ...map[string]string) map[string]string {
	out := Defaults()
	for _, layer := range layers {
		for k, v := range layer {
			if f, ok := Find(k); ok && !f.accepts(v) {
				continue
			}
			out[k] = v
		}
	}
	return out
}

// ValidateCurated reports the first problem a curated flag has with a patch.
// Non-curated keys pass — the caller checks them against the live catalog and
// lets tmux judge the value. A nil value means "clear this field".
func ValidateCurated(patch map[string]*string) error {
	for k, v := range patch {
		f, ok := Find(k)
		if !ok {
			continue
		}
		if v != nil && !f.accepts(*v) {
			return fmt.Errorf("%q is not a value %s takes", *v, k)
		}
	}
	return nil
}

// Apply folds a patch into a stored layer, returning the new layer. Clearing
// a field removes it, which is what makes it inherit again.
func Apply(base map[string]string, patch map[string]*string) map[string]string {
	out := make(map[string]string, len(base))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		if v == nil {
			delete(out, k)
			continue
		}
		out[k] = *v
	}
	return out
}
