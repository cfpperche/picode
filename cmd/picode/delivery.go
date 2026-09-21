package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/mcptool"
)

const deliveryHelp = `Usage: picode delivery <action> [options]
  capabilities
  register --title TEXT --branch NAME --revision FULL_SHA --target NAME --request-id KEY
  update --id ID --expected-version N --title TEXT --branch NAME --revision FULL_SHA --target NAME --request-id KEY
  request-review --id ID --expected-version N --request-id KEY
  withdraw-review --id ID --expected-version N --request-id KEY
  show --id ID
  list [--before SEQUENCE]

Output is JSON. Mutations require a retry key; reuse the exact payload for retries.
Use the returned version for your next change. An update clears the review request.
The repository comes from your PiCode launch folder, not the shell's current directory.
Review requested is not approved. Integration queues and deploy are unavailable.
Requires an existing PiCode agent or terminal identity and daemon access.
Optional MCP interface: picode mcp delivery (the same contract).
`

func runDelivery(args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	env := mcptool.Env(os.Getenv)
	c := &mcptool.Caller{Identity: mcptool.IdentityFrom(env)}
	data := browserhost.DataDir()
	if base, err := mcptool.ResolveURL(env, data); err == nil {
		c.Daemon = mcptool.NewHTTPDaemon(base, func() string { return mcptool.ReadToken(env, data) })
	} else {
		c.Unreachable = err.Error()
	}
	if code := deliveryMain(ctx, args, c, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}

func deliveryMain(ctx context.Context, args []string, c *mcptool.Caller, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(out, deliveryHelp)
		return 0
	}
	action := args[0]
	switch action {
	case "capabilities", "register", "update", "request-review", "withdraw-review", "show", "list", "request-integration", "request-deployment":
	default:
		fmt.Fprintln(errOut, "unknown delivery action")
		return 2
	}
	fs := flag.NewFlagSet("delivery "+action, flag.ContinueOnError)
	fs.SetOutput(errOut)
	id := fs.String("id", "", "delivery ID")
	title := fs.String("title", "", "one-line title")
	branch := fs.String("branch", "", "local branch")
	rev := fs.String("revision", "", "full source revision")
	target := fs.String("target", "", "local target branch")
	key := fs.String("request-id", "", "idempotency key")
	version := fs.Int("expected-version", 0, "last returned version")
	before := fs.Int64("before", 0, "pagination cursor")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "unexpected positional arguments")
		return 2
	}
	p := map[string]any{"action": action}
	for k, v := range map[string]string{"id": *id, "title": *title, "branch": *branch, "revision": *rev, "target": *target, "requestId": *key} {
		if v != "" {
			p[k] = v
		}
	}
	if *version != 0 {
		p["expectedVersion"] = *version
	}
	if *before != 0 {
		p["before"] = *before
	}
	raw, _ := json.Marshal(p)
	result, err := mcptool.CallDelivery(ctx, c, raw)
	if err != nil {
		_ = json.NewEncoder(out).Encode(map[string]any{"error": err.Error()})
		return 1
	}
	fmt.Fprintln(out, string(result))
	return 0
}
