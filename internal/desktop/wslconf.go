package desktop

import (
	"fmt"
	"regexp"
	"strings"
)

// The distro's own settings file, /etc/wsl.conf (ADR-0198). The table is
// Microsoft's documented keys (learn.microsoft.com windows/wsl/wsl-config,
// "wsl.conf"), each with a type; anything the table does not own passes
// through untouched — the file is the person's. Keys repeat across
// sections ([automount] enabled, [interop] enabled), so every match is on
// section *and* key.

// ConfKind is how a wsl.conf value is checked before it is written.
type ConfKind string

const (
	ConfBool     ConfKind = "bool"
	ConfUser     ConfKind = "user"     // a Linux account name
	ConfHostname ConfKind = "hostname" // an RFC 1123 label
	ConfPath     ConfKind = "path"     // an absolute path
	ConfText     ConfKind = "text"     // one line, no control characters
	// ConfReadOnly is shown, never written: [boot] command runs as root at
	// every boot, and a window that could set it would be a root shell.
	ConfReadOnly ConfKind = "readonly"
)

// ConfKey is one owned setting.
type ConfKey struct {
	Section string   `json:"section"`
	Key     string   `json:"key"`
	Kind    ConfKind `json:"kind"`
	Hint    string   `json:"hint"`
	Default string   `json:"default"`
}

// WSLConfKeys is the owned table, in the order the window shows it.
var WSLConfKeys = []ConfKey{
	{"boot", "systemd", ConfBool, "start systemd as PID 1 (services, timers, user units)", "false"},
	{"boot", "command", ConfReadOnly, "command run as root when the distro starts", "none"},
	{"user", "default", ConfUser, "account a new terminal and wsl.exe log in as", "the account created at install"},
	{"interop", "enabled", ConfBool, "run Windows programs from Linux", "true"},
	{"interop", "appendWindowsPath", ConfBool, "add the Windows PATH to Linux's PATH", "true"},
	{"automount", "enabled", ConfBool, "mount Windows drives under the automount root", "true"},
	{"automount", "root", ConfPath, "where Windows drives are mounted", "/mnt/"},
	{"automount", "options", ConfText, "mount options for Windows drives (e.g. metadata,umask=22)", "none"},
	{"automount", "mountFsTab", ConfBool, "mount /etc/fstab entries at start", "true"},
	{"network", "hostname", ConfHostname, "the distro's host name", "the Windows host name"},
	{"network", "generateHosts", ConfBool, "write /etc/hosts from Windows", "true"},
	{"network", "generateResolvConf", ConfBool, "write /etc/resolv.conf (DNS) from Windows", "true"},
	{"gpu", "enabled", ConfBool, "give Linux access to the Windows GPU", "true"},
	{"time", "useWindowsTimezone", ConfBool, "follow the Windows time zone", "true"},
}

// ConfValue is one owned key's current value in the file.
type ConfValue struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

// ConfEdit sets (Value non-empty) or removes (empty) one owned key.
type ConfEdit struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func confKey(section, key string) (ConfKey, bool) {
	for _, k := range WSLConfKeys {
		if strings.EqualFold(k.Section, section) && strings.EqualFold(k.Key, key) {
			return k, true
		}
	}
	return ConfKey{}, false
}

// iniLine splits a key=value line; comments and blank lines are not keys.
func iniLine(line string) (key, value string, ok bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, ";") || strings.HasPrefix(t, "[") {
		return "", "", false
	}
	k, v, found := strings.Cut(t, "=")
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(k), strings.TrimSpace(v), true
}

func sectionOf(line string) (string, bool) {
	t := strings.TrimSpace(strings.TrimPrefix(line, "\uFEFF"))
	// A header may carry a trailing comment: "[boot] # start-up".
	if i := strings.IndexAny(t, "#;"); i > 0 {
		t = strings.TrimSpace(t[:i])
	}
	if len(t) >= 2 && t[0] == '[' && t[len(t)-1] == ']' {
		return strings.TrimSpace(t[1 : len(t)-1]), true
	}
	return "", false
}

// ParseWSLConf returns the owned keys the file sets, in file order.
func ParseWSLConf(raw string) []ConfValue {
	var out []ConfValue
	section := ""
	for _, line := range strings.Split(raw, "\n") {
		if s, ok := sectionOf(line); ok {
			section = s
			continue
		}
		k, v, ok := iniLine(line)
		if !ok {
			continue
		}
		if ck, owned := confKey(section, k); owned {
			v = strings.Trim(v, `"`)
			// WSL reads booleans case-insensitively; the window's select
			// only knows true/false, and a "True" shown as "default" would
			// be removed by the next save of any other key.
			if ck.Kind == ConfBool {
				v = strings.ToLower(v)
			}
			out = append(out, ConfValue{Section: ck.Section, Key: ck.Key, Value: v})
		}
	}
	return out
}

var (
	userRe     = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	hostnameRe = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
	// Mount options are a comma list of words and key=value pairs.
	optionsRe = regexp.MustCompile(`^[A-Za-z0-9=,_./:-]+$`)
	// The automount root: absolute, plain segments, ending in a slash.
	mountRootRe = regexp.MustCompile(`^/([A-Za-z0-9_-][A-Za-z0-9._-]*/)*$`)
)

// ValidateConf checks one edit against the table. Removing a key (empty
// value) is always allowed for writable keys.
func ValidateConf(e ConfEdit) error {
	ck, ok := confKey(e.Section, e.Key)
	if !ok {
		return fmt.Errorf("[%s] %s is not a setting this window writes", e.Section, e.Key)
	}
	if ck.Kind == ConfReadOnly {
		return fmt.Errorf("[%s] %s is shown, never written here", ck.Section, ck.Key)
	}
	v := e.Value
	if v == "" {
		return nil
	}
	if strings.ContainsAny(v, "\r\n\x00") || strings.ContainsRune(v, '#') {
		return fmt.Errorf("[%s] %s: one line, no comments", ck.Section, ck.Key)
	}
	switch ck.Kind {
	case ConfBool:
		if v != "true" && v != "false" {
			return fmt.Errorf("[%s] %s must be true or false", ck.Section, ck.Key)
		}
	case ConfUser:
		if !userRe.MatchString(v) {
			return fmt.Errorf("[%s] %s: %q is not a Linux account name", ck.Section, ck.Key, v)
		}
		// Every terminal and every PiCode call would run as root, and the
		// distro half would look for picode in /root.
		if v == "root" {
			return fmt.Errorf("[%s] %s: logging in as root is not offered here", ck.Section, ck.Key)
		}
	case ConfHostname:
		if !hostnameRe.MatchString(v) {
			return fmt.Errorf("[%s] %s: %q is not a host name", ck.Section, ck.Key, v)
		}
	case ConfPath:
		if !mountRootRe.MatchString(v) || strings.Contains(v, "/../") || strings.Contains(v, "/./") {
			return fmt.Errorf("[%s] %s must be an absolute path of plain folder names ending in /, like /mnt/", ck.Section, ck.Key)
		}
	case ConfText:
		if !optionsRe.MatchString(v) {
			return fmt.Errorf("[%s] %s: use letters, digits and = , _ . / : - only", ck.Section, ck.Key)
		}
	}
	return nil
}

// EditWSLConf applies validated edits: an owned key is replaced where it
// stands (all its copies in that section, keeping the first), removed when
// its value is empty, or appended to its section — created at the end when
// the file has none. Every other line is kept byte for byte.
func EditWSLConf(raw string, edits []ConfEdit) (string, error) {
	want := map[[2]string]ConfEdit{}
	var order [][2]string
	for _, e := range edits {
		if err := ValidateConf(e); err != nil {
			return "", err
		}
		ck, _ := confKey(e.Section, e.Key)
		id := [2]string{ck.Section, ck.Key}
		if _, dup := want[id]; !dup {
			order = append(order, id)
		}
		e.Section, e.Key = ck.Section, ck.Key
		want[id] = e
	}

	// Keep the file's own line endings and BOM: a CRLF file stays CRLF.
	bom := strings.HasPrefix(raw, "\uFEFF")
	body := strings.TrimPrefix(raw, "\uFEFF")
	nl := "\n"
	if strings.Contains(body, "\r\n") {
		nl = "\r\n"
		body = strings.ReplaceAll(body, "\r\n", "\n")
	}
	var lines []string
	if body != "" {
		lines = strings.Split(body, "\n")
	}
	trailingNL := strings.HasSuffix(body, "\n")
	if trailingNL {
		lines = lines[:len(lines)-1]
	}
	written := map[[2]string]bool{}
	removedFrom := map[string]bool{} // sections a removal took a line from
	var out []string
	section := ""
	for _, line := range lines {
		if s, ok := sectionOf(line); ok {
			section = s
			out = append(out, line)
			continue
		}
		if k, _, ok := iniLine(line); ok {
			if ck, owned := confKey(section, k); owned {
				id := [2]string{ck.Section, ck.Key}
				if e, edited := want[id]; edited {
					if e.Value != "" && !written[id] {
						out = append(out, ck.Key+"="+e.Value)
					}
					if e.Value == "" {
						removedFrom[strings.ToLower(ck.Section)] = true
					}
					written[id] = true
					continue
				}
			}
		}
		out = append(out, line)
	}

	// Keys the file did not have: after the last key line of their
	// section, or in a new section at the end.
	for _, id := range order {
		e := want[id]
		if written[id] || e.Value == "" {
			continue
		}
		at := -1
		cur := ""
		for i, line := range out {
			if s, ok := sectionOf(line); ok {
				cur = s
				if strings.EqualFold(cur, id[0]) {
					at = i + 1
				}
				continue
			}
			if strings.EqualFold(cur, id[0]) && strings.TrimSpace(line) != "" {
				at = i + 1
			}
		}
		entry := id[1] + "=" + e.Value
		if at < 0 {
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
				out = append(out, "")
			}
			out = append(out, "["+id[0]+"]", entry)
			continue
		}
		out = append(out[:at], append([]string{entry}, out[at:]...)...)
	}

	out = dropEmptiedSections(out, removedFrom)

	result := strings.Join(out, nl)
	if trailingNL || (raw == "" && result != "") {
		result += nl
	}
	if bom {
		result = "\uFEFF" + result
	}
	return result, nil
}

// dropEmptiedSections removes a section header an edit left with nothing in
// it (a removal took its last line; a section already empty is the person's) — with the blank line before it when it was the
// file's last section — so a set and a remove round-trip to the original. A section no edit touched stays,
// empty or not: it is the person's.
func dropEmptiedSections(lines []string, touched map[string]bool) []string {
	var out []string
	for i := 0; i < len(lines); i++ {
		s, ok := sectionOf(lines[i])
		if ok && touched[strings.ToLower(s)] {
			empty := true
			j := i + 1
			for ; j < len(lines); j++ {
				if _, next := sectionOf(lines[j]); next {
					break
				}
				if strings.TrimSpace(lines[j]) != "" {
					empty = false
					break
				}
			}
			if empty {
				// The blank line that separated it goes too when it was
				// the last section; otherwise it now separates the next.
				if n := len(out); j == len(lines) && n > 0 && strings.TrimSpace(out[n-1]) == "" {
					out = out[:n-1]
				}
				i = j - 1
				continue
			}
		}
		out = append(out, lines[i])
	}
	return out
}
