package snips

// Parse collects unique placeholder names from body. It does not unescape:
// {{{{name}}}} is not a placeholder. First occurrence wins the default.
func Parse(body string) ([]Placeholder, error) {
	var out []Placeholder
	seen := make(map[string]int)
	i := 0
	for i < len(body) {
		if hasAt(body, i, "{{{{") || hasAt(body, i, "}}}}") {
			i += 4
			continue
		}
		if hasAt(body, i, "{{") {
			ph, end, err := readPlaceholder(body, i)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[ph.Name]; !ok {
				if len(out) >= MaxPlaceholders {
					return nil, parseErr(i, ph.Name, "too many placeholders")
				}
				seen[ph.Name] = len(out)
				out = append(out, ph)
			}
			i = end
			continue
		}
		i++
	}
	return out, nil
}
