package main

import _ "embed"

// trayIcon is the notification-area icon: a white pi on PiCode's indigo, at
// the six sizes Windows picks between for a tray. It is generated rather than
// drawn by hand — `go generate ./cmd/picode-desktop` rewrites it from
// mkicon.go, so the asset is reproducible instead of an opaque blob.
//
//go:generate go run mkicon.go icon.ico
//go:embed icon.ico
var trayIcon []byte
