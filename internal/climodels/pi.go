package climodels

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Pi's catalog is `pi --list-models --offline`, a table (ADR-0009). Measured
// 2026-09-23 against pi 0.87.1, on a copy of the owner's home:
//
//   - The answer is decided by auth.json (only signed-in providers are
//     listed: 527 rows with it, 3 without), models.json (custom providers),
//     models-store.json and the installed packages (the owner's four add a
//     provider: 537 rows against 527 on a home without them). A settings
//     `enabledModels`, global or in a project's .pi/settings.json, did not
//     change it. settings.json stays in the set because it names the packages.
//   - Listing rewrites none of those files (sizes and times unchanged), so
//     unlike omp the credential store can be an input.
//   - The command itself takes 0.2–0.5 s; the 2 s the catalog used to cost was
//     its llama.cpp probe, not Pi.
//
// The binary is an input too: an upgrade may list other models.
func piInputs(command, dir string) []string {
	agent := filepath.Join(home(), ".pi", "agent")
	if v := os.Getenv("PI_CODING_AGENT_DIR"); v != "" {
		agent = v
	}
	files := []string{
		filepath.Join(agent, "auth.json"),
		filepath.Join(agent, "models.json"),
		filepath.Join(agent, "models-store.json"),
		filepath.Join(agent, "settings.json"),
		filepath.Join(agent, "npm", "package-lock.json"),
	}
	if bin, err := exec.LookPath(command); err == nil {
		files = append(files, bin)
	}
	if dir != "" {
		files = append(files, filepath.Join(dir, ".pi", "settings.json"))
	}
	return files
}

func probePi(ctx context.Context, command, dir string) (Report, error) {
	cmd := exec.CommandContext(ctx, command, "--list-models", "--offline")
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if ctx.Err() != nil {
			msg = "it did not answer in time"
		}
		return Report{}, fmt.Errorf("pi --list-models: %s", bounded(msg, 400))
	}
	return Report{CLI: "pi", Dir: dir, Models: ParsePiTable(stdout.String()), Kinds: []string{"chat"}}, nil
}

// ParsePiTable turns the table `pi --list-models` prints into rows:
// provider | model… | context | max-out | thinking | images.
func ParsePiTable(text string) []Model {
	rows := []Model{}
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "provider" {
			continue
		}
		images := fields[len(fields)-1]
		thinking := fields[len(fields)-2]
		// The CLI also prints prose when no models are available. A table
		// row always ends in two yes/no capability columns; prose is not a
		// provider or a model, even when it contains six or more words.
		if (images != "yes" && images != "no") || (thinking != "yes" && thinking != "no") {
			continue
		}
		provider := fields[0]
		model := strings.Join(fields[1:len(fields)-4], " ")
		ctxLabel, outLabel := fields[len(fields)-4], fields[len(fields)-3]
		m := Model{
			Provider: provider, ID: model, Kind: "chat", Selector: provider + "/" + model,
			Context: sizeOf(ctxLabel), ContextLabel: ctxLabel,
			MaxOut: sizeOf(outLabel), MaxOutLabel: outLabel,
			Reasoning: thinking == "yes",
		}
		if images == "yes" {
			m.Input = []string{"image"}
		}
		rows = append(rows, m)
	}
	return rows
}

// sizeOf reads Pi's "200K" / "1.0M" labels as token counts; 0 when unreadable.
func sizeOf(label string) int {
	mult := 1.0
	switch {
	case strings.HasSuffix(label, "M"):
		mult, label = 1_000_000, strings.TrimSuffix(label, "M")
	case strings.HasSuffix(label, "K"):
		mult, label = 1_000, strings.TrimSuffix(label, "K")
	}
	f, err := strconv.ParseFloat(label, 64)
	if err != nil || f < 0 {
		return 0
	}
	return int(f*mult + 0.5)
}
