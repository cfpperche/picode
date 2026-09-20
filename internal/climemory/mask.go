package climemory

import "regexp"

// Memory files hold whatever the agent learned, and an agent that was shown a
// token can have written it down. Reads mask the shapes that are credentials
// beyond reasonable doubt, and nothing else: masking ordinary prose would make
// the pane lie about what the file says.
//
// This is a read-time mask, not a promise the file is clean. The pane says so,
// and the file itself stays whatever the CLI wrote.
//
// Each rule carries its replacement, so a rule that keeps context (the name of
// an assignment, the host of a URL) says so in one place. The adversarial
// review of 2026-09-20 found both halves wrong: several ordinary credential
// shapes were published, and an unanchored `sk-` destroyed prose like
// "risk-assessment-and-mitigation".
type maskRule struct {
	re   *regexp.Regexp
	with string
}

const maskText = "[redacted by PiCode]"

var secretShapes = []maskRule{
	// Provider keys, anchored on a word boundary.
	{regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`), maskText},
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}`), maskText},
	{regexp.MustCompile(`\b[sr]k_(?:live|test)_[A-Za-z0-9]{16,}`), maskText},
	{regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}`), maskText},
	{regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}`), maskText},
	{regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}`), maskText},
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}`), maskText},
	{regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{30,}`), maskText},
	{regexp.MustCompile(`\bGOCSPX-[A-Za-z0-9_-]{20,}`), maskText},
	{regexp.MustCompile(`\bnpm_[A-Za-z0-9]{30,}`), maskText},
	// A Slack webhook is a credential in URL form.
	{regexp.MustCompile(`https://hooks\.slack\.com/services/[A-Za-z0-9/_-]{20,}`), maskText},
	// JSON web tokens.
	{regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`), maskText},
	// A bearer token in a header the agent copied.
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/-]{20,}=*`), "${1}" + maskText},
	// A password in a URL's userinfo: the scheme, the user and the host stay,
	// so the line still says which service it was.
	{regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://[^\s:/@]+):[^\s/@]{4,}@`), "${1}:" + maskText + "@"},
	// An assignment whose name says the value is a credential. The name stays,
	// so the reader still knows what was there. The optional word-character
	// prefix catches `DB_PASSWORD=` and `MY_API_KEY:`.
	{regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*(?:secret_access_key|client_secret|password|passwd|api[_-]?key|access[_-]?token|auth[_-]?token|private[_-]?key)\s*[:=]\s*)\S{8,}`), "${1}" + maskText},
	// Private-key blocks, including PGP (whose marker ends in BLOCK) and a
	// block whose END marker never came — a non-greedy match needs the
	// terminator, so a truncated key used to be published whole.
	{regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY( BLOCK)?-----.*?-----END [A-Z0-9 ]*PRIVATE KEY( BLOCK)?-----`), maskText},
	{regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY( BLOCK)?-----.*`), maskText},
}

func mask(s string) string {
	for _, rule := range secretShapes {
		s = rule.re.ReplaceAllString(s, rule.with)
	}
	return s
}
