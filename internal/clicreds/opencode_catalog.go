package clicreds

import "sync"

// OpenCode's own provider list (ADR-0201). OpenCode serves its catalog —
// models.dev's providers, each with the variables it reads — from its own
// HTTP server (`opencode serve`, GET /provider), which the server package
// runs; what it read is handed here so For("opencode") carries it, and the
// vault can file a login OpenCode stored for any of them.

var opencodeCatalog struct {
	sync.Mutex
	rows []Provider
}

// SetOpencodeCatalog replaces OpenCode's catalog rows (declared ids and their
// OpenCode aliases are dropped here, so a row never appears twice).
func SetOpencodeCatalog(rows []Provider) {
	declared := map[string]bool{}
	if s, ok := staticSpec("opencode"); ok {
		for _, p := range s.Providers {
			declared[p.Provider] = true
			for _, alias := range opencodeIDs[p.Provider] {
				declared[alias] = true
			}
		}
	}
	out := make([]Provider, 0, len(rows))
	for _, r := range rows {
		if r.Provider == "" || declared[r.Provider] {
			continue
		}
		r.Native = nativeOpencode
		out = append(out, r)
	}
	opencodeCatalog.Lock()
	opencodeCatalog.rows = out
	opencodeCatalog.Unlock()
}

func withOpencodeCatalog(s Spec) Spec {
	if s.CLI != "opencode" {
		return s
	}
	opencodeCatalog.Lock()
	extra := opencodeCatalog.rows
	opencodeCatalog.Unlock()
	if len(extra) == 0 {
		return s
	}
	out := s
	out.Providers = append(append(make([]Provider, 0, len(s.Providers)+len(extra)), s.Providers...), extra...)
	return out
}

// OpencodeVaultID is the vault's id for an OpenCode provider id: the declared
// provider it is an alias of, else the id itself.
func OpencodeVaultID(id string) string {
	for vault, aliases := range opencodeIDs {
		for _, a := range aliases {
			if a == id {
				return vault
			}
		}
	}
	return id
}

// OpencodeID is OpenCode's id for a vault provider (the first alias OpenCode
// uses for it, else the id itself).
func OpencodeID(vault string) string {
	if aliases := opencodeIDs[vault]; len(aliases) > 0 {
		return aliases[0]
	}
	return vault
}

func staticSpec(cli string) (Spec, bool) {
	for _, s := range catalog {
		if s.CLI == cli {
			return s, true
		}
	}
	return Spec{}, false
}
