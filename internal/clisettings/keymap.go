package clisettings

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

// The document primitives a *key-map* writer needs (ADR-0174). Apply writes
// declared scalar fields; a key map is one file of action -> chord list, so the
// CLI key-map engine opens the same kind of document here and reuses the same
// parser, byte-span splice, atomic write and revision check rather than a
// second implementation of guarantees that took two adversarial reviews to get
// right (a file with no final newline, a `[table]`-shaped line inside a string,
// a `// comment` before a closing brace).
//
// Doc keeps the bytes it was read from, so a save splices them: comments, key
// order and every key PiCode does not know survive.

// ErrShape is returned when a key holds something other than a string or a list
// of strings. PiCode does not rewrite a shape it cannot recognise, and it never
// silently replaces one.
var ErrShape = errors.New("the file holds this key in a shape PiCode does not rewrite")

// Doc is one configuration file opened for edit.
type Doc struct {
	path     string
	format   Format
	text     []byte
	doc      map[string]any
	revision string
	exists   bool
}

// OpenDoc reads one file. A file that does not exist is an empty document — the
// CLI is running on its own defaults, and a write creates it — not an error.
func OpenDoc(path string, f Format) (Doc, error) {
	d := Doc{path: path, format: f, doc: map[string]any{}}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return d, nil
	}
	if err != nil {
		return Doc{}, err
	}
	doc, err := decode(raw, f, path)
	if err != nil {
		return Doc{}, err
	}
	d.text, d.doc, d.exists, d.revision = raw, doc, true, revision(raw)
	return d, nil
}

// Revision is the hash of the bytes this document was read from, empty when
// there was no file: a caller hands it back to Save, which refuses a file that
// moved since.
func (d Doc) Revision() string { return d.revision }

// Exists reports whether the file was there when it was opened.
func (d Doc) Exists() bool { return d.exists }

// value is the decoded value of one key. key is the file's own key, dots and
// all: a key map's action ids are literal keys, never a nested path.
func (d Doc) value(key string) (any, bool) { return lookup(d.doc, []string{key}) }

// Strings reads one key as the list of strings the format can carry: a bare
// string is one entry, a list of strings is the list, and a key that is not
// there reports found=false. Any other shape is ErrShape — the caller names the
// key and the file, because a key map that holds a table is a file PiCode must
// not touch.
func (d Doc) Strings(key string) (values []string, found bool, err error) {
	v, ok := d.value(key)
	if !ok {
		return nil, false, nil
	}
	out, err := asStrings(v)
	if err != nil {
		return nil, true, fmt.Errorf("%s: %w", key, err)
	}
	return out, true, nil
}

// SetStrings replaces the value of one key with a list of strings, inserting
// the key when the file does not have it. An existing value is spliced in place
// when it is one line — so a comment beside it stays beside it — and removed
// and re-inserted when it is a block, which a single span cannot address.
func (d *Doc) SetStrings(key string, values []string) error {
	if key == "" {
		return errors.New("no key to write")
	}
	lit, err := listLiteral(d.format, values)
	if err != nil {
		return err
	}
	if v, ok := d.value(key); ok {
		if _, err := asStrings(v); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		start, end, ok := valueSpan(d.text, d.format, []string{key})
		if ok && !bytes.Contains(d.text[start:end], []byte("\n")) {
			d.text = append(append(append([]byte{}, d.text[:start]...), lit...), d.text[end:]...)
			return d.reparse()
		}
		if d.text, err = removeValue(d.text, d.format, []string{key}); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	if d.text, err = insertValue(d.text, d.format, []string{key}, lit); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return d.reparse()
}

// RemoveKey drops one key, so the CLI falls back to its own default, and takes
// the container with it when that key was the last one in it. A key the file
// does not set is not an error.
func (d *Doc) RemoveKey(key string) error {
	text, err := removeValue(d.text, d.format, []string{key})
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	d.text = text
	return d.reparse()
}

// Save writes the document atomically. rev is the revision the caller read: a
// file that changed underneath is refused with ErrStale rather than
// overwritten, and a splice that produced an unreadable file is refused here
// rather than discovered by the CLI.
func (d Doc) Save(rev string) error {
	raw, err := os.ReadFile(d.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		raw = nil
	}
	// The hash is always of the bytes on disk, empty file included, and an
	// empty *revision* means the caller asked for no check (a file that is not
	// there yet, which the settings engine reads the same way). Comparing the
	// hash of an empty string against "" made an existing empty file
	// unwritable (2026-09-21).
	if rev != "" && revision(raw) != rev {
		return ErrStale
	}
	if len(bytes.TrimSpace(d.text)) > 0 {
		if _, err := decode(d.text, d.format, d.path); err != nil {
			return fmt.Errorf("refusing to write %s: the result would not parse (%w)", d.path, err)
		}
	}
	return writeAtomic(d.path, d.text)
}

// reparse re-reads the document from the text after a splice, so a second edit
// in the same request sees the first.
func (d *Doc) reparse() error {
	doc, err := decode(d.text, d.format, d.path)
	if err != nil {
		return fmt.Errorf("refusing to write %s: the result would not parse (%w)", d.path, err)
	}
	d.doc = doc
	return nil
}

// listLiteral renders a list of strings in the file's own syntax. A key map's
// value is `chord | [chord]` in every CLI that keeps one, so the only shapes
// written are a list and an empty list (`[]`, which is how a CLI unbinds).
func listLiteral(f Format, values []string) (string, error) {
	switch f {
	case FormatJSON, FormatJSONC:
		var b bytes.Buffer
		b.WriteByte('[')
		for i, v := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteJSON(v))
		}
		b.WriteByte(']')
		return b.String(), nil
	case FormatYAML:
		// A YAML flow sequence, quoted item by item: a chord carries `+`, which
		// is a plain scalar in YAML, but quoting every item means the one that
		// needs it (a key named `-`, or `[]` itself) cannot be the exception.
		var b bytes.Buffer
		b.WriteByte('[')
		for i, v := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteJSON(v))
		}
		b.WriteByte(']')
		return b.String(), nil
	case FormatTOML:
		// A TOML array would splice as one line, but no key map PiCode writes
		// is TOML (codex's is nested and gets its own engine): refuse rather
		// than ship a path nothing has exercised.
		return "", fmt.Errorf("this format cannot carry a key map")
	}
	return "", fmt.Errorf("unknown format %q", f)
}

// asStrings accepts the two shapes a chord list is written in.
func asStrings(v any) ([]string, error) {
	switch value := v.(type) {
	case string:
		return []string{value}, nil
	case []string:
		return value, nil
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			s, ok := item.(string)
			if !ok {
				return nil, ErrShape
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, ErrShape
}
