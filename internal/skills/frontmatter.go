package skills

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the part of SKILL.md the Agent Skills spec defines.
type Frontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Metadata      map[string]string `yaml:"metadata"`
	// Loose: the header is not valid YAML and was read line by line.
	Loose bool `yaml:"-"`
}

// maxFrontmatter bounds how much of a SKILL.md is read for its header.
const maxFrontmatter = 64 << 10

// readFrontmatter parses the YAML between the leading "---" lines. A file
// without a header answers an empty Frontmatter and ok=false.
func readFrontmatter(path string) (Frontmatter, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Frontmatter{}, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), maxFrontmatter)
	if !sc.Scan() || strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\ufeff")) != "---" {
		return Frontmatter{}, false
	}
	var buf bytes.Buffer
	closed := false
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	if !closed {
		return Frontmatter{}, false
	}
	var fm Frontmatter
	if err := yaml.Unmarshal(buf.Bytes(), &fm); err != nil {
		// A value the typed decode refuses (metadata that is not a string
		// map) must not hide the name and description.
		var loose map[string]any
		if yaml.Unmarshal(buf.Bytes(), &loose) == nil {
			fm.Name, _ = loose["name"].(string)
			fm.Description, _ = loose["description"].(string)
		} else {
			// Not YAML at all — an unquoted description with ": " in it. The
			// CLIs read such a header line by line (measured: Grok and Muse
			// load it), so the report does too, and says so.
			fm = lineFrontmatter(buf.String())
			fm.Loose = true
		}
	}
	fm.Name = strings.TrimSpace(fm.Name)
	fm.Description = strings.TrimSpace(fm.Description)
	return fm, true
}

// lineFrontmatter reads top-level "key: value" lines.
func lineFrontmatter(s string) Frontmatter {
	var fm Frontmatter
	for _, line := range strings.Split(s, "\n") {
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.TrimSpace(k) {
		case "name":
			fm.Name = v
		case "description":
			fm.Description = v
		case "license":
			fm.License = v
		}
	}
	return fm
}

// problems lists the spec rules a skill breaks (agentskills.io
// specification): the pane shows them; the CLI may still load the skill.
func problems(fm Frontmatter, hasHeader bool, folder string) []string {
	if !hasHeader {
		return []string{"SKILL.md has no YAML frontmatter"}
	}
	var out []string
	if fm.Loose {
		out = append(out, "its header has a formatting problem; the CLIs still read it")
	}
	switch {
	case fm.Name == "":
		out = append(out, "no name")
	case !validName(fm.Name):
		out = append(out, "name must be 1–64 lowercase letters, digits and single hyphens")
	case fm.Name != folder:
		out = append(out, "name does not match its folder ("+folder+")")
	}
	if fm.Description == "" {
		out = append(out, "no description")
	} else if utf8.RuneCountInString(fm.Description) > 1024 {
		out = append(out, "description is longer than 1024 characters")
	}
	return out
}

func validName(s string) bool {
	if s == "" || len(s) > 64 || s[0] == '-' || s[len(s)-1] == '-' || strings.Contains(s, "--") {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}
