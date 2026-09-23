package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/mcptool"
)

const missionHelp = `Usage: picode mission <action> --id ID [options]
  show | context
  acknowledge --generation N --expected-version N --request-id KEY
  report --note TEXT [--next-action TEXT] --generation N --expected-version N --request-id KEY
  block --note TEXT --generation N --expected-version N --request-id KEY
  evidence --criterion ID --kind note|file|delivery --value TEXT --outcome pass|fail --generation N --expected-version N --request-id KEY
  request-review [--note TEXT] --generation N --expected-version N --request-id KEY

Only your current PiCode assignment may be updated. Read show for its current
version and generation. Reuse the exact request ID and payload after a timeout.
The owner controls assignment, transfer, scope, pause, cancellation and acceptance.
Evidence is agent-reported; a review request is not approval, merge or deploy.
Optional MCP: picode mcp mission.
`

func runMission(args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	env := mcptool.Env(os.Getenv)
	c := &mcptool.Caller{Identity: mcptool.IdentityFrom(env)}
	data := browserhost.DataDir()
	if base, e := mcptool.ResolveURL(env, data); e == nil {
		c.Daemon = mcptool.NewHTTPDaemon(base, func() string { return mcptool.ReadToken(env, data) })
	} else {
		c.Unreachable = e.Error()
	}
	if code := missionMain(ctx, args, c, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}

func missionMain(ctx context.Context, args []string, c *mcptool.Caller, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(out, missionHelp)
		return 0
	}
	if !slices.Contains(mcptool.MissionActions, args[0]) {
		fmt.Fprintln(errOut, "unknown mission action")
		return 2
	}
	f := flag.NewFlagSet("mission", flag.ContinueOnError)
	f.SetOutput(errOut)
	id := f.String("id", "", "mission ID")
	key := f.String("request-id", "", "retry key")
	version := f.Int("expected-version", 0, "current version")
	generation := f.Int("generation", 0, "assignment generation")
	note := f.String("note", "", "checkpoint or blocker")
	next := f.String("next-action", "", "next action")
	criterion := f.String("criterion", "", "criterion ID")
	kind := f.String("kind", "note", "evidence kind")
	value := f.String("value", "", "evidence content or reference")
	outcome := f.String("outcome", "pass", "pass or fail")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 {
		return 2
	}
	p := map[string]any{"action": args[0], "id": *id}
	if *key != "" {
		p["requestId"] = *key
	}
	if *version != 0 {
		p["expectedVersion"] = *version
	}
	if *generation != 0 {
		p["generation"] = *generation
	}
	if *note != "" {
		p["note"] = *note
	}
	if *next != "" {
		p["nextAction"] = *next
	}
	if args[0] == "evidence" {
		p["evidence"] = map[string]any{"criterionId": *criterion, "kind": *kind, "value": *value, "outcome": *outcome}
	}
	raw, _ := json.Marshal(p)
	res, e := mcptool.CallMission(ctx, c, raw)
	if e != nil {
		_ = json.NewEncoder(out).Encode(map[string]any{"error": e.Error()})
		return 1
	}
	fmt.Fprintln(out, string(res))
	return 0
}
