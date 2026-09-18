package clilaunch

// Injection describes an adapter branch, not a claim that the installed CLI
// implements every lifecycle event. Args precede user-supplied arguments.
type Injection struct {
	When string   `json:"when"`
	Args []string `json:"args"`
}

type IntegrationPlan struct {
	Summary     string            `json:"summary"`
	Branches    []Injection       `json:"branches"`
	Files       []string          `json:"files"`
	Environment map[string]string `json:"environment"`
}

type Plan struct {
	Snapshot
	Origins   map[string]string `json:"origins"`
	Injection IntegrationPlan   `json:"injection"`
	// ToolInjection is what ADR-0154's launch scope adds: the MCP servers
	// for Config.Tools, or the reason this CLI gets none at launch.
	ToolInjection *IntegrationPlan `json:"toolInjection,omitempty"`
	InheritedPath []string         `json:"inheritedPath"`
	ManagedEnv    []string         `json:"managedEnv"`
	Problem       string           `json:"problem,omitempty"`
}

type Diagnostic struct {
	Version       string `json:"version,omitempty"`
	Error         string `json:"error,omitempty"`
	CheckedAt     string `json:"checkedAt"`
	Prerequisites bool   `json:"prerequisites"`
	Executable    string `json:"executable,omitempty"`
	Identity      string `json:"identity,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	Stale         bool   `json:"stale"`

	// Lifecycle facts (ADR-0087). UpdateSource is "npm" or "vendor".
	// InstallMethod is clilifecycle.Method: npm, native, vendor, git, unknown.
	Latest          string `json:"latest,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable"`
	UpdateSource    string `json:"updateSource,omitempty"`
	UpdateCheckedAt string `json:"updateCheckedAt,omitempty"`
	UpdateError     string `json:"updateError,omitempty"`
	InstallMethod   string `json:"installMethod,omitempty"`
}

type Attempt struct {
	At    string `json:"at"`
	Error string `json:"error,omitempty"`
}

// CopyOverrides pins a profile's complete configuration, not its relationship
// to another record. An empty executable still means automatic detection.
func CopyOverrides(c Config, base Config) Overrides {
	c = Resolve(c, Overrides{})
	v := Overrides{Executable: &c.Executable, Args: &c.Args, Path: &c.Path, Integration: &c.Integration, Env: map[string]*string{}}
	for k := range base.Env {
		v.Env[k] = nil
	}
	for k, value := range c.Env {
		v.Env[k] = &value
	}
	return v
}
