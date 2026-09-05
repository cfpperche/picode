//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// Elevation is decided at runtime rather than declared in a manifest. A
// `requireAdministrator` manifest applies to the whole executable, and this one
// is also the tray — which must stay unelevated, or it cannot reach Explorer's
// notification area and would hand the browser administrator rights. So only
// installation asks up front; startup repair asks only after access denied.

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
	procIsUserAdmin  = shell32.NewProc("IsUserAnAdmin")
)

// isAdmin reports whether this process is already elevated.
func isAdmin() bool {
	ret, _, _ := procIsUserAdmin.Call()
	return ret != 0
}

// elevate re-launches this program with the same arguments through the "runas"
// verb, which is what raises the UAC prompt. It returns true when a child was
// started and this process should simply exit.
func elevate() (bool, error) {
	if isAdmin() {
		return false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	args, _ := syscall.UTF16PtrFromString(quoteArgs(os.Args[1:]))

	const swShowNormal = 1
	ret, _, callErr := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(args)),
		0,
		swShowNormal,
	)
	// ShellExecuteW returns >32 on success. Anything at or below that is an
	// error code, and 5 (ERROR_ACCESS_DENIED) is specifically the user
	// declining the prompt — worth saying plainly rather than as a number.
	if ret <= 32 {
		if ret == 5 {
			return false, fmt.Errorf("administrator rights were declined")
		}
		return false, fmt.Errorf("could not ask for administrator rights (code %d): %v", ret, callErr)
	}
	return true, nil
}

// quoteArgs rebuilds a command line, quoting anything containing a space so a
// distro named "My Distro" survives the round trip.
func quoteArgs(args []string) string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if strings.ContainsAny(a, " \t") {
			a = `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
		}
		out = append(out, a)
	}
	return strings.Join(out, " ")
}
