package session

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// TreeNode is one JSONL entry in parent/child form.
type TreeNode struct {
	ID       string     `json:"id"`
	ParentID string     `json:"parentId,omitempty"`
	Kind     string     `json:"kind"`
	Role     string     `json:"role,omitempty"`
	Text     string     `json:"text,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// Tree is the session as roots + current leaf.
type Tree struct {
	Tree   []TreeNode `json:"tree"`
	LeafID string     `json:"leafId,omitempty"`
}

type treeRow struct {
	Type     string          `json:"type"`
	ID       string          `json:"id"`
	ParentID *string         `json:"parentId"`
	Message  json.RawMessage `json:"message"`
	Summary  string          `json:"summary"`
}

// ReadTree builds the entry tree from a JSONL session file.
func ReadTree(path string) (Tree, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Tree{Tree: []TreeNode{}}, nil
		}
		return Tree{}, err
	}
	defer f.Close()

	var rows []treeRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var r treeRow
		if json.Unmarshal(line, &r) != nil || r.ID == "" {
			continue
		}
		if r.Type == "session" || r.Type == "session_info" {
			continue
		}
		rows = append(rows, r)
	}
	if err := sc.Err(); err != nil {
		return Tree{}, err
	}

	byID := map[string]TreeNode{}
	order := make([]string, 0, len(rows))
	for _, r := range rows {
		n := TreeNode{ID: r.ID, Kind: r.Type}
		if r.ParentID != nil {
			n.ParentID = *r.ParentID
		}
		if r.Type == "message" {
			n.Role, n.Text = messageBits(r.Message)
		} else if r.Summary != "" {
			n.Text = clip(r.Summary, 120)
		}
		byID[n.ID] = n
		order = append(order, n.ID)
	}
	kids := map[string][]string{}
	var roots []string
	for _, id := range order {
		n := byID[id]
		if n.ParentID != "" {
			if _, ok := byID[n.ParentID]; ok {
				kids[n.ParentID] = append(kids[n.ParentID], id)
				continue
			}
		}
		roots = append(roots, id)
	}
	var build func(id string) TreeNode
	build = func(id string) TreeNode {
		n := byID[id]
		n.Children = nil
		for _, cid := range kids[id] {
			n.Children = append(n.Children, build(cid))
		}
		return n
	}
	out := make([]TreeNode, 0, len(roots))
	for _, id := range roots {
		out = append(out, build(id))
	}
	leaf := ""
	if len(order) > 0 {
		leaf = order[len(order)-1]
	}
	return Tree{Tree: out, LeafID: leaf}, nil
}

func messageBits(raw json.RawMessage) (role, text string) {
	if len(raw) == 0 {
		return "", ""
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return "", ""
	}
	role, _ = m["role"].(string)
	switch c := m["content"].(type) {
	case string:
		text = clip(c, 160)
	case []any:
		var b strings.Builder
		for _, part := range c {
			p, _ := part.(map[string]any)
			if p != nil && p["type"] == "text" {
				if t, _ := p["text"].(string); t != "" {
					if b.Len() > 0 {
						b.WriteByte(' ')
					}
					b.WriteString(t)
				}
			}
		}
		text = clip(b.String(), 160)
	}
	return role, text
}
