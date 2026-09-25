package version

import (
	"strconv"
	"strings"
)

// The client handshake (ADR-0216). A process outside the daemon that calls
// its API — the picode MCP servers, the browser host, the communication
// CLI, CLI subcommands — can outlive a deploy: the binary on disk is new,
// the running process is not. Each sends ClientHeader; the daemon answers
// 426 when the client's protocol is older than MinClientProtocol, with a
// message the client shows instead of failing on a changed route.
//
// APIProtocol is bumped, and MinClientProtocol raised with it, only when an
// /api change makes an older client *wrong* (a route removed, a field that
// changed meaning) — never for additions, which older clients ignore.
const APIProtocol = 1

// MinClientProtocol is the oldest client protocol the daemon still serves.
// A var so tests can raise it.
var MinClientProtocol = 1

// ClientHeader names the out-of-process client and its build:
// "picode-mcp/0.7.0+abc1234; protocol=1".
const ClientHeader = "X-PiCode-Client"

// ClientValue is what a client of this build sends.
func ClientValue(kind string) string {
	return kind + "/" + Build() + "; protocol=" + strconv.Itoa(APIProtocol)
}

// Client is a parsed ClientHeader.
type Client struct {
	Kind     string
	Build    string
	Protocol int
}

// ParseClient reads a ClientHeader value. ok is false when the header is
// absent or malformed; such a request is served as before (browsers, curl
// and third-party scripts never send it).
func ParseClient(v string) (Client, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return Client{}, false
	}
	head, params, _ := strings.Cut(v, ";")
	kind, build, ok := strings.Cut(strings.TrimSpace(head), "/")
	if !ok || kind == "" {
		return Client{}, false
	}
	c := Client{Kind: kind, Build: build}
	for _, p := range strings.Split(params, ";") {
		k, val, _ := strings.Cut(strings.TrimSpace(p), "=")
		if strings.EqualFold(k, "protocol") {
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return Client{}, false
			}
			c.Protocol = n
		}
	}
	return c, true
}
