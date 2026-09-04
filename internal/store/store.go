// Package store is PiCode's persistence layer: SQLite (pure Go driver)
// holding the orchestration overlay only — canonical agent data stays in
// pi's own files (ADR-0005).
package store

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO; single-binary)
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps the SQLite database. Safe for concurrent use (database/sql
// pools connections; pragmas are set per connection via the DSN).
type Store struct {
	db   *sql.DB
	path string

	// OnEvent fires after every committed event (ADR-0048: the change
	// feed listens; push and the UI consume the feed). Called
	// synchronously on the writer's goroutine; the listener must return
	// fast. Optional.
	OnEvent func(Event)

	pendMu  sync.Mutex
	pending map[*sql.Tx][]Event // events appended in an open tx, announced on commit
}

// Open creates/opens the database at path, applies pragmas and migrations,
// and (once) imports the legacy M1 JSON registry if present next to it.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	db, err := sqlOpen(path)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db, path: path}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.importLegacy(filepath.Join(filepath.Dir(path), "workspaces.json")); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := s.recoverPendingInboxReplies(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: recover interrupted reply bursts: %w", err)
	}
	return s, nil
}

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("store: migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("store: read migrations: %w", err)
	}

	for _, e := range entries {
		name := e.Name()
		var version int
		if _, err := fmt.Sscanf(name, "%d_", &version); err != nil {
			return fmt.Errorf("store: migration %q: bad name", name)
		}
		var applied bool
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version).Scan(&applied); err != nil {
			return fmt.Errorf("store: check migration %d: %w", version, err)
		}
		if applied {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("store: read migration %q: %w", name, err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("store: tx migration %d: %w", version, err)
		}
		if _, err := tx.Exec(string(body)); err != nil {
			s.rollback(tx)
			return fmt.Errorf("store: apply migration %d: %w", version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, version, nowUTC()); err != nil {
			s.rollback(tx)
			return fmt.Errorf("store: record migration %d: %w", version, err)
		}
		if err := s.commit(tx); err != nil {
			return fmt.Errorf("store: commit migration %d: %w", version, err)
		}
	}
	return nil
}

// importLegacy performs a one-time migration from the M1 JSON registry
// (workspaces.json next to the db). On success the file is renamed to
// workspaces.json.migrated so it never imports twice.
func (s *Store) importLegacy(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("store: legacy registry: %w", err)
	}
	var legacy []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Path      string `json:"path"`
		CreatedAt string `json:"createdAt"`
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		// Empty file: just retire it.
		return os.Rename(path, path+".migrated")
	}
	if err := unmarshalJSON(data, &legacy); err != nil {
		return fmt.Errorf("store: legacy registry parse: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { s.rollback(tx) }()
	for _, w := range legacy {
		created := w.CreatedAt
		if created == "" {
			created = nowUTC()
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO workspaces (id, name, path, created_at) VALUES (?, ?, ?, ?)`,
			w.ID, w.Name, w.Path, created); err != nil {
			return fmt.Errorf("store: import workspace %q: %w", w.ID, err)
		}
		if _, err := ensureDefaultAgentTx(tx, w.ID, w.Name, created); err != nil {
			return err
		}
	}
	if err := s.commit(tx); err != nil {
		return err
	}
	return os.Rename(path, path+".migrated")
}

// ---------- shared helpers ----------

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339Nano) }

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

// newID builds a readable, collision-unlikely id: slug of name + hex suffix.
func newID(name, prefix string) string {
	slug := strings.Trim(slugPattern.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		slug = prefix
	}
	if len(slug) > 32 {
		slug = slug[:32]
	}
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(b))
}
