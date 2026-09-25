package climodels

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Antigravity's catalog is `agy models`: one `id<TAB>display name` line per
// model, fetched from the network with the signed-in account. Measured
// 2026-09-25 against agy 1.2.11:
//
//   - Signed out it refuses ("Please sign in to view available models").
//   - Run against the owner's home it refreshed their OAuth token file
//     (2026-09-23). So the probe runs with HOME set to a throwaway folder
//     holding a copy of that one file: the refresh lands in the copy, and
//     the owner's file is never written (same size, mtime and hash after the
//     run). The refresh changes the access token, its expiry and the id
//     token; the refresh token stays the same, so the owner's sign-in is not
//     invalidated by it.
//   - ~4 s (network each time).
//   - The selector is the display name: Antigravity's `model` setting holds
//     it ("Gemini 3.8 Flash (Medium)"), as the CLI itself writes it.
func agyTokenFile() string {
	return filepath.Join(home(), ".gemini", "antigravity-cli", "antigravity-oauth-token")
}

func agyInputs(command, _ string) []string {
	files := []string{agyTokenFile()}
	if bin, err := exec.LookPath(command); err == nil {
		files = append(files, bin)
	}
	return files
}

func probeAgy(ctx context.Context, command, dir string) (Report, error) {
	token, err := os.ReadFile(agyTokenFile())
	if err != nil {
		return Report{}, &Blocked{Msg: "Antigravity is not signed in — sign in to see its models", Action: ActionSignIn}
	}
	scratch, err := os.MkdirTemp("", "picode-agy-models-")
	if err != nil {
		return Report{}, fmt.Errorf("agy models: %w", err)
	}
	defer os.RemoveAll(scratch)
	root := filepath.Join(scratch, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return Report{}, fmt.Errorf("agy models: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, "antigravity-oauth-token"), token, 0o600); err != nil {
		return Report{}, fmt.Errorf("agy models: %w", err)
	}
	// exec keeps the last of duplicate keys, so this HOME wins.
	out, err := runQuiet(ctx, scratch, []string{"HOME=" + scratch}, command, "models")
	if err != nil {
		// A sign-in that no longer holds reads like none at all to agy.
		if strings.Contains(strings.ToLower(err.Error()), "sign in") {
			return Report{}, &Blocked{Msg: "Antigravity's sign-in has expired — sign in again to see its models", Action: ActionSignIn}
		}
		return Report{}, fmt.Errorf("agy models: %s", err)
	}
	models := parseAgy(out)
	return Report{CLI: "agy", Dir: dir, Models: models, Kinds: []string{"chat"}}, nil
}

func parseAgy(raw []byte) []Model {
	rows := []Model{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		id, name, ok := strings.Cut(sc.Text(), "\t")
		id, name = strings.TrimSpace(id), strings.TrimSpace(name)
		if !ok || id == "" || name == "" {
			continue
		}
		rows = append(rows, Model{Provider: "antigravity", ID: id, Name: name, Kind: "chat", Selector: name})
	}
	return rows
}
