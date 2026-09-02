package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cfpperche/picode/internal/provision"
)

// runProvision converges this machine on what PiCode needs (ADR-0020). It is
// the Linux half of the desktop installer, and it is also the one command a
// native Linux user needs instead of `install` plus scripts/setup-cert.sh.
func runProvision(args []string) {
	fs := flag.NewFlagSet("provision", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "report what would change, touch nothing")
	asJSON := fs.Bool("json", false, "emit results as JSON")
	target := fs.String("user", "", "provision for this account (default: the current user)")
	shared := fs.Bool("shared", false, "a member of a shared box (ADR-0051): create the account, its environment and daemon behind the gateway; root, with --user")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("provision: %v", err)
	}
	if *shared && *target == "" {
		log.Fatalf("provision: --shared needs --user <linux user>")
	}

	env, err := provision.Detect(*target)
	if err != nil {
		log.Fatalf("provision: %v", err)
	}
	steps := provision.Steps()
	if *shared {
		steps = provision.MemberSteps()
	}
	results := provision.Run(env, steps, *dryRun)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(provision.NewReport(env, results, *dryRun)); err != nil {
			log.Fatalf("provision: %v", err)
		}
	} else {
		printResults(env, results, *dryRun)
	}

	if !provision.Converged(results) && !*dryRun {
		os.Exit(1)
	}
}

// printResults keeps one line per step: the mark says what happened, the
// detail says why, so the reader never has to guess which half is the reason.
func printResults(env provision.Env, results []provision.Result, dryRun bool) {
	if dryRun {
		fmt.Printf("Provisioning plan for %s (nothing will be changed)\n\n", env.User)
	} else {
		fmt.Printf("Provisioning PiCode for %s\n\n", env.User)
	}

	var planned, unresolved int
	for _, r := range results {
		state := r.Before
		if r.After != nil {
			state = *r.After
		}
		title := r.Title
		if r.Action == provision.ActionPlanned && r.Scope == provision.ScopeRoot {
			title += " (root)"
		}
		fmt.Printf("  %-6s %-44s %s\n", mark(r.Action), title, state.Detail)
		if r.Error != "" {
			fmt.Printf("         %-44s %s\n", "", r.Error)
		}
		switch r.Action {
		case provision.ActionPlanned:
			planned++
		case provision.ActionSkipped, provision.ActionFailed:
			unresolved++
		}
	}

	// A skipped step is not a step that "would change" — it is one this run
	// could not settle, usually because another pass owns it. Counting the two
	// together tells the reader work is pending when there may be none.
	fmt.Println()
	if planned == 0 && unresolved == 0 {
		if dryRun {
			fmt.Println("Nothing to do — this machine is already provisioned.")
		} else {
			fmt.Println("Provisioned.")
		}
		return
	}
	if planned > 0 {
		fmt.Printf("%d step(s) would change. Run without --dry-run to apply.\n", planned)
	}
	if unresolved > 0 {
		fmt.Printf("%d step(s) left unresolved by this run — see the notes above.\n", unresolved)
	}
}

func mark(a provision.Action) string {
	switch a {
	case provision.ActionNone:
		return "ok"
	case provision.ActionFixed:
		return "fixed"
	case provision.ActionPlanned:
		return "todo"
	case provision.ActionSkipped:
		return "skip"
	default:
		return "FAIL"
	}
}
