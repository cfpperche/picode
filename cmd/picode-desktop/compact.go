package main

import (
	"fmt"

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
func runDiskCompact(distroFlag, userFlag, method string, yes, dryRun, force bool) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
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

	fmt.Printf("%s — before: %s on disk · ≈%s held · %s free on C:\n\n",
		a.distro, hostfs.Bytes(facts.AllocatedBytes), hostfs.Bytes(held), hostfs.Bytes(facts.FreeBytes))
	fmt.Printf("Plan: %s\n", why)
	fmt.Println("Stopping the distro ends everything inside it: agents, terminals and tmux sessions do not come back.")

	if dryRun {
		fmt.Println("\nDry run — nothing was stopped.")
		return nil
	}
	if held <= 0 {
		fmt.Println("\nNothing is held: the distro has already given back what it freed. Nothing to do.")
		return nil
	}

	// The interlock. A server that cannot answer is a refusal, not a shrug:
	// this is the one action here that ends other people's work.
	url, urlErr := desktop.ServerURL(a.runner, a.distro, a.user)
	if urlErr != nil && !force {
		return fmt.Errorf("PiCode is not answering, so I cannot check whether anyone is working — start it first, or use --force")
	}
	if urlErr == nil {
		ready, busy, err := desktop.DeployReady(url)
		if err != nil {
			return fmt.Errorf("ask PiCode whether anyone is working: %w", err)
		}
		if !ready {
			names := desktop.DescribeBusy(busy)
			if !force {
				return fmt.Errorf("still working: %s — finish first, or use --force", names)
			}
			fmt.Printf("  warn   still working, proceeding because of --force: %s\n", names)
		}
	}

	if chosen == desktop.CompactOptimizeVHD && !isAdmin() {
		return fmt.Errorf("Optimize-VHD needs an administrator terminal — run this command from one, or upgrade WSL to 2.0+ for the sparse path")
	}

	if !yes {
		fmt.Println("\nRe-run with --yes to stop " + a.distro + " and compact now.")
		return nil
	}

	res, err := desktop.Compact(a.runner, a.distro, a.user, chosen, facts.VHDXPath, facts.AllocatedBytes, func(s string) {
		fmt.Println("  " + s)
	})
	if err != nil {
		return err
	}

	fmt.Printf("\nBefore %s · after %s · ≈%s back on C:.\n",
		hostfs.Bytes(res.BeforeBytes), hostfs.Bytes(res.AfterBytes), hostfs.Bytes(res.Returned()))
	if res.Sparse {
		fmt.Println("The file is sparse now: WSL returns freed blocks on its own from here on.")
	}
	fmt.Println(a.distro + " is starting again; its sessions do not come back.")
	return nil
}
