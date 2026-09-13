package snips

import "strings"

// Expand unescapes {{{{ / }}}} then substitutes placeholders in one pass.
// Inserted values are not re-scanned. Missing required names are listed;
// their slots emit empty string. Reserved names come from ctx only.
func Expand(body string, values, ctx map[string]string) (string, []string, error) {
	var b strings.Builder
	var missing []string
	seenMiss := map[string]bool{}
	i := 0
	for i < len(body) {
		if hasAt(body, i, "{{{{") {
			b.WriteString("{{")
			i += 4
			continue
		}
		if hasAt(body, i, "}}}}") {
			b.WriteString("}}")
			i += 4
			continue
		}
		if hasAt(body, i, "{{") {
			ph, end, err := readPlaceholder(body, i)
			if err != nil {
				return "", nil, err
			}
			val, ok := lookup(ph, values, ctx)
			if !ok {
				if !seenMiss[ph.Name] {
					missing = append(missing, ph.Name)
					seenMiss[ph.Name] = true
				}
			} else {
				b.WriteString(val)
			}
			i = end
			continue
		}
		b.WriteByte(body[i])
		i++
	}
	return b.String(), missing, nil
}
