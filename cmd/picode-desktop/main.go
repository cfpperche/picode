// picode-desktop is the Windows/WSL boundary tool of PiCode Desktop
// (ADR-0020): it provisions the distro, measures and compacts its disk,
// and owns the logon task — headless, one command at a time. The shell
// (picode-shell.exe) is the resident since ADR-0142: it holds WSL open,
// keeps the tray, and drives this program as a subprocess tool.
//
// It owns only the Windows/WSL boundary. Everything inside the distro belongs
// to `picode provision`, which is why this program never mentions systemd.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/install"
	"github.com/cfpperche/picode/internal/provision"
	"github.com/cfpperche/picode/internal/version"
)

type app struct {
	runner desktop.Runner
	distro string
	user   string
}

func main() {
	// Chrome's native-messaging launch is argv[1]=chrome-extension://… and, on
	// Windows, --parent-window=<hwnd>. That flag is not ours; parsing it with
	// ExitOnError dumps usage on stdout and Chrome reports "Error when
	// communicating with the native messaging host." Host mode must run
	// before flag.Parse.
	if browserhost.IsHostArg(command()) {
		desktop.WSLExe = desktop.ResolveWSLExe()
		desktop.ResolveWindowsTools()
		exit(runBrowserHost("", ""))
		return
	}

	fs := flag.NewFlagSet("picode-desktop", flag.ExitOnError)
	distro := fs.String("distro", "", "WSL distribution (default: the only WSL 2 one, else the default)")
	user := fs.String("user", "", "Linux account to provision (default: the distro's own)")
	// Kept parsing so a pre-migration logon task fails with the retired
	// message below instead of an unknown-flag dump.
	tray := fs.Bool("tray", false, "retired with the Go tray (ADR-0142)")
	asJSON := fs.Bool("json", false, "with `disk`: emit the measurement as JSON")
	yes := fs.Bool("yes", false, "with disk-compact: stop the distro and compact without asking again; with install: install the runtime on an adopted distro without asking")
	dryRun := fs.Bool("dry-run", false, "with disk-compact: print the plan, stop nothing")
	force := fs.Bool("force", false, "with disk-compact: proceed even when someone is mid-turn")
	method := fs.String("method", "", "with disk-compact: sparse (default) | optimize-vhd")
	apply := fs.String("apply", "", "with clean: comma-separated cache ids to prune")
	listOnly := fs.Bool("list", false, "with clean: measure and print the prunable caches")
	retargetShell := fs.Bool("retarget-shell", false, "with startup-repair: move the task to the shell resident")
	fs.Usage = usage
	_ = fs.Parse(commandArgs())

	desktop.WSLExe = desktop.ResolveWSLExe()
	desktop.ResolveWindowsTools()

	switch cmd := command(); {
	case cmd == "doctor":
		exit(runDoctor(*distro, *user))
	case cmd == "disk":
		exit(runDisk(*distro, *user, *asJSON))
	case cmd == "disk-compact":
		exit(runDiskCompact(*distro, *user, *method, *yes, *dryRun, *force, *asJSON))
	case cmd == "clean":
		exit(runClean(*distro, *user, *apply, *listOnly, *yes))
	case cmd == "startup-check":
		exit(runStartupCheck())
	case cmd == "startup-repair":
		exit(runStartupRepair(*retargetShell))
	case cmd == "install":
		exit(runInstall(*distro, *user, *yes))
	case cmd == "uninstall":
		exit(runUninstall())
	case cmd == "update":
		exit(runUpdate())
	case cmd == "extension-install":
		exit(runExtensionInstall())
	case cmd == "extension-uninstall":
		exit(runExtensionUninstall())
	case cmd == "version":
		fmt.Printf("picode-desktop %s\n", version.Version)
	case cmd == "help":
		usage()
	default:
		if *tray {
			exit(runRetiredTray())
		}
		if cmd == "" {
			usage()
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
		os.Exit(2)
	}
}

// runRetiredTray is the kind failure mode for a logon task that still points
// at the Go tray: loud, on stderr, with the repair named. The task retries a
// failing launch three times and then stops — no retry storm.
func runRetiredTray() error {
	return fmt.Errorf("the Go tray retired (ADR-0142) — run `picode-desktop startup-repair --retarget-shell` to move startup to the shell")
}

// command is the first bare argument; flags may come before or after it.
func command() string {
	for _, a := range os.Args[1:] {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

func commandArgs() []string {
	var out []string
	skipped := false
	for _, a := range os.Args[1:] {
		if !skipped && !strings.HasPrefix(a, "-") {
			skipped = true
			continue
		}
		out = append(out, a)
	}
	return out
}

func usage() {
	fmt.Println(`picode-desktop — the Windows/WSL boundary tool (ADR-0020, resident retired in ADR-0142)

Usage:
  picode-desktop doctor          report what setup would change, touch nothing
  picode-desktop disk            report both halves of the disk: Windows' file and the distro's use
  picode-desktop disk-compact    give the held space back: stop the distro, convert the file, start it again
  picode-desktop startup-check   inspect Windows startup without starting WSL
  picode-desktop startup-repair  repair the existing task, without restarting anything
    --retarget-shell  move the task to the shell resident (ADR-0142) as well
  picode-desktop install         set the machine up and start with Windows
  picode-desktop uninstall       stop starting with Windows (PiCode stays installed)
  picode-desktop update          replace the tool and the shell with a newer release
  picode-desktop extension-install    register the Chrome native host (ADR-0043)
  picode-desktop extension-uninstall  remove that host registration
  picode-desktop version         print this build's version

Flags:
  --distro string   WSL distribution (default: the only WSL 2 one, else the default)
  --user string     Linux account to provision (default: the distro's own)
  --json            with the disk command: emit the measurement as JSON
  --yes             with disk-compact: stop the distro and compact without asking again
                  with install: install the runtime on an adopted distro without asking
  --dry-run         with disk-compact: print the plan, stop nothing
  --force           with disk-compact: proceed even when someone is mid-turn
  --method string   with disk-compact: sparse (default) | optimize-vhd

The distro half of the work is done by ` + "`picode provision`" + ` inside WSL.`)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "picode-desktop:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// resolve settles which distro and account this run is about. Doing it once,
// up front, keeps every later step from re-deriving it.
func resolve(distroFlag, userFlag string) (app, error) {
	a := app{runner: osRunner{}}

	distros, err := desktop.ListDistros(a.runner)
	if err != nil {
		return a, err
	}
	picked, err := desktop.Pick(distros, distroFlag)
	if err != nil {
		return a, err
	}
	a.distro = picked.Name

	a.user = userFlag
	if a.user == "" {
		if a.user, err = desktop.DefaultUser(a.runner, a.distro); err != nil {
			return a, err
		}
	}
	return a, nil
}

func runDoctor(distroFlag, userFlag string) error {
	// On a machine without WSL there is no distro to resolve, so the state of
	// the machine itself is the report.
	if stage := desktop.NextStage(desktop.Detect(osRunner{}, distroFlag)); stage != desktop.StageProvision {
		fmt.Println("This machine is not set up for PiCode yet.")
		fmt.Println()
		fmt.Printf("  %-6s %s\n", "todo", desktop.Describe(stage, distroFlag))
		if stage == desktop.StageInstallWSL {
			fmt.Printf("  %-6s %s\n", "", "Windows restarts once; setup resumes when you sign back in")
		}
		fmt.Println("\nRun `picode-desktop install` to do it.")
		return nil
	}

	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	fmt.Printf("Distro %s, account %s — reporting only, nothing will change\n\n", a.distro, a.user)

	reports, err := desktop.Provision(a.runner, a.distro, a.user, true)
	if err != nil {
		return err
	}
	merged := desktop.Merge(reports)
	printSteps(merged)
	windowsReady := printWindowsSteps(a)

	// The summary has to account for both halves. Reporting only the distro's
	// state told a fully set-up machine to run install again.
	fmt.Println()
	switch {
	case !desktop.Converged(merged):
		fmt.Printf("%d step(s) would change. Run `picode-desktop install` to apply.\n", len(pending(merged)))
	case runtime.GOOS != "windows":
		fmt.Println("The distro is ready. Run `picode-desktop install` on Windows to finish.")
	case !windowsReady:
		fmt.Println("The distro is ready. Follow the Windows startup checks above.")
	default:
		fmt.Println("Everything is set up — PiCode starts with Windows.")
	}
	return nil
}

func runInstall(distroFlag, userFlag string, yes bool) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("install runs on Windows — from inside the distro use `picode provision`")
	}
	// Installing WSL and writing to the machine certificate store both need
	// administrator rights. Ask once, here, rather than failing halfway
	// through with half a machine set up.
	if relaunched, err := elevate(); err != nil {
		return err
	} else if relaunched {
		return nil // the elevated copy took over
	}

	// A clean machine has no distro to resolve yet, so the bootstrap runs
	// first and only then is there something to name.
	pre := app{runner: osRunner{}}
	ready, err := bootstrap(&pre, distroFlag, yes)
	if err != nil {
		return err
	}
	if !ready {
		return nil // a restart is pending; setup resumes at the next logon
	}

	a, err := resolve(distroFlag, userFlag)
	if err != nil {
		return err
	}
	fmt.Printf("\nSetting PiCode up on %s for %s\n\n", a.distro, a.user)

	reports, err := desktop.Provision(a.runner, a.distro, a.user, false)
	if err != nil {
		return err
	}
	merged := desktop.Merge(reports)
	printSteps(merged)

	if !desktop.Converged(merged) {
		return fmt.Errorf("the distro is not ready — see the steps above")
	}

	if err := installWindowsSide(a); err != nil {
		return err
	}
	// Install ends with the resident running, not just registered: the
	// task launch is what a logon does, and a failure here still leaves
	// the logon path intact.
	if err := a.runner.Run("schtasks", desktop.TaskRunArgs()...); err != nil {
		fmt.Printf("  warn   the shell did not start now (%v) — it starts at sign-in\n", err)
	} else {
		fmt.Println("  ok     the shell is running in the tray")
	}
	url, err := desktop.ServerURL(a.runner, a.distro, a.user)
	if err != nil {
		return err
	}
	fmt.Printf("\nPiCode is at %s and will start with Windows.\n", url)
	return nil
}

func runUninstall() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("uninstall runs on Windows")
	}
	r := osRunner{}
	if err := r.Run("schtasks", desktop.TaskDeleteArgs()...); err != nil {
		return fmt.Errorf("remove the logon task: %w", err)
	}
	fmt.Println("PiCode no longer starts with Windows.")
	fmt.Println("It is still installed in the distro — `picode uninstall` there removes it.")
	return nil
}

// installWindowsSide is everything that lives outside the distro: trusting the
// certificate authority and registering the logon task. It runs after
// provisioning because the CA has to exist before it can be trusted.
func installWindowsSide(a app) error {
	fmt.Println()
	if err := trustCA(a); err != nil {
		// A missing CA is not fatal: PiCode serves HTTPS either way, the
		// browser just warns until the certificate is trusted.
		fmt.Printf("  warn   certificate authority not trusted: %v\n", err)
	} else {
		fmt.Println("  ok     certificate authority trusted by Windows")
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	shell, err := stageShellExe(exe)
	if err != nil {
		return err
	}
	if _, err := desktop.InstallTask(a.runner, shell, desktop.ShellArgs); err != nil {
		return fmt.Errorf("register the logon task: %w", err)
	}
	fmt.Println("  ok     starts at sign-in; no time limit; launch retry policy configured")
	return nil
}

// stageShellExe finds the shell install registers: the sibling beside this
// program when it is there, otherwise the release matching this build,
// verified before it lands. A missing shell is an error, not a tray-shaped
// fallback — there is no tray left to fall back to.
func stageShellExe(selfExe string) (string, error) {
	if shell, err := desktop.ShellExe(selfExe, os.Getenv("LOCALAPPDATA")); err == nil {
		return shell, nil
	}
	var rel install.Release
	var err error
	if tag := shellReleaseTag(); tag == "" {
		rel, err = install.LatestReleaseFor(ShellAsset)
	} else {
		rel, err = install.ReleaseByTag(tag, ShellAsset)
	}
	if err != nil {
		return "", err
	}
	if rel.AssetURL == "" {
		return "", fmt.Errorf("release %s has no %s — download it from %s", rel.Tag, ShellAsset, rel.URL)
	}
	if rel.SumsURL == "" {
		return "", fmt.Errorf("release %s has no %s — refusing to install an unverified binary", rel.Tag, install.SumsAsset)
	}
	sums, err := install.Fetch(rel.SumsURL)
	if err != nil {
		return "", fmt.Errorf("%s: %w", install.SumsAsset, err)
	}
	dest := filepath.Join(filepath.Dir(selfExe), desktop.ShellExeName)
	if err := install.Download(rel.AssetURL, dest); err != nil {
		return "", fmt.Errorf("download %s: %w", ShellAsset, err)
	}
	if err := install.VerifySHA256(dest, sums, ShellAsset); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	return dest, nil
}

// shellReleaseTag pins the staged shell to this build's release. An
// unstamped source build tracks latest instead — "" means "latest".
func shellReleaseTag() string {
	if version.Stamped == "release" && version.Version != "" {
		return "v" + version.Version
	}
	return ""
}

func trustCA(a app) error {
	out, err := a.runner.Output("powershell", desktop.CACountArgs()...)
	if err == nil && desktop.CATrusted(out) {
		return nil
	}
	path, err := desktop.ExportCA(a.runner, a.distro, a.user)
	if err != nil {
		return err
	}
	return a.runner.Run("powershell", desktop.CAImportArgs(path)...)
}

// printWindowsSteps reports the half that lives outside the distro, and says
// whether it is finished.
func printWindowsSteps(a app) (ready bool) {
	if runtime.GOOS != "windows" {
		fmt.Printf("  %-6s %-44s %s\n", "skip", "Windows setup", "not running on Windows")
		return false
	}

	out, err := a.runner.Output("powershell", desktop.CACountArgs()...)
	caTrusted := err == nil && desktop.CATrusted(out)
	if caTrusted {
		fmt.Printf("  %-6s %-44s %s\n", "ok", "certificate authority trusted by Windows", `mkcert root in LocalMachine\Root`)
	} else {
		fmt.Printf("  %-6s %-44s %s\n", "todo", "certificate authority trusted by Windows", "not imported yet")
	}

	task, taskErr := desktop.InspectTask(a.runner)
	taskReady := printStartupStatus(os.Stdout, task, taskErr)
	return caTrusted && taskReady
}

func printSteps(steps []provision.Result) {
	for _, s := range steps {
		state := s.Before
		if s.After != nil {
			state = *s.After
		}
		fmt.Printf("  %-6s %-44s %s\n", mark(s.Action), s.Title, state.Detail)
		if s.Error != "" {
			fmt.Printf("         %-44s %s\n", "", s.Error)
		}
	}
}

func pending(steps []provision.Result) []provision.Result {
	var out []provision.Result
	for _, s := range steps {
		if s.Action != provision.ActionNone && s.Action != provision.ActionFixed {
			out = append(out, s)
		}
	}
	return out
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
