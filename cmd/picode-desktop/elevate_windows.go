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
// `requireAdministrator` manifest applies to the whole executable, and this
// one does its everyday reads — disk, doctor, the shell's probes —
// unelevated; prompting for those would train the owner to click Yes
// without reading. So only installation asks up front; startup repair asks
// only after access denied.

var (
	shell32            = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteEx = shell32.NewProc("ShellExecuteExW")
	procIsUserAdmin    = shell32.NewProc("IsUserAnAdmin")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procWaitForSingle  = kernel32.NewProc("WaitForSingleObject")
	procGetExitCode    = kernel32.NewProc("GetExitCodeProcess")
)

const (
	seeMaskNoCloseProcess = 0x40
	errCancelled          = 1223 // ERROR_CANCELLED: the user declined the prompt
	infinite              = 0xFFFFFFFF
	swShowNormal          = 1
)

// shellExecuteInfo mirrors SHELLEXECUTEINFOW. Field order is the layout:
// Go's alignment matches the C struct on amd64 and arm64.
type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hIconOrMon   uintptr
	hProcess     uintptr
}

// isAdmin reports whether this process is already elevated.
func isAdmin() bool {
	ret, _, _ := procIsUserAdmin.Call()
	return ret != 0
}

// elevate re-launches this program with the same arguments through the "runas"
// verb, which is what raises the UAC prompt. It returns true when a child took
// over and this process should simply exit. The parent waits for the child
// and reports a failing exit code: returning before the install finished is
// how a failure once hid behind an exit 0.
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

	info := shellExecuteInfo{
		fMask:        seeMaskNoCloseProcess,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: args,
		nShow:        swShowNormal,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))

	ret, _, callErr := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		// A declined prompt deserves plain words rather than a number.
		if errno, ok := callErr.(syscall.Errno); ok && errno == errCancelled {
			return false, fmt.Errorf("administrator rights were declined")
		}
		return false, fmt.Errorf("could not ask for administrator rights: %v", callErr)
	}
	defer syscall.CloseHandle(syscall.Handle(info.hProcess))

	// The return value is the signal for both calls; the last error is
	// only meaningful when the call itself reports failure.
	if waited, _, _ := procWaitForSingle.Call(info.hProcess, infinite); waited == 0xFFFFFFFF {
		return true, fmt.Errorf("setup ran elevated, but waiting for it failed")
	}
	var code uint32
	if ok, _, callErr := procGetExitCode.Call(info.hProcess, uintptr(unsafe.Pointer(&code))); ok == 0 {
		return true, fmt.Errorf("setup ran elevated, but its result is unknown: %v", callErr)
	}
	return true, childExitError(code)
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
