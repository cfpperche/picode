package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/hostfs"
)

// diskReport is one disk, measured from both sides: what Windows holds for the
// distro's file, and what the distro itself is using. Either half can be
// missing, and a report that says which half is missing is worth more than a
// report that quietly shows zeroes.
type diskReport struct {
	Windows      *desktop.DiskFacts `json:"windows,omitempty"`
	WindowsError string             `json:"windowsError,omitempty"`
	Distro       *hostfs.Report     `json:"distro,omitempty"`
	DistroError  string             `json:"distroError,omitempty"`
	// System is the root-owned caches (ADR-0198), measured as root.
	System      []desktop.MeasuredSystemCache `json:"system,omitempty"`
	SystemError string                        `json:"systemError,omitempty"`
	// Held is what the VHDX holds that the distro has already freed: the space
	// Windows will not get back until the file is compacted or made sparse.
	Held int64     `json:"held,omitempty"`
	At   time.Time `json:"at"`
}

// runDisk prints the two-sided report. It changes nothing: making the file
// sparse or compacting it stops the distro and costs the sessions in it, so
// that decision belongs to a reviewed, confirmed action
// (docs/plans/wsl-control.md), never to a command someone runs to read.
//
// With stream (and JSON), each half is printed as a scanStep line the moment
// it is read, and the whole report is the last line — the same one-object-
// per-line contract disk-compact and clean use, so the Management window
// shows the scan as it happens instead of a page of dashes.
func runDisk(distroFlag, userFlag string, asJSON, stream bool) error {
	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	var onStep func(scanStep)
	if asJSON && stream {
		enc := json.NewEncoder(os.Stdout)
		onStep = func(s scanStep) { _ = enc.Encode(s) }
	}
	rep := collectDisk(a, onStep)

	if asJSON && stream {
		return json.NewEncoder(os.Stdout).Encode(rep)
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	printDisk(a.distro, rep)
	return nil
}

// scanStep is one line of a streamed scan: which half, where it is, and the
// half itself once it is read. Progress is the sentence a person reads; the
// shell forwards every line that carries it and treats the first line
// without it as the outcome.
type scanStep struct {
	Progress string                        `json:"progress"`
	Stage    string                        `json:"stage"` // "windows" | "distro" | "system"
	State    string                        `json:"state"` // "running" | "done" | "failed"
	Windows  *desktop.DiskFacts            `json:"windows,omitempty"`
	Distro   *hostfs.Report                `json:"distro,omitempty"`
	System   []desktop.MeasuredSystemCache `json:"system,omitempty"`
	Error    string                        `json:"error,omitempty"`
}

// collectDisk reads both halves, keeping each failure with its half. The
// Windows half goes first: it is a few registry and file reads, while the
// distro half walks the home directory, so the first card fills in seconds.
// onStep, when set, hears each half start and end.
func collectDisk(a app, onStep func(scanStep)) diskReport {
	step := func(s scanStep) {
		if onStep != nil {
			onStep(s)
		}
	}
	rep := diskReport{At: time.Now().UTC()}

	step(scanStep{Stage: "windows", State: "running", Progress: "Reading the disk file from Windows"})
	facts, err := desktop.DistroDisk(a.runner, a.distro)
	if err != nil {
		rep.WindowsError = err.Error()
		step(scanStep{Stage: "windows", State: "failed", Progress: "Windows could not be read", Error: rep.WindowsError})
	} else {
		rep.Windows = &facts
		step(scanStep{Stage: "windows", State: "done", Progress: "Read the disk file from Windows", Windows: &facts})
	}

	step(scanStep{Stage: "distro", State: "running", Progress: "Measuring inside " + a.distro})
	distro, err := distroReport(a.runner, a.distro, a.user)
	if err != nil {
		rep.DistroError = err.Error()
		step(scanStep{Stage: "distro", State: "failed", Progress: a.distro + " could not be measured", Error: rep.DistroError})
	} else {
		rep.Distro = &distro
		step(scanStep{Stage: "distro", State: "done", Progress: "Measured inside " + a.distro, Distro: &distro})
	}

	step(scanStep{Stage: "system", State: "running", Progress: "Measuring system caches"})
	// Bounded on the real runner: a hung `wsl -u root` must not hold the
	// scan (and the shell's one-job lock) with it.
	sysRunner := a.runner
	if _, real := a.runner.(osRunner); real {
		sysRunner = timedRunner{d: 60 * time.Second}
	}
	if sys, err := desktop.MeasureSystemCaches(sysRunner, a.distro); err != nil {
		rep.SystemError = err.Error()
		step(scanStep{Stage: "system", State: "failed", Progress: "System caches could not be measured", Error: rep.SystemError})
	} else {
		rep.System = sys
		step(scanStep{Stage: "system", State: "done", Progress: "Measured system caches", System: sys})
	}

	if rep.Windows != nil && rep.Distro != nil {
		rep.Held = desktop.Held(*rep.Windows, rep.Distro.FS.UsedBytes)
	}
	return rep
}

// distroReport runs the Linux half in the distro, the same way provisioning
// does: the Windows program never learns what a filesystem is, it asks the
// binary inside.
func distroReport(r desktop.Runner, distro, user string) (hostfs.Report, error) {
	exe, err := desktop.PicodePath(r, distro, user)
	if err != nil {
		return hostfs.Report{}, err
	}
	outs, err := r.Output(desktop.WSLExe, desktop.WSLArgs(distro, user, exe, "disk", "--json")...)
	if err != nil && len(outs) == 0 {
		return hostfs.Report{}, fmt.Errorf("picode disk: %w", err)
	}
	text := desktop.DecodeWindows(outs)
	start := strings.Index(text, "{")
	if start < 0 {
		if err != nil {
			return hostfs.Report{}, fmt.Errorf("picode disk: %w", err)
		}
		// The usual cause is an older picode inside the distro, which refuses a
		// subcommand it does not know rather than printing JSON.
		return hostfs.Report{}, fmt.Errorf("picode disk printed no JSON — is the picode in %s older than this program?", distro)
	}
	var rep hostfs.Report
	if err := json.Unmarshal([]byte(text[start:]), &rep); err != nil {
		return hostfs.Report{}, fmt.Errorf("picode disk JSON: %w", err)
	}
	return rep, nil
}

// distroUsed reads one number, the way the disk line needs it: a df inside the
// distro, without walking a single directory.
func distroUsed(r desktop.Runner, distro, user string) (int64, error) {
	out, err := r.Output(desktop.WSLExe, desktop.WSLArgs(distro, user, "df", "-B1", "--output=size,used,avail,target")...)
	if err != nil && len(out) == 0 {
		return 0, fmt.Errorf("df in %s: %w", distro, err)
	}
	root, err := hostfs.Root(hostfs.ParseDF([]byte(desktop.DecodeWindows(out))))
	if err != nil {
		return 0, err
	}
	return root.UsedBytes, nil
}

// printDisk writes the report for a person: Windows first, because the number
// they went looking for is the one on the drive letter.
func printDisk(distro string, rep diskReport) {
	fmt.Printf("%s — Windows and the distro, %s\n\n", distro, rep.At.Local().Format("15:04"))

	fmt.Println("Windows")
	if rep.WindowsError != "" {
		fmt.Printf("  Could not be read: %s\n", rep.WindowsError)
	} else {
		f := rep.Windows
		fmt.Printf("  %-24s %s free of %s\n", driveOf(f.VHDXPath)+":", hostfs.Bytes(f.FreeBytes), hostfs.Bytes(f.VolumeBytes))
		if f.Sparse {
			fmt.Printf("  %-24s %s long · %s on disk · sparse\n", "its disk file",
				hostfs.Bytes(f.VHDXBytes), hostfs.Bytes(f.AllocatedBytes))
		} else {
			fmt.Printf("  %-24s %s on disk · not sparse\n", "its disk file", hostfs.Bytes(f.AllocatedBytes))
		}
		fmt.Printf("  %-24s %s\n", "", f.VHDXPath)
		switch {
		case rep.Held > 0:
			fmt.Printf("  %-24s ≈%s the distro has already freed\n", "held for nothing", hostfs.Bytes(rep.Held))
			if f.Sparse {
				fmt.Println("                           WSL gives that back on its own while the file is sparse; compacting it gives it back now.")
			} else {
				fmt.Println("                           Windows keeps every block it ever wrote while the file is not sparse.")
				if f.CanSparse {
					fmt.Printf("                           WSL %s can convert it: wsl --manage %s --set-sparse true\n", f.WSL, distro)
				}
			}
		case f.Sparse:
			fmt.Println("                           Nothing is held back: the file is sparse, so WSL returns freed blocks.")
		}
	}

	fmt.Println("\nInside " + distro)
	if rep.DistroError != "" {
		fmt.Printf("  Could not be measured: %s\n", rep.DistroError)
	} else {
		d := rep.Distro
		fmt.Printf("  %-24s %s used of %s\n", "/", hostfs.Bytes(d.FS.UsedBytes), hostfs.Bytes(d.FS.SizeBytes))
		if giveback := hostfs.Reclaimable(d.Consumers, hostfs.KindSafe, hostfs.KindRedownload); giveback > 0 {
			fmt.Printf("  %-24s %s · %s of it costs only the time to rebuild\n", "safe to reclaim",
				hostfs.Bytes(giveback), hostfs.Bytes(hostfs.Reclaimable(d.Consumers, hostfs.KindSafe)))
		}
		var biggest []string
		for i, c := range d.Consumers {
			if i == 3 {
				break
			}
			biggest = append(biggest, fmt.Sprintf("%s %s", c.Title, hostfs.Bytes(c.Bytes)))
		}
		if len(biggest) > 0 {
			fmt.Printf("  %-24s %s\n", "biggest", strings.Join(biggest, " · "))
		}
	}

	fmt.Println("\nNothing was changed. `picode disk` inside the distro lists every item in full.")
}

// driveOf names the volume a Windows path lives on, so the report shows the
// drive letter a person recognises instead of assuming C:.
func driveOf(path string) string {
	if len(path) >= 2 && path[1] == ':' {
		return strings.ToUpper(path[:1])
	}
	return "C"
}

// The two numbers that make the disk line worth reading: a volume this close to
// full is one that starts refusing writes, and a file holding this much unused
// space is worth one conversion.
const (
	lowFree = 20 << 30 // 20 GiB
	// heldWorthTelling is deliberately smaller than a Go build cache: the
	// point is to name the day the distro started costing real space, not to
	// nag about the first gigabyte.
	heldWorthTelling = 8 << 30 // 8 GiB
)

// diskLine is the one line about the disk, and its pure half: the wording is
// tested here instead of discovered in a
// screenshot nobody can take from CI.
//
// It returns the menu title and whether the wording is a warning. A line with
// no facts says so rather than showing zero, which would read as "the disk is
// empty".
func diskLine(facts *desktop.DiskFacts, usedBytes int64, err error) (title string, warn bool) {
	if err != nil || facts == nil {
		return "Disk: not read", false
	}

	held := desktop.Held(*facts, usedBytes)
	parts := []string{"WSL " + hostfs.Bytes(facts.AllocatedBytes)}
	if held >= heldWorthTelling {
		parts = append(parts, "≈"+hostfs.Bytes(held)+" held by Windows")
	}
	parts = append(parts, "C: "+hostfs.Bytes(facts.FreeBytes)+" free")

	warn = facts.FreeBytes < lowFree
	if warn {
		parts[len(parts)-1] += " — low"
	}
	return strings.Join(parts, " · "), warn
}
