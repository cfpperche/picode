package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/hostfs"
)

// runDiskCompact gives the distro's held disk space back to Windows: stop the
// distro, convert or compact the file, start it again, report the measured
// difference.
//
// The order of the guards is the order of the promises. The server's readiness
// interlock is asked before anything stops — the same question `picode deploy`
// asks, because the same work ends. --yes is the one confirmation, --force the
// one override of the interlock, and --dry-run stops before the first promise.
func runDiskCompact(distroFlag, userFlag, method string, yes, dryRun, force, asJSON bool) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	out := compactOutcome{Distro: a.distro}

	abort := func(msg string) error {
		out.Error = msg
		if asJSON {
			return emitCompact(out)
		}
		return fmt.Errorf("%s", msg)
	}
	refuse := func(msg string) error {
		out.Refused = msg
		if asJSON {
			return emitCompact(out)
		}
		return fmt.Errorf("%s", msg)
	}
	note := func(msg string) error {
		out.Note = msg
		if asJSON {
			return emitCompact(out)
		}
		fmt.Println("\n" + msg)
		return nil
	}

	facts, err := desktop.DistroDisk(a.runner, a.distro)
	if err != nil {
		return err
	}
	used, err := distroUsed(a.runner, a.distro, a.user)
	if err != nil {
		return err
	}
	held := desktop.Held(facts, used)

	chosen, why := desktop.PlanCompact(facts)
	if method != "" {
		switch m := desktop.CompactMethod(method); m {
		case desktop.CompactSparse, desktop.CompactOptimizeVHD:
			chosen = m
		default:
			return fmt.Errorf("unknown method %q (sparse or optimize-vhd)", method)
		}
	}

	out.Method = string(chosen)
	out.Plan = why
	out.Held = held
	out.Before = facts.AllocatedBytes
	out.Free = facts.FreeBytes

	if !asJSON {
		fmt.Printf("%s — before: %s on disk · ≈%s held · %s free on C:\n\n",
			a.distro, hostfs.Bytes(facts.AllocatedBytes), hostfs.Bytes(held), hostfs.Bytes(facts.FreeBytes))
		fmt.Printf("Plan: %s\n", why)
		fmt.Println("Stopping the distro ends everything inside it: agents, terminals and tmux sessions do not come back.")
	}

	if dryRun {
		return note("Dry run — nothing was stopped.")
	}
	if held <= 0 {
		return note("Nothing is held: the distro has already given back what it freed. Nothing to do.")
	}

	// The interlock. A server that cannot answer is a refusal, not a shrug:
	// this is the one action here that ends other people's work.
	url, urlErr := desktop.ServerURL(a.runner, a.distro, a.user)
	if urlErr != nil && !force {
		return refuse("PiCode is not answering, so I cannot check whether anyone is working — start it first, or use --force")
	}
	if urlErr == nil {
		ready, busy, err := desktop.DeployReady(url)
		if err != nil {
			return abort("ask PiCode whether anyone is working: " + err.Error())
		}
		if !ready {
			names := desktop.DescribeBusy(busy)
			if !force {
				return refuse("still working: " + names + " — finish first, or use --force")
			}
			if !asJSON {
				fmt.Printf("  warn   still working, proceeding because of --force: %s\n", names)
			}
		}
	}

	if chosen == desktop.CompactOptimizeVHD && !isAdmin() {
		return abort("Optimize-VHD needs an administrator terminal — run this command from one, or upgrade WSL to 2.0+ for the sparse path")
	}

	if !yes {
		return note("Re-run with --yes to stop " + a.distro + " and compact now.")
	}

	// From here on the distro is stopped first thing: any failure after
	// this line has already ended every session inside it, and the outcome
	// has to say so rather than read like a harmless refusal.
	out.Stopped = true
	res, err := desktop.Compact(a.runner, a.distro, a.user, chosen, facts.VHDXPath, facts.AllocatedBytes, func(s string) {
		if asJSON {
			// One progress object per line, so a subprocess consumer can
			// stream steps and still find the final outcome as the last line.
			fmt.Println(progressLine(s))
			return
		}
		fmt.Println("  " + s)
	})
	if err != nil {
		return abort(err.Error())
	}

	after := res.AfterBytes
	returned := res.Returned()
	sparse := res.Sparse
	out.After, out.Returned, out.Sparse = &after, &returned, &sparse

	if !asJSON {
		fmt.Printf("\nBefore %s · after %s · ≈%s back on C:.\n",
			hostfs.Bytes(res.BeforeBytes), hostfs.Bytes(res.AfterBytes), hostfs.Bytes(res.Returned()))
		if res.Sparse {
			fmt.Println("The file is sparse now: WSL returns freed blocks on its own from here on.")
		}
		fmt.Println(a.distro + " is starting again; its sessions do not come back.")
	}
	return emitCompact(out)
}

// compactOutcome is the whole story of one disk-compact run, in an order a
// person can also read top to bottom: what was planned, what was refused and
// why, what was measured.
type compactOutcome struct {
	Distro string `json:"distro"`
	Method string `json:"method,omitempty"`
	Plan   string `json:"plan,omitempty"`

	Held   int64 `json:"held"`
	Before int64 `json:"before"`
	Free   int64 `json:"free"`

	// Refused is the interlock speaking: someone is working, or the server
	// that could tell is down. Nothing was stopped.
	Refused string `json:"refused,omitempty"`
	// Note is an outcome that is not an error: dry run, nothing held,
	// confirmation still required.
	Note string `json:"note,omitempty"`
	// Error is a failure after the bookkeeping — the distro was restarted by
	// the flow itself.
	Error string `json:"error,omitempty"`
	// Stopped says the flow reached the stop: the distro was terminated and
	// its sessions ended, whether or not the rest succeeded.
	Stopped bool `json:"stopped,omitempty"`

	After    *int64 `json:"after,omitempty"`
	Returned *int64 `json:"returned,omitempty"`
	Sparse   *bool  `json:"sparse,omitempty"`
}

// emitCompact prints the outcome for the shell. Always exit zero: the fields
// carry the result, and an exit code would make the shell guess which stream
// to trust.
func emitCompact(out compactOutcome) error {
	return json.NewEncoder(os.Stdout).Encode(out)
}

// progressLine renders one step as a JSON object on its own line. %q keeps a
// step name with quotes or newlines from becoming a different object.
func progressLine(step string) string {
	return fmt.Sprintf("{\"progress\":%q}", step)
}
