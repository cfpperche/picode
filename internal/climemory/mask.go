package climemory

import "regexp"

// Memory files hold whatever the agent learned, and an agent that was shown a
// token can have written it down. Reads mask the shapes that are credentials
// beyond reasonable doubt, and nothing else: masking ordinary prose would make
// the pane lie about what the file says.
//
// This is a read-time mask, not a promise the file is clean. The pane says so,
// and the file itself stays whatever the CLI wrote.
var secretShapes = []*regexp.Regexp{
	// OpenAI and compatible keys.
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{16,}`),
	// GitHub tokens, classic and fine-grained.
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
	// Slack.
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
	// AWS access key ids.
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	// Google API keys.
	regexp.MustCompile(`AIza[0-9A-Za-z_-]{30,}`),
	// Anthropic.
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`),
	// JSON web tokens.
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
	// A bearer token in a header the agent copied.
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/-]{20,}=*`),
	// Private keys, whole block.
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
}

const maskText = "[redacted by PiCode]"

func mask(s string) string {
	for _, re := range secretShapes {
		s = re.ReplaceAllString(s, maskText)
	}
	return s
}
