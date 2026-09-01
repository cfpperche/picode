package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/browserhost"
)

func runBrowserHost() {
	if err := browserhost.Serve(os.Stdin, os.Stdout, browserhost.NewClient().Handle); err != nil {
		fmt.Fprintf(os.Stderr, "picode browser-host: %v\n", err)
		os.Exit(1)
	}
}

func runExtensionInstall() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("extension-install: %v", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		log.Fatalf("extension-install: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("extension-install: %v", err)
	}
	path, err := browserhost.WriteHostManifest(home, exe)
	if err != nil {
		log.Fatalf("extension-install: %v", err)
	}
	fmt.Println("Chrome native host installed:")
	fmt.Println("  " + path)
	fmt.Println("Load the unpacked extension from ext/ in this repo.")
	fmt.Println("Then chrome://extensions → PiCode → details → side panel.")
	if _, err := os.Stat("/mnt/c/Windows"); err == nil {
		fmt.Println()
		fmt.Println("Chrome on Windows needs picode-desktop extension-install —")
		fmt.Println("this file is only visible to Chrome running inside Linux.")
	}
}

func runExtensionUninstall() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("extension-uninstall: %v", err)
	}
	if err := browserhost.RemoveHostManifest(home); err != nil {
		log.Fatalf("extension-uninstall: %v", err)
	}
	fmt.Println("Chrome native host removed.")
}
