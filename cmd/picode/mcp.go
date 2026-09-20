package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/mcptool"
	"github.com/cfpperche/picode/internal/version"
)

// runMCP is `picode mcp <family…>` (ADR-0154): a stdio MCP server that
// gives a guest CLI PiCode's own tools. It never serves HTTP and never
// starts the daemon — it finds the running one the way the pi packages do.
func runMCP(args []string) {
	if code := mcpMain(args, os.Stdin, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}

// mcpMain is runMCP without the exit, for tests.
func mcpMain(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintf(errOut, "usage: picode mcp <family…>\nfamilies: %s\n", strings.Join(mcptool.FamilyNames(), ", "))
		return 2
	}
	var families []mcptool.Family
	seen := map[string]bool{}
	for _, name := range args {
		f, ok := mcptool.FamilyFor(name)
		if !ok {
			fmt.Fprintf(errOut, "picode mcp: unknown family %q — families: %s\n", name, strings.Join(mcptool.FamilyNames(), ", "))
			return 2
		}
		if !seen[f.Name] {
			seen[f.Name] = true
			families = append(families, f)
		}
	}
	name := "picode"
	if len(families) == 1 {
		name = "picode-" + families[0].Name
	}
	env := mcptool.Env(os.Getenv)
	dataDir := browserhost.DataDir()
	caller := &mcptool.Caller{Identity: mcptool.IdentityFrom(env), Captures: mcptoolCaptureDir(dataDir)}
	// A missing daemon is not a startup failure: the CLI would show a dead
	// server and the model nothing. The tool answers with the reason instead.
	if base, err := mcptool.ResolveURL(env, dataDir); err != nil {
		caller.Unreachable = err.Error()
	} else {
		caller.Daemon = mcptool.NewHTTPDaemon(base, func() string { return mcptool.ReadToken(env, dataDir) })
	}
	server := mcptool.NewServer(name, version.Build(), families, caller)
	if err := server.Serve(context.Background(), in, out); err != nil {
		fmt.Fprintln(errOut, "picode mcp:", err)
		return 1
	}
	return 0
}

// mcptoolCaptureDir is where screenshots land on disk beside the image
// block, for clients that read files but not blocks (ADR-0154).
func mcptoolCaptureDir(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	return dataDir + "/var/captures"
}
