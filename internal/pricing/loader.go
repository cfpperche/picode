package pricing

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultURL is LiteLLM's published table, the one ccusage and t3code read.
const DefaultURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// URLFromEnv is where the table comes from: PICODE_PRICE_TABLE_URL when set,
// "" when it says "off" (no fetch, no estimates beyond a copy already on
// disk), DefaultURL otherwise.
func URLFromEnv() string {
	v := strings.TrimSpace(os.Getenv("PICODE_PRICE_TABLE_URL"))
	switch strings.ToLower(v) {
	case "":
		return DefaultURL
	case "off", "0", "false", "none":
		return ""
	}
	return v
}

// Loader keeps Current fresh: the last good copy from disk at boot, then a
// fetch now and every Every. A failed fetch keeps whatever table is loaded
// — an old price is still a price, and the version names it.
type Loader struct {
	Path  string // <data dir>/var/litellm-prices.json
	URL   string // "" disables fetching
	HTTP  *http.Client
	Every time.Duration
}

// maxTableBytes bounds the download; the real file is ~3 MB.
const maxTableBytes = 64 << 20

// LoadCached installs the copy on disk, if there is one. It never reaches
// the network, so boot is not slowed by it.
func (l Loader) LoadCached() error {
	raw, err := os.ReadFile(l.Path)
	if err != nil {
		return err
	}
	t, err := Parse(raw)
	if err != nil {
		return err
	}
	Set(t)
	return nil
}

// Run loads the cached copy, then fetches until ctx ends.
func (l Loader) Run(ctx context.Context) {
	if err := l.LoadCached(); err != nil && !os.IsNotExist(err) {
		log.Printf("pricing: cached table unreadable: %v", err)
	}
	if l.URL == "" {
		return
	}
	every := l.Every
	if every <= 0 {
		every = 24 * time.Hour
	}
	for {
		if err := l.Refresh(ctx); err != nil && ctx.Err() == nil {
			log.Printf("pricing: refresh failed, keeping the table in use: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}

// Refresh fetches the table once, installs it, and writes it to disk
// atomically. Nothing is replaced unless the new table parses and prices
// at least one model.
func (l Loader) Refresh(ctx context.Context) error {
	client := l.HTTP
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.URL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", l.URL, resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxTableBytes))
	if err != nil {
		return err
	}
	t, err := Parse(raw)
	if err != nil {
		return err
	}
	if t.Len() == 0 {
		return fmt.Errorf("GET %s: no priced model in the table", l.URL)
	}
	Set(t)
	if l.Path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	tmp := l.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, l.Path)
}
