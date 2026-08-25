package llama

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const Releases = "https://github.com/ggml-org/llama.cpp/releases"

// Setup is what the UI needs to show the next config step. No secrets.
type Setup struct {
	Binary    string `json:"binary"`
	URL       string `json:"url"`
	Reachable bool   `json:"reachable"`
	Models    int    `json:"models"`
	Loaded    int    `json:"loaded"`
	HFToken   bool   `json:"hfToken"`
	ModelsDir string `json:"modelsDir"`
	StartCmd  string `json:"startCmd"`
	Releases  string `json:"releases"`
}

func modelsDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return "llama-models"
	}
	return filepath.Join(home, ".picode", "llama-models")
}

func findBinary() string {
	if p := bundledBinary(); p != "" {
		return p
	}
	for _, n := range []string{"llama-server", "llama-server.exe"} {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}

func startArgs(bin, dir, rawURL string) []string {
	host, port := "127.0.0.1", "8080"
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		h, p, ok := strings.Cut(u.Host, ":")
		if ok {
			host, port = h, p
		} else if u.Hostname() != "" {
			host = u.Hostname()
		}
		if u.Port() != "" {
			port = u.Port()
		}
	}
	if bin == "" {
		bin = "llama-server"
	}
	return []string{
		bin,
		"--models-dir", dir,
		"--no-models-autoload",
		"--jinja",
		"--host", host,
		"--port", port,
	}
}

// Inspect reports install/router/model state.
func Inspect(serverURL string, models []Model, reachable bool) Setup {
	dir := modelsDir()
	bin := findBinary()
	s := Setup{
		Binary:    bin,
		URL:       serverURL,
		Reachable: reachable,
		HFToken:   hfToken() != "",
		ModelsDir: dir,
		StartCmd:  strings.Join(quote(startArgs(bin, dir, serverURL)), " "),
		Releases:  Releases,
	}
	for _, m := range models {
		s.Models++
		if m.Status == "loaded" || m.Status == "sleeping" {
			s.Loaded++
		}
	}
	return s
}

func quote(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			out[i] = strconv.Quote(a)
		} else {
			out[i] = a
		}
	}
	return out
}

// StartRouter launches llama-server in the background if the binary is on PATH.
func StartRouter(serverURL string) error {
	bin := findBinary()
	if bin == "" {
		return fmt.Errorf("llama-server not on PATH")
	}
	dir := modelsDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	args := startArgs(bin, dir, serverURL)[1:]
	cmd := exec.Command(bin, args...)
	logPath := filepath.Join(filepath.Dir(dir), "llama-server.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	cmd.Stdout, cmd.Stderr = f, f
	if err := cmd.Start(); err != nil {
		_ = f.Close()
		return err
	}
	go func() {
		_ = cmd.Wait()
		_ = f.Close()
	}()
	time.Sleep(400 * time.Millisecond)
	return nil
}
