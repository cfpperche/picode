package climodels

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Muse Code has no listing verb; its documented MSP schema has `model/list`
// ("a query, not a command … a client may list before it has a session"),
// served by `muse serve` as newline-delimited JSON-RPC on stdio. Measured
// 2026-09-23 against muse 1.3.0-R3401.1:
//
//   - initialize → initialized → model/list answers in ~0.08 s, offline,
//     from ~/.local/share/muse/model-catalog/*; a home without that cache
//     answers an empty list (source bundledCatalog).
//   - Its launcher self-updates at most hourly: MUSE_NO_AUTO_UPDATE=1 keeps a
//     read from ever installing anything. `--no-session-log` keeps it from
//     writing a session; one trace log per run remains.
//   - A guessed project settings file did not change the answer (weak
//     evidence: the format is not confirmed), so the folder stays an input.
//   - The selector is the model id; Muse keeps the provider in its own key.
func museInputs(command, dir string) []string {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home(), ".local", "share")
	}
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home(), ".config")
	}
	files, _ := filepath.Glob(filepath.Join(data, "muse", "model-catalog", "*.json"))
	files = append(files,
		filepath.Join(config, "muse", "settings.json"),
		filepath.Join(config, "muse", "auth.json"),
	)
	if bin, err := exec.LookPath(command); err == nil {
		files = append(files, bin)
	}
	if dir != "" {
		files = append(files, filepath.Join(dir, ".muse", "settings.json"))
	}
	return files
}

func probeMuse(ctx context.Context, command, dir string) (Report, error) {
	cmd := exec.CommandContext(ctx, command, "serve", "--no-session-log")
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "MUSE_NO_AUTO_UPDATE=1")
	cmd.WaitDelay = 2 * time.Second
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Report{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Report{}, err
	}
	if err := cmd.Start(); err != nil {
		return Report{}, fmt.Errorf("muse serve: %s", bounded(err.Error(), 400))
	}
	defer func() {
		_ = stdin.Close()
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}()
	enc := json.NewEncoder(stdin)
	lines := bufio.NewScanner(stdout)
	lines.Buffer(make([]byte, 0, 64<<10), 8<<20)
	// await reads until the response with id; notifications and other ids
	// (the server may speak first) are skipped.
	await := func(id int) (json.RawMessage, error) {
		for lines.Scan() {
			var msg struct {
				ID     *int            `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if json.Unmarshal(lines.Bytes(), &msg) != nil || msg.ID == nil || *msg.ID != id {
				continue
			}
			if msg.Error != nil {
				return nil, fmt.Errorf("muse: %s", bounded(msg.Error.Message, 400))
			}
			return msg.Result, nil
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("muse: it did not answer in time")
		}
		return nil, fmt.Errorf("muse serve closed before answering")
	}
	send := func(v any) error { return enc.Encode(v) }
	if err := send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"clientInfo": map[string]string{"name": "picode", "version": "1"}}}); err != nil {
		return Report{}, err
	}
	if _, err := await(1); err != nil {
		return Report{}, err
	}
	if err := send(map[string]any{"jsonrpc": "2.0", "method": "initialized"}); err != nil {
		return Report{}, err
	}
	if err := send(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "model/list", "params": map[string]any{}}); err != nil {
		return Report{}, err
	}
	raw, err := await(2)
	if err != nil {
		return Report{}, err
	}
	models, err := parseMuse(raw)
	if err != nil {
		return Report{}, err
	}
	return Report{CLI: "muse", Dir: dir, Models: models, Kinds: []string{"chat"}}, nil
}

func parseMuse(raw json.RawMessage) ([]Model, error) {
	var body struct {
		Models []struct {
			ProviderID   string `json:"providerId"`
			ModelID      string `json:"modelId"`
			DisplayLabel string `json:"displayLabel"`
			ContextLimit int    `json:"contextLimit"`
			OutputLimit  int    `json:"outputLimit"`
			Cost         *struct {
				Input    string `json:"input"`
				Output   string `json:"output"`
				Currency string `json:"currency"`
			} `json:"cost"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("muse model/list answered something PiCode could not read: %w", err)
	}
	rows := []Model{}
	for _, m := range body.Models {
		if m.ModelID == "" {
			continue
		}
		row := Model{Provider: m.ProviderID, ID: m.ModelID, Name: m.DisplayLabel, Kind: "chat", Selector: m.ModelID,
			Context: m.ContextLimit, MaxOut: m.OutputLimit}
		// Prices are USD per million tokens as strings; another currency is
		// not shown as if it were dollars.
		if m.Cost != nil && (m.Cost.Currency == "" || strings.EqualFold(m.Cost.Currency, "USD")) {
			in, e1 := strconv.ParseFloat(m.Cost.Input, 64)
			out, e2 := strconv.ParseFloat(m.Cost.Output, 64)
			if e1 == nil && e2 == nil {
				row.Cost = &Cost{Input: in, Output: out}
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
