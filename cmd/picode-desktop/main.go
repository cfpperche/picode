// picode-desktop is the Windows half of PiCode Desktop (ADR-0020). It brings
// the WSL distro up at logon, drives `picode provision` inside it, and then
// lives in the notification area so PiCode is one click away.
//
// It owns only the Windows/WSL boundary. Everything inside the distro belongs
// to `picode provision`, which is why this program never mentions systemd.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/provision"
	"github.com/cfpperche/picode/internal/version"
)

type app struct {
	runner desktop.Runner
	distro string
	user   string
}

func main() {
	fs := flag.NewFlagSet("picode-desktop", flag.ExitOnError)
	distro := fs.String("distro", "", "WSL distribution (default: the only WSL 2 one, else the default)")
	user := fs.String("user", "", "Linux account to provision (default: the distro's own)")
	tray := fs.Bool("tray", false, "run in the notification area (the logon task passes this)")
	fs.Usage = usage
	_ = fs.Parse(commandArgs())

	desktop.WSLExe = desktop.ResolveWSLExe()

	switch command() {
	case "doctor":
		exit(runDoctor(*distro, *user))
	case "install":
		exit(runInstall(*distro, *user))
	case "uninstall":
		exit(runUninstall())
	case "update":
		exit(runUpdate())
	case "version":
		fmt.Printf("picode-desktop %s\n", version.Version)
	case "help":
		usage()
	default:
		if *tray || command() == "" {
			exit(runTray(*distro, *user))
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command())
		usage()
		os.Exit(2)
	}
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
	fmt.Println(`picode-desktop — PiCode in the Windows notification area (ADR-0020)

Usage:
  picode-desktop                 run in the notification area
  picode-desktop doctor          report what setup would change, touch nothing
  picode-desktop install         set the machine up and start with Windows
  picode-desktop uninstall       stop starting with Windows (PiCode stays installed)
  picode-desktop update          replace this program with a newer release
  picode-desktop version         print this build's version

Flags:
  --distro string   WSL distribution (default: the only WSL 2 one, else the default)
  --user string     Linux account to provision (default: the distro's own)
  --tray            run in the notification area (the logon task passes this)

The distro half of the work is done by ` + "`picode provision`" + ` inside WSL.`)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "picode-desktop:", err)
		os.Exit(1)
	}
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
	printWindowsSteps(a)

	if desktop.Converged(merged) {
		fmt.Println("\nThe distro is ready. Run `picode-desktop install` to start with Windows.")
	} else {
		fmt.Printf("\n%d step(s) would change. Run `picode-desktop install` to apply.\n", len(pending(merged)))
	}
	return nil
}

func runInstall(distroFlag, userFlag string) error {
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
	ready, err := bootstrap(&pre, distroFlag)
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
	if err := a.runner.Run("schtasks", desktop.TaskCreateArgs(exe)...); err != nil {
		return fmt.Errorf("register the logon task: %w", err)
	}
	fmt.Println("  ok     starts with Windows")
	return nil
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

func printWindowsSteps(a app) {
	if runtime.GOOS != "windows" {
		fmt.Printf("  %-6s %-44s %s\n", "skip", "Windows setup", "not running on Windows")
		return
	}
	out, err := a.runner.Output("powershell", desktop.CACountArgs()...)
	switch {
	case err == nil && desktop.CATrusted(out):
		fmt.Printf("  %-6s %-44s %s\n", "ok", "certificate authority trusted by Windows", "")
	default:
		fmt.Printf("  %-6s %-44s %s\n", "todo", "certificate authority trusted by Windows", "not imported yet")
	}
	if err := a.runner.Run("schtasks", desktop.TaskQueryArgs()...); err == nil {
		fmt.Printf("  %-6s %-44s %s\n", "ok", "starts with Windows", desktop.TaskName)
	} else {
		fmt.Printf("  %-6s %-44s %s\n", "todo", "starts with Windows", "logon task not registered")
	}
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
