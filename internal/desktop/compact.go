package desktop

import (
	"fmt"
	"strings"
)

// The compact flow gives a WSL disk file's held space back to Windows: stop
// the distro, convert the file to sparse (or compact it), start the distro
// again. It exists because a permanently running distro never gets the chance
// — WSL can only rewrite the file while nothing holds it open, and PiCode's
// own keepalive holds it open by design (ADR-0020).
//
// Stopping the distro ends every session inside it. Nothing here decides that:
// the callers show the cost and ask (the tray confirms in a dialog, the CLI
// wants --yes), and the server's readiness interlock is consulted before this
// code runs at all.

// CompactMethod is how the space comes back.
type CompactMethod string

const (
	// CompactSparse converts the file in place. No administrator rights, and
	// WSL keeps reclaiming freed blocks from then on. WSL 2.0+ only.
	CompactSparse CompactMethod = "sparse"
	// CompactOptimizeVHD compacts with Hyper-V's Optimize-VHD. Needs an
	// elevated terminal, and the Hyper-V module, which Windows Home lacks.
	CompactOptimizeVHD CompactMethod = "optimize-vhd"
)

// CompactResult is what the run did, measured rather than promised.
type CompactResult struct {
	Method      CompactMethod `json:"method"`
	BeforeBytes int64         `json:"beforeBytes"`
	AfterBytes  int64         `json:"afterBytes"`
	Sparse      bool          `json:"sparse"`
}

// Returned is the space the compact gave back to the volume.
func (res CompactResult) Returned() int64 {
	if res.BeforeBytes <= 0 || res.AfterBytes >= res.BeforeBytes {
		return 0
	}
	return res.BeforeBytes - res.AfterBytes
}

// PlanCompact picks the method this machine can run and says why. The sparse
// conversion wins whenever the WSL build supports it: no elevation, and the
// reclaim continues without anyone asking again.
func PlanCompact(facts DiskFacts) (CompactMethod, string) {
	if facts.CanSparse {
		return CompactSparse, "convert the disk file to sparse — WSL returns freed blocks on its own from then on"
	}
	return CompactOptimizeVHD, "compact the disk file with Optimize-VHD — from an administrator terminal"
}

// TerminateArgs stops the distro. Everything inside it ends with it.
func TerminateArgs(distro string) []string {
	return []string{"--terminate", distro}
}

// SetSparseArgs converts the distro's disk file to sparse. The distro must be
// stopped; the file is the user's own, so no elevation is involved.
func SetSparseArgs(distro string) []string {
	return []string{"--manage", distro, "--set-sparse", "true"}
}

// OptimizeVHDArgs compacts the file. PowerShell, because Optimize-VHD is a
// Hyper-V module cmdlet; -LiteralPath because the WSL folder name has braces.
func OptimizeVHDArgs(path string) []string {
	if strings.Contains(path, "'") {
		// Unreachable from the registry's BasePath; refused rather than escaped
		// so a surprising path fails loudly instead of surprisingly.
		return []string{"Optimize-VHD", "refused: quote in path"}
	}
	return []string{
		"-NoProfile", "-NonInteractive", "-Command",
		`Optimize-VHD -LiteralPath '` + path + `' -Mode Full`,
	}
}

// StartDistroArgs boots the distro again. Any command does it; `true` is the
// most honest one — this exists to wake the VM, nothing else.
func StartDistroArgs(distro, user string) []string {
	return WSLArgs(distro, user, "true")
}

// FileFootprint is what a disk file costs on its volume right now: the bytes
// NTFS holds, and whether it is sparse. Non-sparse means every byte of the
// file is on the volume and no second measurement is needed.
func FileFootprint(r Runner, path string) (allocated int64, sparse bool, err error) {
	_, allocated, sparse, err = footprintOf(r, path)
	return allocated, sparse, err
}

// footprintOf reads the file's length, the volume around it and the allocated
// extents in the two calls those facts take.
func footprintOf(r Runner, path string) (vol Volume, allocated int64, sparse bool, err error) {
	vol, err = volumeOf(r, path)
	if err != nil {
		return Volume{}, 0, false, err
	}
	allocated, sparse, err = sparseRange(r, path)
	if err != nil {
		return Volume{}, 0, false, err
	}
	if !sparse {
		allocated = vol.Length
	}
	return vol, allocated, sparse, nil
}

// Compact runs stop → convert → start, and measures the file before and after
// so the caller reports a number, not a hope. The distro comes back up even
// when the conversion fails: a machine left without its distro is worse than
// the state this started in, and the keepalive and the service both expect it.
//
// The caller owns the decision. This function never asks.
func Compact(r Runner, distro, user string, method CompactMethod, vhdxPath string, before int64, progress func(string)) (res CompactResult, err error) {
	if method != CompactSparse && method != CompactOptimizeVHD {
		return res, fmt.Errorf("unknown compact method %q", method)
	}
	res.Method = method
	res.BeforeBytes = before

	say := func(s string) {
		if progress != nil {
			progress(s)
		}
	}

	say("stopping " + distro + " — everything inside it ends")
	if err = r.Run(WSLExe, TerminateArgs(distro)...); err != nil {
		return res, fmt.Errorf("stop %s: %w", distro, err)
	}

	defer func() {
		say("restarting " + distro)
		if startErr := r.Run(WSLExe, StartDistroArgs(distro, user)...); startErr != nil && err == nil {
			err = fmt.Errorf("restart %s: %w", distro, startErr)
		}
	}()

	switch method {
	case CompactSparse:
		say("converting the disk file to sparse")
		if err = r.Run(WSLExe, SetSparseArgs(distro)...); err != nil {
			return res, fmt.Errorf("convert to sparse: %w", err)
		}
	case CompactOptimizeVHD:
		say("compacting with Optimize-VHD")
		if err = r.Run(PowerShellExe, OptimizeVHDArgs(vhdxPath)...); err != nil {
			return res, fmt.Errorf("Optimize-VHD: %w", err)
		}
	}

	after, sparse, mErr := FileFootprint(r, vhdxPath)
	if mErr != nil {
		// The conversion ran; only the measurement failed. The caller hears
		// about it, and the before number stays the only honest one.
		return res, fmt.Errorf("measure %s after compacting: %w", vhdxPath, mErr)
	}
	res.AfterBytes = after
	res.Sparse = sparse
	return res, nil
}
