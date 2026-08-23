package workspace

import (
	"encoding/json"
	"math/rand"
)

// json helpers live here (not in workspace.go) so the registry logic stays
// free of encoding concerns.
func marshalWorkspaces(ws []Workspace) ([]byte, error) { return json.MarshalIndent(ws, "", "  ") }

func unmarshalWorkspaces(data []byte, ws *[]Workspace) error { return json.Unmarshal(data, ws) }

const suffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func randSuffix(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = suffixAlphabet[rand.Intn(len(suffixAlphabet))]
	}
	return string(b)
}
