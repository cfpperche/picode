package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/install"
)

// install-picode and install-runtime executors (ADR-0098): the work behind
// the two stages the bootstrap loop routes to. Each one checks first, fixes
// second, verifies third; re-running a finished stage changes nothing.

// ensureTargetUser resolves the account a stage installs for and proves it
// exists before anything is downloaded: `id` as root is both the existence
// check and the name validator (a hostile --user arrives as one argv element,
// and id refuses it the same way it refuses a ghost).
func ensureTargetUser(a *app, distro string, state desktop.MachineState, userFlag string) (string, error) {
	target := state.TargetUser
	if target == "" {
		target = state.DefaultUser
	}
	if userFlag == "" {
		return target, nil
	}
	if _, err := a.runner.Output(desktop.WSLExe, desktop.WSLArgs(distro, "root", "id", "-u", userFlag)...); err != nil {
		return "", fmt.Errorf("unknown Linux account %q on %s", userFlag, distro)
	}
	return userFlag, nil
}

// runInstallPicode delivers the Linux binary into the distro: the release
// matching this build (latest when unstamped), verified, staged through a
// temp file the distro reads over /mnt, and version-checked through a login
// shell — the same shell PicodePath resolves through later.
func runInstallPicode(a *app, state desktop.MachineState, distroFlag, userFlag string) error {
	picked, err := desktop.Pick(state.Distros, distroFlag)
	if err != nil {
		return err
	}
	user, err := ensureTargetUser(a, picked.Name, state, userFlag)
	if err != nil {
		return err
	}
	asset, err := desktop.LinuxAssetName(runtime.GOARCH)
	if err != nil {
		return err
	}
	rel, err := linuxRelease(asset)
	if err != nil {
		return err
	}
	if rel.AssetURL == "" {
		return fmt.Errorf("release %s has no %s — download it from %s", rel.Tag, asset, rel.URL)
	}
	if rel.SumsURL == "" {
		return fmt.Errorf("release %s has no %s — refusing to install an unverified binary", rel.Tag, install.SumsAsset)
	}
	want := strings.TrimPrefix(rel.Tag, "v")
	if state.PicodeVersion == want {
		fmt.Printf("  ok     picode %s is already in %s — leaving it alone\n", want, picked.Name)
		return nil
	}

	bin, err := fetchLinuxBinary(rel, asset)
	if err != nil {
		return err
	}
	defer os.RemoveAll(bin.tmp)
	inside, err := desktop.WSLPath(bin.staged)
	if err != nil {
		return err
	}
	if err := a.runner.Run(desktop.WSLExe, desktop.WSLArgs(picked.Name, user, "sh", "-c", desktop.PlacePicodeScript(inside))...); err != nil {
		return fmt.Errorf("place the picode binary: %w", err)
	}
	out, err := a.runner.Output(desktop.WSLExe, desktop.WSLArgs(picked.Name, user, "bash", "-lc", "~/.local/bin/picode --version")...)
	if err != nil {
		return fmt.Errorf("verify the picode binary: %w", err)
	}
	got := picodeVersionOf(out)
	if got == "" {
		return fmt.Errorf("the installed picode does not report a version")
	}
	if got != want {
		return fmt.Errorf("installed picode %s, wanted %s", got, want)
	}
	fmt.Printf("  ok     picode %s installed in %s\n", got, picked.Name)
	return nil
}

// linuxRelease resolves the release install-picode downloads: this build's
// own tag when stamped, latest otherwise.
func linuxRelease(asset string) (install.Release, error) {
	if tag := shellReleaseTag(); tag == "" {
		return install.LatestReleaseFor(asset)
	} else {
		return install.ReleaseByTag(tag, asset)
	}
}

// linuxBinary is a verified Linux binary waiting in a temp dir.
type linuxBinary struct {
	tmp    string
	staged string
}

// fetchLinuxBinary downloads the release asset into a temp dir and verifies
// it against SHA256SUMS. The caller removes tmp.
func fetchLinuxBinary(rel install.Release, asset string) (linuxBinary, error) {
	sums, err := install.Fetch(rel.SumsURL)
	if err != nil {
		return linuxBinary{}, fmt.Errorf("%s: %w", install.SumsAsset, err)
	}
	tmp, err := os.MkdirTemp("", "picode-install-")
	if err != nil {
		return linuxBinary{}, err
	}
	staged := filepath.Join(tmp, asset)
	if err := install.Download(rel.AssetURL, staged); err != nil {
		os.RemoveAll(tmp)
		return linuxBinary{}, fmt.Errorf("download %s: %w", asset, err)
	}
	if err := install.VerifySHA256(staged, sums, asset); err != nil {
		os.RemoveAll(tmp)
		return linuxBinary{}, err
	}
	return linuxBinary{tmp: tmp, staged: staged}, nil
}

// picodeVersionOf reads `picode --version` ("picode 0.3.1", "picode
// 0.3.1+a892ede").
func picodeVersionOf(out []byte) string {
	fields := strings.Fields(desktop.DecodeWindows(out))
	if len(fields) != 2 || fields[0] != "picode" {
		return ""
	}
	if v, _, _ := strings.Cut(fields[1], "+"); v != "" {
		return v
	}
	return ""
}

// runInstallRuntime converges the distro on tmux, git, curl, node and npm —
// never on an agent CLI (ADR-0179: the user installs those from Agent CLIs).
// Ubuntu only: any other family gets the missing list and stops. On a
// distro this program did not register, anything that would change asks
// first (yes skips the question); a stage with nothing to do never asks.
func runInstallRuntime(a *app, state desktop.MachineState, distroFlag, userFlag string, yes bool, stdin io.Reader) error {
	picked, err := desktop.Pick(state.Distros, distroFlag)
	if err != nil {
		return err
	}
	user, err := ensureTargetUser(a, picked.Name, state, userFlag)
	if err != nil {
		return err
	}
	if len(state.Missing) == 0 {
		fmt.Printf("  ok     tmux, git, curl, node and npm are already in %s\n", picked.Name)
		return nil
	}
	onlyPrefix := len(state.Missing) == 1 && state.Missing[0] == desktop.NpmUserPrefix
	if state.Family != "ubuntu" && !onlyPrefix {
		return fmt.Errorf("%s is %s, not Ubuntu — automatic setup only knows Ubuntu.\nInstall inside the distro: %s, then re-run install",
			picked.Name, familyName(state.Family), strings.Join(state.Missing, ", "))
	}

	missing := map[string]bool{}
	for _, t := range state.Missing {
		missing[t] = true
	}
	// Node comes from NodeSource at the CI major when node or npm is missing
	// or node is older than that major (the probe names it NodeUpgrade). A
	// fresh NodeSource npm has a root-owned /usr prefix, so the account's
	// prefix moves to ~/.local with it: Agent CLIs installs as the account.
	needNode := missing["node"] || missing["npm"] || missing[desktop.NodeUpgrade]
	needPrefix := needNode || missing[desktop.NpmUserPrefix]
	nodeMajor := desktop.RuntimeNodeMajor
	var base []string
	for _, p := range desktop.BasePackages {
		if missing[p] {
			base = append(base, p)
		}
	}

	// The confirm names the changes, not the stage: on someone else's
	// distro "install the runtime" is not informed consent.
	if !state.RegisteredByDesktop && !yes {
		actions := []string{}
		if len(base) > 0 {
			actions = append(actions, "apt-get install "+strings.Join(base, " "))
		}
		if needNode {
			line := "nodejs " + nodeMajor + " from the NodeSource repository"
			if missing[desktop.NodeUpgrade] {
				line += " (replaces node " + state.NodeMajor + ")"
			}
			actions = append(actions, line)
		}
		if needPrefix {
			actions = append(actions, "npm config set prefix ~/.local as "+user+" (so Agent CLIs installs without root)")
		}
		ok, err := confirm(stdin, fmt.Sprintf("%s is your distro, not one PiCode created.\nRun inside %s:\n  %s\nProceed?", picked.Name, picked.Name, strings.Join(actions, "\n  ")))
		if err != nil {
			return fmt.Errorf("confirm the runtime install: %w (or re-run with --yes)", err)
		}
		if !ok {
			return fmt.Errorf("runtime install declined — re-run with --yes to proceed without asking")
		}
	}

	asRoot := func(command ...string) error {
		return a.runner.Run(desktop.WSLExe, desktop.WSLArgs(picked.Name, "root", command...)...)
	}
	if len(base) > 0 || needNode {
		if err := asRoot("apt-get", "update"); err != nil {
			return fmt.Errorf("apt-get update: %w", err)
		}
	}
	if len(base) > 0 {
		if err := asRoot(append([]string{"apt-get", "install", "-y"}, base...)...); err != nil {
			return fmt.Errorf("apt-get install %s: %w", strings.Join(base, " "), err)
		}
		fmt.Printf("  ok     %s installed\n", strings.Join(base, ", "))
	}
	// mkcert is a courtesy, not a promise: archives without it keep the
	// provision cert step's existing message.
	if err := asRoot("apt-get", "install", "-y", "mkcert"); err != nil {
		fmt.Println("  warn   mkcert is not in this archive — the certificate step will say so")
	}
	if needNode {
		script, err := desktop.NodeSourceScript(nodeMajor)
		if err != nil {
			return err
		}
		if err := asRoot("sh", "-c", script); err != nil {
			return fmt.Errorf("install nodejs %s: %w", nodeMajor, err)
		}
		fmt.Printf("  ok     nodejs %s installed\n", nodeMajor)
	}
	if needPrefix {
		if err := a.runner.Run(desktop.WSLExe, desktop.WSLArgs(picked.Name, user, "sh", "-c", desktop.NpmUserPrefixScript())...); err != nil {
			return fmt.Errorf("set %s's npm prefix to ~/.local: %w", user, err)
		}
		fmt.Printf("  ok     npm installs as %s into ~/.local\n", user)
	}
	out, err := a.runner.Output(desktop.WSLExe, desktop.WSLArgs(picked.Name, user, desktop.ProbeArgs()...)...)
	if err != nil {
		return fmt.Errorf("verify the runtime: %w", err)
	}
	if still := desktop.ParseProbe(out).Missing; len(still) > 0 {
		return fmt.Errorf("still missing after install: %s", strings.Join(still, ", "))
	}
	return nil
}

func familyName(family string) string {
	if family == "" || family == "unknown" {
		return "an unidentified distribution"
	}
	return family
}

// confirm asks one Y/n question on stdin. An empty answer means yes; a
// closed stdin refuses with an error instead of guessing.
func confirm(stdin io.Reader, prompt string) (bool, error) {
	fmt.Printf("%s [Y/n] ", prompt)
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("no answer on stdin")
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "", "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
