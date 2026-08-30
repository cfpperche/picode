// Package termopts is the registry of tmux options PiCode lets a user change,
// and the rule for layering them (ADR-0024).
//
// The list is deliberately short. tmux offers hundreds of options; almost all
// of them are either irrelevant to a browser terminal or a way to break one.
// A flag earns a place here when a real parity gap between two TUIs has been
// found and no single default can serve both — which is exactly how `mouse`
// got here.
package termopts

import "fmt"

// GlobalScope is the store key holding the defaults every terminal inherits.
// Terminal overrides are stored under the terminal's own id, so the two share
// one table without a scope column that could disagree with itself.
const GlobalScope = "global"

// When a change to a flag takes hold. A panel that hides this lies: the user
// flips a switch, nothing happens, and the setting looks broken.
const (
	// EffectLive: the running session picks it up with no reattach. Measured
	// for `mouse` on 2026-08-30 — see ADR-0024's open questions.
	EffectLive = "live"
	// EffectNewPanes: existing panes keep what they have.
	EffectNewPanes = "new-panes"
)

// Flag is one tmux option offered to the user.
type Flag struct {
	Key     string   `json:"key"`     // the tmux option name, passed to set-option
	Label   string   `json:"label"`   // what the panel calls it
	Help    string   `json:"help"`    // the trade being made, in the user's terms
	Effect  string   `json:"effect"`  // when a change takes hold
	Default string   `json:"default"` // PiCode's built-in default
	Values  []string `json:"values"`  // every accepted value, in display order
}

var flags = []Flag{{
	Key:     "mouse",
	Label:   "Mouse belongs to the terminal",
	Effect:  EffectLive,
	Default: "on",
	Values:  []string{"on", "off"},
	Help: "On, the wheel scrolls this terminal's history — the only thing " +
		"that scrolls a program that ignores the mouse. Off, dragging selects " +
		"text the way it does anywhere else. Shift and drag always selects.",
}}

// Flags returns the registry. The copy is not paranoia: the slice is handed
// straight to a JSON encoder in a handler, and a caller that sorted or
// truncated it in place would change what every later request sees.
func Flags() []Flag {
	out := make([]Flag, len(flags))
	copy(out, flags)
	return out
}

// Find returns the flag with this key.
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

// Defaults is what a terminal gets when nobody has set anything.
func Defaults() map[string]string {
	out := make(map[string]string, len(flags))
	for _, f := range flags {
		out[f.Key] = f.Default
	}
	return out
}

// Clean drops anything the registry does not recognise. Stored rows outlive
// the code that wrote them: a flag retired in a later version, or one written
// by a newer binary the user then rolled back, would otherwise be handed to
// `tmux set-option` as an unknown option.
func Clean(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if f, ok := Find(k); ok && f.accepts(v) {
			out[k] = v
		}
	}
	return out
}

// Resolve layers maps left to right, later winning, over the built-in
// defaults. Every layer is cleaned on the way in, so the result only ever
// holds keys and values the registry knows.
func Resolve(layers ...map[string]string) map[string]string {
	out := Defaults()
	for _, layer := range layers {
		for k, v := range Clean(layer) {
			out[k] = v
		}
	}
	return out
}

// Validate reports the first problem in a patch. A nil value means "clear this
// field", which is always legal; anything else must be a value the flag
// accepts. Unknown keys are refused rather than dropped: a typo that silently
// does nothing is the worst possible answer to someone changing a setting.
func Validate(patch map[string]*string) error {
	for k, v := range patch {
		f, ok := Find(k)
		if !ok {
			return fmt.Errorf("%q is not a terminal setting", k)
		}
		if v != nil && !f.accepts(*v) {
			return fmt.Errorf("%q is not a value %s takes", *v, k)
		}
	}
	return nil
}

// Apply folds a patch into a stored layer, returning the new layer. Clearing a
// field removes it, which is what makes the field inherit again — storing the
// inherited value instead would pin it and quietly stop following the global.
func Apply(base map[string]string, patch map[string]*string) map[string]string {
	out := Clean(base)
	for k, v := range patch {
		if v == nil {
			delete(out, k)
			continue
		}
		out[k] = *v
	}
	return Clean(out)
}
