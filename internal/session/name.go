package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// SetName appends a session_info entry (same store pi /name uses).
func SetName(path, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	parent, err := lastEntryID(path)
	if err != nil {
		return err
	}
	entry := map[string]any{
		"type":      "session_info",
		"id":        fmt.Sprintf("name-%d", time.Now().UnixNano()),
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"name":      name,
	}
	if parent != "" {
		entry["parentId"] = parent
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func lastEntryID(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	id := ""
	for sc.Scan() {
		var raw struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(sc.Bytes(), &raw) == nil && raw.ID != "" {
			id = raw.ID
		}
	}
	return id, sc.Err()
}
