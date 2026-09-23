package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cfpperche/picode/internal/hostfs"
)

// runDisk reports what occupies this machine and what PiCode could give back.
//
// It prints and nothing else. Deleting a cache is a state change with a
// review step (docs/plans/wsl-control.md), and a command a person meets by
// surprise is the wrong place for it: `picode disk` answers "why is my disk
// full" without ever being the reason it got fuller.
func runDisk(args []string) {
	fs := flag.NewFlagSet("disk", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "emit the measurement as JSON")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("disk: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("disk: %v", err)
	}
	rep, err := hostfs.MeasureWith(hostfs.Exec{}, home, locatedConsumers(home))
	if err != nil {
		log.Fatalf("disk: %v", err)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			log.Fatalf("disk: %v", err)
		}
		return
	}
	printDisk(rep)
}

// printDisk is the human half. Every number here is either measured or named
// as unmeasured — a category that quietly swallowed the remainder would make
// the reader trust a total that does not add up.
func printDisk(rep hostfs.Report) {
	fmt.Printf("Filesystem %s\n", rep.FS.Mount)
	fmt.Printf("  %s total · %s used · %s free\n\n",
		hostfs.Bytes(rep.FS.SizeBytes), hostfs.Bytes(rep.FS.UsedBytes), hostfs.Bytes(rep.FS.AvailBytes))

	giveback := hostfs.Reclaimable(rep.Consumers, hostfs.KindSafe, hostfs.KindRedownload)
	fmt.Printf("Give back (measured %s)\n", rep.Measured.Format("15:04"))
	if giveback == 0 {
		fmt.Println("  Nothing to give back — none of the caches PiCode knows about are here.")
	}
	for _, c := range rep.Consumers {
		if c.Kind == hostfs.KindData {
			continue
		}
		fmt.Printf("  %-9s %-11s %-26s %s\n", hostfs.Bytes(c.Bytes), c.Kind, c.Title, c.How)
	}
	if giveback > 0 {
		fmt.Printf("  %s in total · %s of it costs only the time to build again\n",
			hostfs.Bytes(giveback), hostfs.Bytes(hostfs.Reclaimable(rep.Consumers, hostfs.KindSafe)))
	}

	var data []hostfs.Measured
	for _, c := range rep.Consumers {
		if c.Kind == hostfs.KindData {
			data = append(data, c)
		}
	}
	if len(data) > 0 {
		fmt.Println("\nYours, not a cache")
		for _, c := range data {
			fmt.Printf("  %-9s %-26s %s\n", hostfs.Bytes(c.Bytes), c.Title, c.Note)
		}
	}

	fmt.Printf("\nHome (%s)\n", rep.HomeDir)
	if len(rep.Home) == 0 {
		fmt.Println("  Nothing measured here.")
	}
	for i, e := range rep.Home {
		if i == 8 {
			fmt.Printf("  … and %d more\n", len(rep.Home)-i)
			break
		}
		fmt.Printf("  %-9s %s\n", hostfs.Bytes(e.Bytes), e.Path)
	}
	if rep.Other > 0 {
		fmt.Printf("\n  %s not accounted for: paths this account cannot read (docker's own\n"+
			"  storage is the usual one) and files outside your home.\n", hostfs.Bytes(rep.Other))
	}
}
