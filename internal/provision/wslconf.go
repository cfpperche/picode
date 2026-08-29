// Package provision converges a machine on the state PiCode needs: systemd
// available, the user unit installed and enabled, a usable certificate
// (ADR-0020). Every step is check → fix → verify, idempotent and additive:
// provisioning runs against live environments, so a step that finds nothing
// to do must change nothing at all.
package provision

import "strings"

// ConfPath is the per-distro WSL configuration file. It is a var so tests
// can converge a scratch file instead of the real one.
var ConfPath = "/etc/wsl.conf"

// EnsureKey guarantees that section/key carries value, editing lines instead
// of reserializing the file. Content that already satisfies the pair comes
// back byte for byte identical — comments, blank lines, key order and the
// spacing around "=" all survive, because nothing is rewritten.
//
// When the key exists with another value, only the value is swapped and the
// line's own spacing is kept. When the section exists without the key, the
// key lands after the section's last key=value line (or right after the
// header when it has none), so trailing comments stay trailing. When the
// section is absent it is appended, separated by one blank line.
//
// Section and key match case-insensitively, as does the value comparison, so
// an existing "Systemd = True" is left alone rather than duplicated.
func EnsureKey(content, section, key, value string) (string, bool) {
	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	trailing := content != "" && strings.HasSuffix(content, "\n")
	lines := splitLines(content)

	start, end := findSection(lines, section)
	if start < 0 {
		return appendSection(content, eol, section, key, value), true
	}

	insert := start + 1
	for i := start + 1; i < end; i++ {
		k, v, ok := splitKV(lines[i])
		if !ok {
			continue
		}
		if !strings.EqualFold(k, key) {
			insert = i + 1
			continue
		}
		if strings.EqualFold(strings.TrimSpace(v), value) {
			return content, false
		}
		lines[i] = replaceValue(lines[i], value)
		return joinLines(lines, eol, trailing), true
	}

	lines = append(lines[:insert], append([]string{key + "=" + value}, lines[insert:]...)...)
	return joinLines(lines, eol, trailing), true
}

// EnsureSystemd turns systemd on for the distro: [boot] systemd=true.
func EnsureSystemd(content string) (string, bool) {
	return EnsureKey(content, "boot", "systemd", "true")
}

// splitLines drops line endings; a trailing newline does not produce a final
// empty element, so joinLines can restore the original bytes.
func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	body := strings.TrimSuffix(strings.TrimSuffix(content, "\n"), "\r")
	out := strings.Split(body, "\n")
	for i, l := range out {
		out[i] = strings.TrimSuffix(l, "\r")
	}
	return out
}

func joinLines(lines []string, eol string, trailing bool) string {
	out := strings.Join(lines, eol)
	if trailing {
		out += eol
	}
	return out
}

// findSection returns the header index and the index one past the section's
// last line. A missing section reports -1.
func findSection(lines []string, section string) (int, int) {
	start := -1
	for i, l := range lines {
		name, ok := sectionName(l)
		if !ok {
			continue
		}
		if start >= 0 {
			return start, i
		}
		if strings.EqualFold(name, section) {
			start = i
		}
	}
	if start < 0 {
		return -1, -1
	}
	return start, len(lines)
}

func sectionName(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if len(t) < 2 || !strings.HasPrefix(t, "[") || !strings.HasSuffix(t, "]") {
		return "", false
	}
	return strings.TrimSpace(t[1 : len(t)-1]), true
}

// splitKV reports the key and the raw value of a key=value line. Comments and
// anything without "=" are not pairs.
func splitKV(line string) (string, string, bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, ";") {
		return "", "", false
	}
	i := strings.Index(line, "=")
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), line[i+1:], true
}

// replaceValue swaps the value while keeping the line's spacing, so
// "generateResolvConf = false" stays spaced the way its author wrote it.
func replaceValue(line, value string) string {
	i := strings.Index(line, "=")
	rest := line[i+1:]
	pad := rest[:len(rest)-len(strings.TrimLeft(rest, " \t"))]
	return line[:i+1] + pad + value
}

func appendSection(content, eol, section, key, value string) string {
	out := content
	if out != "" {
		if !strings.HasSuffix(out, "\n") {
			out += eol
		}
		if !strings.HasSuffix(out, eol+eol) {
			out += eol
		}
	}
	return out + "[" + section + "]" + eol + key + "=" + value + eol
}
