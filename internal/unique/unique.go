package unique

// Strings removes duplicates from s.
func Strings(s ...string) []string {
	if len(s) < 2 {
		return s
	}

	m := make(map[string]bool)
	out := make([]string, 0, len(s))

	for _, s := range s {
		if m[s] {
			continue
		}
		m[s] = true
		out = append(out, s)
	}

	return out
}
