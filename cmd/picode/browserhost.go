package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/browserhost"
)

func runBrowserHost() {
	if err := browserhost.Serve(os.Stdin, os.Stdout, browserhost.NewClient().Handle); err != nil {
		fmt.Fprintf(os.Stderr, "picode browser-host: %v\n", err)
		os.Exit(1)
	}
}

func runExtensionInstall(args []string) {
	fs := flag.NewFlagSet("extension-install", flag.ExitOnError)
	server := fs.String("server", "", "PiCode on another machine, e.g. https://box.tailxxxx.ts.net:8445 (ADR-0050)")
	token := fs.String("token", "", "that server's install token (picode token, on the server); prompted when --server is set and this is empty")
	ca := fs.String("ca", "", "PEM file to trust for that server (mkcert rootCA.pem copied from it); optional")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("extension-install: %v", err)
	}
	if *server != "" {
		tok := strings.TrimSpace(*token)
		if tok == "" {
			fmt.Print("Install token for " + *server + " (echoes): ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			tok = strings.TrimSpace(line)
		}
		if tok == "" {
			log.Fatalf("extension-install: a remote server needs its install token")
		}
		if *ca != "" {
			if abs, err := filepath.Abs(*ca); err == nil {
				*ca = abs
			}
		}
		path, err := browserhost.WriteRemote(browserhost.Remote{URL: *server, Token: tok, CAFile: *ca})
		if err != nil {
			log.Fatalf("extension-install: %v", err)
		}
		fmt.Println("Remote PiCode recorded in " + path)
		fmt.Println("  the extension and scripts on this machine will talk to " + strings.TrimRight(*server, "/"))
	}
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
