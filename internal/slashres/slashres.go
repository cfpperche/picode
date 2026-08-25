// Package slashres lists pi skills and prompt templates for the composer picker.
package slashres

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Item is one picker row. Secrets never appear.
type Item struct {
	Name string `json:"name"`
	Hint string `json:"hint"`
	Kind string `json:"kind"` // skill | template
}

// List scans global dirs, plus project dirs when trusted.
func List(cwd string, trusted bool) []Item {
	home, _ := os.UserHomeDir()
	var roots []string
	if home != "" {
		roots = append(roots,
			filepath.Join(home, ".pi", "agent", "skills"),
			filepath.Join(home, ".agents", "skills"),
		)
	}
	if trusted && cwd != "" {
		roots = append(roots,
			filepath.Join(cwd, ".pi", "skills"),
			filepath.Join(cwd, ".agents", "skills"),
		)
	}
	seen := map[string]bool{}
	var out []Item
	for _, root := range roots {
		for _, it := range skillsIn(root) {
			key := "skill:" + it.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, it)
		}
	}
	var tproots []string
	if home != "" {
		tproots = append(tproots, filepath.Join(home, ".pi", "agent", "prompts"))
	}
	if trusted && cwd != "" {
		tproots = append(tproots, filepath.Join(cwd, ".pi", "prompts"))
	}
	for _, root := range tproots {
		for _, it := range templatesIn(root) {
			key := "tpl:" + it.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, it)
		}
	}
	return out
}

func skillsIn(root string) []Item {
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil
	}
	var out []Item
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && (d.Name() == "node_modules" || d.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(d.Name(), "SKILL.md") && !(filepath.Dir(path) == root && strings.HasSuffix(strings.ToLower(d.Name()), ".md")) {
			return nil
		}
		fm := readFM(path)
		if !strings.EqualFold(d.Name(), "SKILL.md") && strings.TrimSpace(fm["description"]) == "" {
			return nil
		}
		name := strings.TrimSpace(fm["name"])
		if name == "" {
			if strings.EqualFold(d.Name(), "SKILL.md") {
				name = filepath.Base(filepath.Dir(path))
			} else {
				name = strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			}
		}
		name = sanitizeName(name)
		if name == "" {
			return nil
		}
		hint := strings.TrimSpace(fm["description"])
		if hint == "" {
			hint = "Skill"
		}
		out = append(out, Item{Name: name, Hint: clip(hint, 80), Kind: "skill"})
		return nil
	})
	return out
}

func templatesIn(root string) []Item {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []Item
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		path := filepath.Join(root, e.Name())
		fm := readFM(path)
		name := sanitizeName(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		if name == "" {
			continue
		}
		hint := strings.TrimSpace(fm["description"])
		if hint == "" {
			hint = "Template"
		}
		if ah := strings.TrimSpace(fm["argument-hint"]); ah != "" {
			hint = ah + " — " + hint
		}
		out = append(out, Item{Name: name, Hint: clip(hint, 80), Kind: "template"})
	}
	return out
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func readFM(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 256*1024)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return out
	}
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(k))] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return out
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
