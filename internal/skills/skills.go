// Package skills reads the Agent Skills (agentskills.io) each agent CLI
// loads: one declaration per CLI of the folders it reads in its own
// precedence order, and a report of what is there — shadowing, provenance
// from the installers' locks, spec problems, an estimated context cost.
// ADR-0196; docs/architecture/skills.md.
package skills

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Scope is the caller's vocabulary, the same as packages (ADR-0176).
type Scope string

const (
	Machine   Scope = "machine"
	Workspace Scope = "workspace"
	Agent     Scope = "agent" // one agent's own skills, from PiCode's cache (slice 4)
)

// Status is what the CLI does with one skill folder.
const (
	StatusLoaded     = "loaded"      // the CLI loads it
	StatusShadowed   = "shadowed"    // a higher-precedence folder has the same name
	StatusInvalid    = "invalid"     // no frontmatter or no description: CLIs skip it
	StatusNeedsTrust = "needs-trust" // the workspace is not trusted in this CLI
	StatusIfTrusted  = "if-trusted"  // loads once the workspace is trusted; PiCode cannot read the CLI's trust
	StatusMissing    = "missing"     // an agent skill whose cached folder is gone: the launch leaves it out
	StatusDisabled   = "disabled"    // the CLI's own per-skill switch is off (slice 3)
)

// Row is one skill folder a CLI reads.
type Row struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Scope       Scope       `json:"scope"`
	Root        string      `json:"root"` // the folder as the pane names it
	Dir         string      `json:"dir"`
	Status      string      `json:"status"`
	ShadowedBy  string      `json:"shadowedBy,omitempty"`
	Linked      bool        `json:"linked,omitempty"` // the skill folder is a link
	Digest      string      `json:"digest,omitempty"`
	Tokens      int         `json:"tokens"` // estimated startup cost: name + description
	Problems    []string    `json:"problems,omitempty"`
	Provenance  *Provenance `json:"provenance,omitempty"`
	// AlsoLoadedBy lists the other CLIs that read this same folder.
	AlsoLoadedBy []string `json:"alsoLoadedBy,omitempty"`
	// Enabled is the CLI's own per-skill switch; nil when the row has none.
	Enabled *bool `json:"enabled,omitempty"`
}

// Trust is the CLI's gate on workspace skills.
type Trust struct {
	Needed  bool   `json:"needed"`
	Command string `json:"command,omitempty"`
	Note    string `json:"note,omitempty"`
	// Trusted is known only where PiCode can read the CLI's own record.
	Trusted *bool `json:"trusted,omitempty"`
}

// AgentInfo is the agent a report was read for, when the CLI has an agent
// scope: its name as the pane badges it, and whether it runs isolated (it
// then loads none of the folder skills, only its own packages').
type AgentInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Isolated bool   `json:"isolated"`
	// Skills are the agent's own, as the store keeps them.
	Skills []AgentSkill `json:"-"`
}

// AgentSkill is one of an agent's own skills: a folder in PiCode's cache.
type AgentSkill struct {
	Name, Digest, Source, Dir string
}

// Report is one CLI's skills.
type Report struct {
	CLI           string     `json:"cli"`
	Roots         []string   `json:"roots"` // the folders read, in precedence order
	Rows          []Row      `json:"rows"`
	Trust         Trust      `json:"trust"`
	Toggle        string     `json:"toggle,omitempty"`
	SwitchNote    string     `json:"switchNote,omitempty"` // a machine-only switch, said once
	Launch        string     `json:"launch,omitempty"`
	WorkspacePath string     `json:"workspacePath,omitempty"`
	Agent         *AgentInfo `json:"agent,omitempty"`
	Notes         []string   `json:"notes,omitempty"`
	ReadAt        string     `json:"readAt"`
}

// Query is one read. Home defaults to the user's home directory.
type Query struct {
	CLI       string
	Workspace string
	Home      string
	// Trusted answers the CLI's own trust record where PiCode can read it
	// (Pi's trust.json); nil elsewhere.
	Trusted func(cli, workspace string) *bool
	// Agent is set when the pane names an agent of this CLI; a CLI without an
	// agent scope ignores it.
	Agent *AgentInfo
}

// ErrUnknownCLI is a CLI with no declaration.
var ErrUnknownCLI = errors.New("no skills declaration for this CLI")

// resolved is one root at an absolute path.
type resolved struct {
	ScopedRoot
	abs     string
	display string
}

func resolveRoots(spec Spec, homeDir, workspace string) []resolved {
	var out []resolved
	for _, r := range spec.Roots {
		switch r.Scope {
		case Machine:
			abs := r.Path
			if strings.HasPrefix(abs, "~/") {
				if homeDir == "" {
					continue
				}
				abs = filepath.Join(homeDir, strings.TrimPrefix(abs, "~/"))
			}
			out = append(out, resolved{ScopedRoot: r, abs: abs, display: r.Path})
		case Workspace:
			if workspace == "" {
				continue
			}
			dirs := []string{workspace}
			if r.WalkUp {
				dirs = walkUp(workspace)
			}
			for _, d := range dirs {
				disp := r.Path
				if d != workspace {
					if rel, err := filepath.Rel(workspace, d); err == nil {
						disp = filepath.ToSlash(filepath.Join(rel, r.Path))
					}
				}
				out = append(out, resolved{ScopedRoot: r, abs: filepath.Join(d, r.Path), display: disp})
			}
		}
	}
	return out
}

// walkUp is the workspace and its parents up to the repository root,
// nearest first; outside a repository, the workspace alone.
func walkUp(workspace string) []string {
	var dirs []string
	d := filepath.Clean(workspace)
	for {
		dirs = append(dirs, d)
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return dirs
		}
		parent := filepath.Dir(d)
		if parent == d {
			return []string{filepath.Clean(workspace)}
		}
		d = parent
	}
}

// found is one SKILL.md folder under a root.
type found struct {
	dir, rel string
	linked   bool
}

// skillDirs lists the skill folders under root: <root>/<name>/SKILL.md, or
// at any depth for a recursive root (a folder with SKILL.md is a skill and
// is not searched further).
func skillDirs(root string, recursive bool) []found {
	var out []found
	var walk func(dir, rel string, depth int)
	walk = func(dir, rel string, depth int) {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range ents {
			name := e.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" {
				continue
			}
			p := filepath.Join(dir, name)
			linked := e.Type()&os.ModeSymlink != 0
			st, err := os.Stat(p) // follows a linked skill folder
			if err != nil || !st.IsDir() {
				continue
			}
			r := name
			if rel != "" {
				r = rel + "/" + name
			}
			if _, err := os.Stat(filepath.Join(p, "SKILL.md")); err == nil {
				out = append(out, found{dir: p, rel: r, linked: linked})
				continue
			}
			if recursive && depth < 4 {
				walk(p, r, depth+1)
			}
		}
	}
	walk(root, "", 0)
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}

// Read reports what one CLI loads.
func Read(q Query) (Report, error) {
	spec, ok := For(q.CLI)
	if !ok {
		return Report{}, ErrUnknownCLI
	}
	homeDir := q.Home
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}
	rep := Report{
		CLI:           spec.CLI,
		Toggle:        spec.Toggle,
		Launch:        spec.Launch,
		WorkspacePath: q.Workspace,
		ReadAt:        time.Now().UTC().Format(time.RFC3339),
		Rows:          []Row{},
		Trust:         Trust{Needed: spec.ProjectTrust, Command: spec.TrustCommand, Note: spec.TrustNote},
	}
	if spec.AgentScope && q.Agent != nil {
		rep.Agent = q.Agent
	}
	if spec.ProjectTrust && q.Workspace != "" && q.Trusted != nil {
		rep.Trust.Trusted = q.Trusted(spec.CLI, q.Workspace)
	}
	lk := loadLocks(homeDir, q.Workspace)
	rep.Notes = append(rep.Notes, lk.notes...)

	// Which other CLIs read each folder, for "also loaded by".
	readers := map[string][]string{}
	for _, other := range specs {
		for _, r := range resolveRoots(other, homeDir, q.Workspace) {
			readers[r.abs] = appendOnce(readers[r.abs], other.CLI)
		}
	}

	winner := map[string]string{} // name → display root of the loaded copy
	var agent *AgentInfo
	if spec.AgentScope && q.Agent != nil {
		agent = q.Agent
	}
	if agent != nil && spec.AgentOrder == AgentWins {
		rep.Rows = append(rep.Rows, agentRows(agent, winner, false)...)
	}
	roots := resolveRoots(spec, homeDir, q.Workspace)
	for _, r := range roots {
		rep.Roots = append(rep.Roots, r.display)
		for _, f := range skillDirs(r.abs, r.Recursive) {
			fm, hasHeader := readFrontmatter(filepath.Join(f.dir, "SKILL.md"))
			folder := filepath.Base(f.dir)
			row := Row{
				Name:        fm.Name,
				Description: fm.Description,
				Scope:       r.Scope,
				Root:        r.display,
				Dir:         f.dir,
				Linked:      f.linked,
				Problems:    problems(fm, hasHeader, folder),
			}
			if row.Name == "" {
				row.Name = folder
			}
			row.Tokens = (len(row.Name) + len(row.Description) + 3) / 4
			if d, err := Digest(f.dir); err == nil {
				row.Digest = d
			}
			row.Provenance = provenanceFor(lk, row, homeDir, f.rel)
			for _, cli := range readers[r.abs] {
				if cli != spec.CLI {
					row.AlsoLoadedBy = append(row.AlsoLoadedBy, cli)
				}
			}
			switch {
			case !hasHeader || fm.Description == "":
				row.Status = StatusInvalid
			case r.Scope == Workspace && spec.ProjectTrust && rep.Trust.Trusted != nil && !*rep.Trust.Trusted:
				row.Status = StatusNeedsTrust
			case winner[row.Name] != "":
				row.Status = StatusShadowed
				row.ShadowedBy = winner[row.Name]
			case r.Scope == Workspace && spec.ProjectTrust && rep.Trust.Trusted == nil:
				row.Status = StatusIfTrusted
				winner[row.Name] = r.display
			default:
				row.Status = StatusLoaded
				winner[row.Name] = r.display
			}
			rep.Rows = append(rep.Rows, row)
		}
	}
	switch {
	case agent == nil || spec.AgentOrder == AgentWins:
	case spec.AgentOrder == AgentFoldersWin && !agent.Isolated:
		rep.Rows = append(rep.Rows, agentRows(agent, winner, true)...)
	default: // namespaced, or isolated: no folder skill competes for the name
		rep.Rows = append(rep.Rows, agentRows(agent, map[string]string{}, true)...)
	}
	applySwitches(spec, homeDir, q.Workspace, rep.Rows)
	rep.SwitchNote = spec.SwitchNote
	return rep, nil
}

// AgentRoot is how the pane names the agent's own list.
const AgentRoot = "This agent only"

// agentRows are the agent's own skills. yield: a folder copy that already
// took the name wins (Pi); otherwise the agent's copy takes it (Omp).
func agentRows(a *AgentInfo, winner map[string]string, yield bool) []Row {
	var out []Row
	for _, sk := range a.Skills {
		fm, hasHeader := readFrontmatter(filepath.Join(sk.Dir, "SKILL.md"))
		row := Row{Name: sk.Name, Description: fm.Description, Scope: Agent, Root: AgentRoot, Dir: sk.Dir, Digest: sk.Digest}
		if sk.Source != "" {
			row.Provenance = &Provenance{Installer: "picode", Source: sk.Source, Hash: sk.Digest}
		}
		row.Tokens = (len(row.Name) + len(row.Description) + 3) / 4
		switch {
		case !hasHeader:
			row.Status = StatusMissing // the pane words it; the launch leaves it out
		case yield && winner[row.Name] != "":
			row.Status = StatusShadowed
			row.ShadowedBy = winner[row.Name]
		default:
			row.Status = StatusLoaded
			winner[row.Name] = AgentRoot
		}
		out = append(out, row)
	}
	return out
}

func provenanceFor(lk locks, row Row, homeDir, rel string) *Provenance {
	if row.Scope == Workspace {
		if p, ok := lk.project[row.Name]; ok {
			p.Modified = p.Hash != "" && row.Digest != "" && p.Hash != row.Digest
			return &p
		}
		return nil
	}
	if homeDir != "" && strings.HasPrefix(row.Dir, filepath.Join(homeDir, ".hermes", "skills")+string(filepath.Separator)) {
		if p, ok := lk.hermes[rel]; ok {
			return &p
		}
		if p, ok := lk.hermes[row.Name]; ok {
			return &p
		}
		return nil
	}
	if p, ok := lk.global[row.Name]; ok {
		return &p
	}
	return nil
}

func appendOnce(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}
