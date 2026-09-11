//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// The tray has no window of its own to hang a page off, and fyne.io/systray
// v1.12 removed its balloon API, so a decision this heavy — stopping the
// distro — is asked with the one dialog Windows gives every process for free.
// It blocks the calling goroutine, which is exactly what the flow wants: no
// answer, no compact.

var user32 = syscall.NewLazyDLL("user32.dll")
var procMessageBoxW = user32.NewProc("MessageBoxW")

const (
	mbOK            = 0x00000000
	mbYesNo         = 0x00000004
	mbIconWarning   = 0x00000030
	mbSetForeground = 0x00010000
	mbTopmost       = 0x00040000

	idOK  = 1
	idYes = 6
)

func messageBox(text, caption string, flags uintptr) int {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(caption)
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(t)),
		uintptr(unsafe.Pointer(c)),
		flags,
	)
	return int(ret)
}

// confirm asks a yes/no question and means it: the flow behind it stops the
// distro and every session in it.
func confirm(text, caption string) bool {
	return messageBox(text, caption, mbYesNo|mbIconWarning|mbSetForeground|mbTopmost) == idYes
}

// alert reports an outcome. Nothing behind it is waiting on the answer.
func alert(text, caption string) {
	messageBox(text, caption, mbOK|mbIconWarning|mbSetForeground|mbTopmost)
}
