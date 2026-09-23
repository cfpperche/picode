package skills

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding is one thing the advisory scan noticed. It is never a verdict:
// PiCode shows it beside the files and the person decides (ADR-0196). The
// record ClawHavoc and ToxicSkills left (docs/benchmarks/2026-09-23-skills-
// marketplace.md §6) is what the rules look for.
type Finding struct {
	Severity string `json:"severity"` // critical | warning
	Rule     string `json:"rule"`
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Text     string `json:"text"`
}

type scanRule struct {
	id, severity, text string
	re                 *regexp.Regexp
}

var scanRules = []scanRule{
	{"pipe-to-shell", "critical", "downloads a script and runs it", regexp.MustCompile(`(?i)(curl|wget|iwr|invoke-webrequest)[^|\n]{0,200}\|\s*(sudo\s+)?(ba|z|da)?sh\b|(?i)(iex|invoke-expression)\s*\(?\s*\(?(new-object|iwr|invoke-webrequest)`)},
	{"credentials", "critical", "reads credentials or keys", regexp.MustCompile(`(?i)(~|\$home|\$\{home\}|/home/[^/\s]+)/\.(ssh|aws|gnupg|docker/config\.json|kube/config|netrc)|id_(rsa|ed25519)\b|\.env\b.*(cat|read|curl|upload)|security find-(generic|internet)-password|wallet\.dat|browser.*(cookies|login data)`)},
	{"exfiltration", "critical", "sends data to a paste or tunnel service", regexp.MustCompile(`(?i)(pastebin\.com|transfer\.sh|webhook\.site|requestbin|ngrok\.io|discord(app)?\.com/api/webhooks|api\.telegram\.org/bot)`)},
	{"destructive", "critical", "deletes a home or root folder", regexp.MustCompile(`rm\s+-(rf|fr)\s+(/|~|\$HOME)(\s|$)`)},
	{"encoded-blob", "warning", "carries a long encoded blob", regexp.MustCompile(`[A-Za-z0-9+/=]{400,}`)},
	{"instructions-override", "warning", "tells the agent to ignore its instructions", regexp.MustCompile(`(?i)ignore (all |any )?(previous|prior|above) instructions|do not (tell|inform) the user`)},
}

// hidden characters that change how text reads: bidi overrides and
// zero-width marks (unicode smuggling).
var hiddenChars = regexp.MustCompile("[\u202A-\u202E\u2066-\u2069\u200B-\u200D\u2060\uFEFF]")

const maxScanBytes = 1 << 20

// Scan reads a skill folder's text files and flags binaries.
func Scan(dir string) []Finding {
	var out []Finding
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(p)
		if err != nil || len(data) == 0 {
			return nil
		}
		if isExecutable(data) {
			out = append(out, Finding{Severity: "critical", Rule: "binary", File: rel, Text: "ships a compiled program"})
			return nil
		}
		if bytes.IndexByte(data[:min(len(data), 8000)], 0) >= 0 {
			return nil // other binary data (images): nothing to read
		}
		if len(data) > maxScanBytes {
			data = data[:maxScanBytes]
		}
		seen := map[string]bool{}
		sc := bufio.NewScanner(bytes.NewReader(data))
		sc.Buffer(make([]byte, 0, 64<<10), maxScanBytes)
		line := 0
		for sc.Scan() {
			line++
			text := sc.Text()
			if line == 1 {
				text = strings.TrimPrefix(text, "\ufeff")
			}
			if !seen["hidden-text"] && hiddenChars.MatchString(text) {
				seen["hidden-text"] = true
				out = append(out, Finding{Severity: "critical", Rule: "hidden-text", File: rel, Line: line, Text: "contains invisible characters that change how text reads"})
			}
			for _, r := range scanRules {
				if seen[r.id] || !r.re.MatchString(text) {
					continue
				}
				seen[r.id] = true
				out = append(out, Finding{Severity: r.severity, Rule: r.id, File: rel, Line: line, Text: r.text})
			}
		}
		return nil
	})
	return out
}

func isExecutable(b []byte) bool {
	return bytes.HasPrefix(b, []byte("\x7fELF")) || bytes.HasPrefix(b, []byte("MZ")) ||
		bytes.HasPrefix(b, []byte{0xcf, 0xfa, 0xed, 0xfe}) || bytes.HasPrefix(b, []byte{0xfe, 0xed, 0xfa, 0xcf}) ||
		bytes.HasPrefix(b, []byte{0xca, 0xfe, 0xba, 0xbe})
}

func hasCritical(fs []Finding) bool {
	for _, f := range fs {
		if f.Severity == "critical" {
			return true
		}
	}
	return false
}
