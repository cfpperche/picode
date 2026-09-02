package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	MaxPinFiles     = 24
	MaxPinImageSize = 8 << 20
	MaxPinFileSize  = 16 << 20
	MaxPinSceneSize = 2 << 20
)

// PinFile is metadata for a pin attachment. Bytes live on disk.
type PinFile struct {
	ID         string `json:"id"`
	PinID      string `json:"pinId"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Mime       string `json:"mime"`
	Size       int64  `json:"size"`
	CreatedAt  string `json:"createdAt"`
	Source     string `json:"source,omitempty"`
	BaseFileID string `json:"baseFileId,omitempty"`
}

func ClassifyPinFile(name, mime string, size int64) (kind string, err error) {
	mime = strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".exe", ".bat", ".cmd", ".com", ".scr", ".msi", ".dll":
		return "", fmt.Errorf("that file type is not allowed")
	}
	switch mime {
	case "image/png", "image/jpeg", "image/jpg", "image/webp", "image/gif":
		if size > MaxPinImageSize {
			return "", fmt.Errorf("image too large (max 8 MB)")
		}
		return "image", nil
	}
	if size > MaxPinFileSize {
		return "", fmt.Errorf("file too large (max 16 MB)")
	}
	if mime == "" || mime == "application/octet-stream" {
		mime = "application/octet-stream"
	}
	return "file", nil
}

func (s *Store) ListPinFiles(pinID string) ([]PinFile, error) {
	rows, err := s.db.Query(`SELECT id, pin_id, kind, name, mime, size, created_at, source, base_file_id FROM pin_files WHERE pin_id = ? ORDER BY created_at`, pinID)
	if err != nil {
		return nil, fmt.Errorf("store: list pin files: %w", err)
	}
	defer rows.Close()
	out := []PinFile{}
	for rows.Next() {
		var f PinFile
		if err := rows.Scan(&f.ID, &f.PinID, &f.Kind, &f.Name, &f.Mime, &f.Size, &f.CreatedAt, &f.Source, &f.BaseFileID); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) GetPinFile(pinID, id string) (PinFile, error) {
	var f PinFile
	err := s.db.QueryRow(`SELECT id, pin_id, kind, name, mime, size, created_at, source, base_file_id FROM pin_files WHERE id = ? AND pin_id = ?`, id, pinID).
		Scan(&f.ID, &f.PinID, &f.Kind, &f.Name, &f.Mime, &f.Size, &f.CreatedAt, &f.Source, &f.BaseFileID)
	if err == sql.ErrNoRows {
		return PinFile{}, ErrNotFound
	}
	if err != nil {
		return PinFile{}, fmt.Errorf("store: get pin file: %w", err)
	}
	return f, nil
}

func (s *Store) AddPinFile(pinID, name, mime string, size int64) (PinFile, error) {
	if _, err := s.GetPin(pinID); err != nil {
		return PinFile{}, err
	}
	n, err := s.countPinFiles(pinID)
	if err != nil {
		return PinFile{}, err
	}
	if n >= MaxPinFiles {
		return PinFile{}, fmt.Errorf("a pin can have at most %d files", MaxPinFiles)
	}
	kind, err := ClassifyPinFile(name, mime, size)
	if err != nil {
		return PinFile{}, err
	}
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == ".." {
		name = "file"
	}
	if len(name) > 200 {
		name = name[:200]
	}
	now := nowUTC()
	f := PinFile{ID: newID(name, "file"), PinID: pinID, Kind: kind, Name: name, Mime: mime, Size: size, CreatedAt: now}
	if f.Mime == "" {
		f.Mime = "application/octet-stream"
	}
	if _, err := s.db.Exec(`INSERT INTO pin_files (id, pin_id, kind, name, mime, size, created_at, source, base_file_id) VALUES (?, ?, ?, ?, ?, ?, ?, '', '')`,
		f.ID, f.PinID, f.Kind, f.Name, f.Mime, f.Size, f.CreatedAt); err != nil {
		return PinFile{}, fmt.Errorf("store: add pin file: %w", err)
	}
	s.note("pin.updated", nil, nil, idData(pinID))
	return f, nil
}

func (s *Store) DeletePinFile(pinID, id string) error {
	res, err := s.db.Exec(`DELETE FROM pin_files WHERE id = ? AND pin_id = ?`, id, pinID)
	if err != nil {
		return fmt.Errorf("store: delete pin file: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	s.note("pin.updated", nil, nil, idData(pinID))
	return nil
}

func (s *Store) countPinFiles(pinID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM pin_files WHERE pin_id = ?`, pinID).Scan(&n)
	return n, err
}

func cleanPinFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == ".." {
		return "sketch"
	}
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

func (s *Store) AddPinSketch(pinID, name, source, baseID string, previewSize int64) (PinFile, error) {
	if _, err := s.GetPin(pinID); err != nil {
		return PinFile{}, err
	}
	n, err := s.countPinFiles(pinID)
	if err != nil {
		return PinFile{}, err
	}
	if n >= MaxPinFiles {
		return PinFile{}, fmt.Errorf("a pin can have at most %d files", MaxPinFiles)
	}
	if previewSize > MaxPinImageSize {
		return PinFile{}, fmt.Errorf("image too large (max 8 MB)")
	}
	if source != "annotate" {
		source = "blank"
	}
	now := nowUTC()
	f := PinFile{
		ID: newID("sketch", "sketch"), PinID: pinID, Kind: "sketch",
		Name: cleanPinFileName(name), Mime: "image/png", Size: previewSize,
		CreatedAt: now, Source: source, BaseFileID: strings.TrimSpace(baseID),
	}
	if f.Name == "file" || f.Name == "sketch" {
		f.Name = "Sketch"
	}
	if _, err := s.db.Exec(`INSERT INTO pin_files (id, pin_id, kind, name, mime, size, created_at, source, base_file_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.PinID, f.Kind, f.Name, f.Mime, f.Size, f.CreatedAt, f.Source, f.BaseFileID); err != nil {
		return PinFile{}, fmt.Errorf("store: add sketch: %w", err)
	}
	s.note("pin.updated", nil, nil, idData(pinID))
	return f, nil
}

func (s *Store) UpdatePinSketch(pinID, id, name string, previewSize int64) (PinFile, error) {
	f, err := s.GetPinFile(pinID, id)
	if err != nil {
		return PinFile{}, err
	}
	if f.Kind != "sketch" {
		return PinFile{}, fmt.Errorf("not a sketch")
	}
	if previewSize > MaxPinImageSize {
		return PinFile{}, fmt.Errorf("image too large (max 8 MB)")
	}
	if name = cleanPinFileName(name); name != "" && name != "file" {
		f.Name = name
	}
	f.Size = previewSize
	if _, err := s.db.Exec(`UPDATE pin_files SET name = ?, size = ? WHERE id = ? AND pin_id = ?`, f.Name, f.Size, id, pinID); err != nil {
		return PinFile{}, fmt.Errorf("store: update sketch: %w", err)
	}
	s.note("pin.updated", nil, nil, idData(pinID))
	return f, nil
}
