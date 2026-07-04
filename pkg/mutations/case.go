package mutations

import (
	"strings"
	"unicode"
)

func caseVariants(s string) []string {
	candidates := []string{
		strings.ToLower(s),
		strings.ToUpper(s),
		capitalize(s),
		toCamelCase(s),
		s,
	}
	seen := make(map[string]bool, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, v := range candidates {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}

func toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(words) <= 1 {
		return capitalize(s)
	}
	var b strings.Builder
	b.Grow(len(s))
	for i, w := range words {
		if i == 0 {
			b.WriteString(strings.ToLower(w))
		} else {
			b.WriteString(capitalize(w))
		}
	}
	return b.String()
}
