package mutations

import "strings"

var leetTable = map[rune]rune{
	'a': '@',
	'e': '3',
	'i': '1',
	'o': '0',
	's': '$',
	'l': '1',
}

func leetspeak(s string) string {
	lower := strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	changed := false
	for _, r := range lower {
		if sub, ok := leetTable[r]; ok {
			b.WriteRune(sub)
			changed = true
		} else {
			b.WriteRune(r)
		}
	}
	if !changed {
		return s
	}
	return b.String()
}
