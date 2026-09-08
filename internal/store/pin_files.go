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
	UpdatedAt  string `json:"updatedAt"`
	Source     string `json:"source,omitempty"`
	BaseFileID string `json:"baseFileId,omitempty"`
}

const pinFileCols = `id, pin_id, kind, name, mime, size, created_at, updated_at, source, base_file_id`

func scanPinFile(row interface{ Scan(...any) error }, f *PinFile) error {
	return row.Scan(&f.ID, &f.PinID, &f.Kind, &f.Name, &f.Mime, &f.Size, &f.CreatedAt, &f.UpdatedAt, &f.Source, &f.BaseFileID)
}

func ClassifyPinFile(name, mime string, size int64) (kind string, err error) {
	mime = strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".exe", ".bat", ".cmd", ".com", ".scr", ".msi", ".dll":
		return "", invalid("that file type is not allowed")
	}
	switch mime {
	case "image/png", "image/jpeg", "image/jpg", "image/webp", "image/gif":
		if size > MaxPinImageSize {
			return "", invalid("image too large (max 8 MB)")
		}
		return "image", nil
	}
	if size > MaxPinFileSize {
		return "", invalid("file too large (max 16 MB)")
	}
	if mime == "" || mime == "application/octet-stream" {
		mime = "application/octet-stream"
	}
	return "file", nil
}

func (s *Store) ListPinFiles(pinID string) ([]PinFile, error) {
	rows, err := s.db.Query(`SELECT `+pinFileCols+` FROM pin_files WHERE pin_id = ? ORDER BY created_at`, pinID)
	if err != nil {
		return nil, fmt.Errorf("store: list pin files: %w", err)
	}
	defer rows.Close()
	out := []PinFile{}
	for rows.Next() {
		var f PinFile
		if err := scanPinFile(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) GetPinFile(pinID, id string) (PinFile, error) {
	var f PinFile
	err := scanPinFile(s.db.QueryRow(`SELECT `+pinFileCols+` FROM pin_files WHERE id = ? AND pin_id = ?`, id, pinID), &f)
	if err == sql.ErrNoRows {
		return PinFile{}, ErrNotFound
	}
	if err != nil {
		return PinFile{}, fmt.Errorf("store: get pin file: %w", err)
	}
	return f, nil
}

// roomForPinFile checks the pin exists and is under the attachment cap.
func (s *Store) roomForPinFile(pinID string) error {
	if _, err := s.GetPin(pinID); err != nil {
		return err
	}
	n, err := s.countPinFiles(pinID)
	if err != nil {
		return err
	}
	if n >= MaxPinFiles {
		return invalid("a pin can have at most %d files", MaxPinFiles)
	}
	return nil
}

func (s *Store) insertPinFile(f PinFile) error {
	if _, err := s.db.Exec(`INSERT INTO pin_files (`+pinFileCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.PinID, f.Kind, f.Name, f.Mime, f.Size, f.CreatedAt, f.UpdatedAt, f.Source, f.BaseFileID); err != nil {
		return fmt.Errorf("store: add pin file: %w", err)
	}
	s.notePinUpdated(f.PinID)
	return nil
}

func (s *Store) AddPinFile(pinID, name, mime string, size int64) (PinFile, error) {
	if err := s.roomForPinFile(pinID); err != nil {
		return PinFile{}, err
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
	f := PinFile{ID: newID(name, "file"), PinID: pinID, Kind: kind, Name: name, Mime: mime, Size: size, CreatedAt: now, UpdatedAt: now}
	if f.Mime == "" {
		f.Mime = "application/octet-stream"
	}
	if err := s.insertPinFile(f); err != nil {
		return PinFile{}, err
	}
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
	s.notePinUpdated(pinID)
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

// AddPinSketch records a sketch. baseID, for an annotation, must name an
// image of the same pin: the sketch keeps the picture by reference and the
// browser rebuilds the background from /files/{baseId} on open, so the
// scene never embeds the bytes (finding 4 of the pins v2 review).
func (s *Store) AddPinSketch(pinID, name, source, baseID string, previewSize int64) (PinFile, error) {
	if err := s.roomForPinFile(pinID); err != nil {
		return PinFile{}, err
	}
	if previewSize > MaxPinImageSize {
		return PinFile{}, invalid("image too large (max 8 MB)")
	}
	if source != "annotate" {
		source = "blank"
	}
	baseID = strings.TrimSpace(baseID)
	if baseID != "" {
		base, err := s.GetPinFile(pinID, baseID)
		if err != nil || base.Kind != "image" {
			return PinFile{}, invalid("the picture to annotate is not an image of this pin")
		}
	}
	now := nowUTC()
	f := PinFile{
		ID: newID("sketch", "sketch"), PinID: pinID, Kind: "sketch",
		Name: cleanPinFileName(name), Mime: "image/png", Size: previewSize,
		CreatedAt: now, UpdatedAt: now, Source: source, BaseFileID: baseID,
	}
	if f.Name == "file" || f.Name == "sketch" {
		f.Name = "Sketch"
	}
	if err := s.insertPinFile(f); err != nil {
		return PinFile{}, err
	}
	return f, nil
}

// UpdatePinSketch renames and re-sizes a sketch and moves its updatedAt,
// which is what the preview URL carries as its cache-busting version.
func (s *Store) UpdatePinSketch(pinID, id, name string, previewSize int64) (PinFile, error) {
	f, err := s.GetPinFile(pinID, id)
	if err != nil {
		return PinFile{}, err
	}
	if f.Kind != "sketch" {
		return PinFile{}, invalid("not a sketch")
	}
	if previewSize > MaxPinImageSize {
		return PinFile{}, invalid("image too large (max 8 MB)")
	}
	if name = cleanPinFileName(name); name != "" && name != "file" {
		f.Name = name
	}
	f.Size = previewSize
	f.UpdatedAt = nowUTC()
	if _, err := s.db.Exec(`UPDATE pin_files SET name = ?, size = ?, updated_at = ? WHERE id = ? AND pin_id = ?`, f.Name, f.Size, f.UpdatedAt, id, pinID); err != nil {
		return PinFile{}, fmt.Errorf("store: update sketch: %w", err)
	}
	s.notePinUpdated(pinID)
	return f, nil
}
