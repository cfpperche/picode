// Package credentials owns PiCode's credential vault: one file holding every
// provider account this machine has, for Pi and for the guest agent CLIs
// (ADR-0165). It replaces the plaintext `~/.picode/accounts.json` of ADR-0013
// — the old file is read once and never written again.
//
// The file is encrypted at rest with AES-256-GCM under a random key that
// lives beside it (`credentials.key`, 0600). That key is not a secret from
// the person using the machine — it sits next to the data on purpose, so a
// terminal can be opened without typing a passphrase. What the encryption
// buys is that a copy of the vault alone is useless: a backup snapshot
// uploaded somewhere, a dotfiles repository, a support bundle, a file another
// account can read. It does not stop a program already running as this user,
// which can read the key too. Saying more than that would be a lie the first
// time someone copies the file.
//
// Nothing here ever returns secret material the caller did not ask for by
// name: `Row` carries the credential only because Pi's `auth.json` slot and a
// CLI's own file are written from it. Callers that answer the browser strip it
// (see internal/catalog and internal/server).
package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	fileName = "credentials.json"
	keyName  = "credentials.key"
	// Version is the plaintext schema. 2 added origin, hint and health to the
	// ADR-0013 shape; an unreadable file is never overwritten.
	Version = 2
	alg     = "AES-256-GCM"
)

// ErrLocked is returned when the vault exists but its key does not: the rows
// are there and unreadable, which is a state to report, never to paper over by
// minting a new key (that would orphan every stored account).
var ErrLocked = errors.New("credentials: the vault key is missing — secrets are unavailable on this machine")

// ErrCorrupt is returned when the vault cannot be decrypted: the wrong key, a
// truncated file, a hand edit.
var ErrCorrupt = errors.New("credentials: the vault cannot be read — restore a backup or the key file that matches it")

// Store is the vault on disk. All methods are safe for concurrent use; the
// daemon is its only writer.
type Store struct {
	dir string
	mu  sync.Mutex
}

// New returns a store rooted at dir (the data directory: `~/.picode`).
func New(dir string) *Store { return &Store{dir: dir} }

var (
	storesMu sync.Mutex
	stores   = map[string]*Store{}
)

// For returns the store rooted at dir, shared process-wide so two callers
// cannot interleave a read-modify-write of the same file.
func For(dir string) *Store {
	storesMu.Lock()
	defer storesMu.Unlock()
	if s, ok := stores[dir]; ok {
		return s
	}
	s := New(dir)
	stores[dir] = s
	return s
}

// Default is the store for this process: $PICODE_DATA, else ~/.picode — the
// same resolution internal/config uses for the data directory. The directory
// is resolved on every call, never cached: a test that points HOME somewhere
// else must get that vault, and the daemon's HOME never changes.
func Default() *Store { return For(defaultDir()) }

func defaultDir() string {
	if dir := os.Getenv("PICODE_DATA"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".picode")
}

// Path is the vault file this store reads and writes.
func (s *Store) Path() string { return filepath.Join(s.dir, fileName) }

func (s *Store) keyPath() string { return filepath.Join(s.dir, keyName) }

// Load returns the vault. A missing file is an empty vault; a missing key
// beside an existing file is ErrLocked; anything that will not decrypt is
// ErrCorrupt. The legacy ADR-0013 file is absorbed on the first load.
func (s *Store) Load() (File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) loadLocked() (File, error) {
	f, err := s.readFile()
	if err != nil {
		return File{}, err
	}
	if len(f.Providers) == 0 {
		if legacy, ok := s.readLegacy(); ok {
			f = legacy
			if err := s.saveLocked(f); err != nil {
				return f, err
			}
		}
	}
	return f, nil
}

func (s *Store) readFile() (File, error) {
	raw, err := os.ReadFile(s.Path())
	if errors.Is(err, os.ErrNotExist) {
		return newFile(), nil
	}
	if err != nil {
		return File{}, err
	}
	plain, err := s.open(raw)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(plain, &f); err != nil {
		return File{}, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	if f.Providers == nil {
		f.Providers = map[string]Slot{}
	}
	if f.Version == 0 {
		f.Version = Version
	}
	return f, nil
}

// Update runs fn against the vault and writes the result, under one lock —
// the only way a mutation is allowed to happen, so two writers cannot
// interleave a read-modify-write.
func (s *Store) Update(fn func(*File) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadLocked()
	if err != nil {
		return err
	}
	before, _ := json.Marshal(f)
	if err := fn(&f); err != nil {
		return err
	}
	after, _ := json.Marshal(f)
	if string(before) == string(after) {
		return nil
	}
	if f.Version == 0 {
		f.Version = Version
	}
	return s.saveLocked(f)
}

// saveLocked encrypts and replaces the file atomically. A partial write can
// never be observed: temp file in the same directory, then rename.
func (s *Store) saveLocked(f File) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	plain, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	sealed, err := s.seal(plain)
	if err != nil {
		return err
	}
	return writeAtomic(s.Path(), sealed, 0o600)
}

type envelope struct {
	Version int    `json:"version"`
	Alg     string `json:"alg"`
	Nonce   string `json:"nonce"`
	Data    string `json:"data"`
}

func (s *Store) seal(plain []byte) ([]byte, error) {
	key, err := s.loadKey(true)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	env := envelope{
		Version: Version,
		Alg:     alg,
		Nonce:   base64.StdEncoding.EncodeToString(nonce),
		Data:    base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, nil)),
	}
	out, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func (s *Store) open(raw []byte) ([]byte, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Alg != alg {
		return nil, ErrCorrupt
	}
	key, err := s.loadKey(false)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, ErrCorrupt
	}
	data, err := base64.StdEncoding.DecodeString(env.Data)
	if err != nil {
		return nil, ErrCorrupt
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, ErrCorrupt
	}
	return plain, nil
}

// loadKey reads the vault key; create mints one on first write. A key that is
// not 32 bytes is refused rather than replaced (the vault it opens would be
// lost).
func (s *Store) loadKey(create bool) ([]byte, error) {
	raw, err := os.ReadFile(s.keyPath())
	if err == nil {
		if len(raw) != 32 {
			return nil, fmt.Errorf("%w: %s is not a vault key", ErrCorrupt, s.keyPath())
		}
		return raw, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if !create {
		return nil, ErrLocked
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(s.keyPath(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Another writer won the race: its key is the vault's key.
		if errors.Is(err, os.ErrExist) {
			return s.loadKey(false)
		}
		return nil, err
	}
	defer f.Close()
	if _, err := f.Write(key); err != nil {
		return nil, err
	}
	return key, nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".credentials-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() {
		if name != "" {
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	name = ""
	return nil
}
