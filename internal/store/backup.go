package store

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Path is the SQLite file this store opened.
func (s *Store) Path() string { return s.path }

// SchemaVersion is the highest applied migration.
func (s *Store) SchemaVersion() (int, error) {
	var v int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("store: schema version: %w", err)
	}
	return v, nil
}

// VacuumInto writes a consistent copy of the database to path (must not exist).
func (s *Store) VacuumInto(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("store: vacuum path must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("store: vacuum dir: %w", err)
	}
	esc := strings.ReplaceAll(path, "'", "''")
	if _, err := s.db.Exec("VACUUM INTO '" + esc + "'"); err != nil {
		return fmt.Errorf("store: vacuum into: %w", err)
	}
	return nil
}

// ReplaceFrom closes the live DB, swaps in src, and reopens (runs migrations).
func (s *Store) ReplaceFrom(src string) error {
	if s.path == "" {
		return fmt.Errorf("store: replace: no path")
	}
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("store: replace close: %w", err)
	}
	bak := s.path + ".prerestore"
	_ = os.Remove(bak)
	if err := os.Rename(s.path, bak); err != nil && !os.IsNotExist(err) {
		_ = s.reopen()
		return fmt.Errorf("store: replace stash: %w", err)
	}
	_ = os.Remove(s.path + "-wal")
	_ = os.Remove(s.path + "-shm")
	if err := copyFile(src, s.path); err != nil {
		_ = os.Rename(bak, s.path)
		_ = s.reopen()
		return fmt.Errorf("store: replace copy: %w", err)
	}
	if err := s.reopen(); err != nil {
		_ = os.Rename(s.path, s.path+".bad")
		_ = os.Rename(bak, s.path)
		if rerr := s.reopen(); rerr != nil {
			return fmt.Errorf("store: replace reopen: %v (rollback: %w)", err, rerr)
		}
		return fmt.Errorf("store: replace reopen: %w", err)
	}
	_ = os.Remove(bak)
	return nil
}

func (s *Store) reopen() error {
	db, err := sqlOpen(s.path)
	if err != nil {
		return err
	}
	s.db = db
	return s.migrate()
}

func sqlOpen(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
