// Package textutil contains the spelling and casing primitives used by the
// generator. Password candidates are byte strings, but users type them on real
// keyboards, so Turkish case rules and ASCII folding are handled explicitly
// instead of relying on Unicode defaults that produce dotted-I artefacts.
package textutil

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var foldTable = map[rune]rune{
	// Turkish
	'ç': 'c', 'Ç': 'C',
	'ğ': 'g', 'Ğ': 'G',
	'ı': 'i', 'İ': 'I',
	'ö': 'o', 'Ö': 'O',
	'ş': 's', 'Ş': 'S',
	'ü': 'u', 'Ü': 'U',
	// Latin-1 / common European accents
	'á': 'a', 'à': 'a', 'â': 'a', 'ä': 'a', 'ã': 'a', 'å': 'a', 'ā': 'a', 'ă': 'a', 'ą': 'a',
	'Á': 'A', 'À': 'A', 'Â': 'A', 'Ä': 'A', 'Ã': 'A', 'Å': 'A', 'Ā': 'A', 'Ă': 'A', 'Ą': 'A',
	'ć': 'c', 'č': 'c', 'Ć': 'C', 'Č': 'C',
	'ď': 'd', 'Ď': 'D', 'đ': 'd', 'Đ': 'D',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e', 'ē': 'e', 'ė': 'e', 'ę': 'e',
	'É': 'E', 'È': 'E', 'Ê': 'E', 'Ë': 'E', 'Ē': 'E', 'Ė': 'E', 'Ę': 'E',
	'ĝ': 'g', 'Ģ': 'G', 'Ĝ': 'G',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i', 'ī': 'i', 'į': 'i',
	'Í': 'I', 'Ì': 'I', 'Î': 'I', 'Ï': 'I', 'Ī': 'I', 'Į': 'I',
	'ł': 'l', 'Ł': 'L',
	'ñ': 'n', 'ń': 'n', 'ň': 'n', 'Ñ': 'N', 'Ń': 'N', 'Ň': 'N',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ø': 'o', 'ō': 'o', 'ő': 'o',
	'Ó': 'O', 'Ò': 'O', 'Ô': 'O', 'Õ': 'O', 'Ø': 'O', 'Ō': 'O', 'Ő': 'O',
	'ř': 'r', 'Ř': 'R',
	'ś': 's', 'š': 's', 'ș': 's',
	'Ś': 'S', 'Š': 'S', 'Ș': 'S',
	'ť': 't', 'ţ': 't', 'Ť': 'T', 'Ţ': 'T',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ū': 'u', 'ů': 'u', 'ű': 'u',
	'Ú': 'U', 'Ù': 'U', 'Û': 'U', 'Ū': 'U', 'Ů': 'U', 'Ű': 'U',
	'ý': 'y', 'ÿ': 'y', 'Ý': 'Y', 'Ÿ': 'Y',
	'ź': 'z', 'ż': 'z', 'ž': 'z',
	'Ź': 'Z', 'Ż': 'Z', 'Ž': 'Z',
	'æ': 'a', 'Æ': 'A', 'œ': 'o', 'Œ': 'O',
}

// FoldASCII transliterates Turkish and common Latin-1 letters to ASCII. Runes
// with no single-character mapping are preserved unchanged.
func FoldASCII(s string) string {
	ascii := true
	for _, r := range s {
		if r > unicode.MaxASCII {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r <= unicode.MaxASCII {
			b.WriteRune(r)
			continue
		}
		if f, ok := foldTable[r]; ok {
			b.WriteRune(f)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Lower lowercases with a Turkish-safe mapping: U+0130 (İ) becomes plain "i"
// rather than "i" plus a combining dot.
func Lower(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == 'İ' {
			b.WriteByte('i')
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// Upper uppercases using Unicode simple mappings.
func Upper(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// TurkishLower applies Turkish dotless/dotted I rules.
func TurkishLower(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'I':
			b.WriteRune('ı')
		case 'İ':
			b.WriteRune('i')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// TurkishUpper applies Turkish dotless/dotted I rules.
func TurkishUpper(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'i':
			b.WriteRune('İ')
		case 'ı':
			b.WriteRune('I')
		default:
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

// Title capitalises the first rune and lowercases the remainder.
func Title(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		if runes[i] == 'İ' {
			runes[i] = 'i'
		} else {
			runes[i] = unicode.ToLower(runes[i])
		}
	}
	return string(runes)
}

// TurkishTitle is Title with Turkish case rules.
func TurkishTitle(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	switch runes[0] {
	case 'i':
		runes[0] = 'İ'
	case 'ı':
		runes[0] = 'I'
	default:
		runes[0] = unicode.ToUpper(runes[0])
	}
	for i := 1; i < len(runes); i++ {
		switch runes[i] {
		case 'I':
			runes[i] = 'ı'
		case 'İ':
			runes[i] = 'i'
		default:
			runes[i] = unicode.ToLower(runes[i])
		}
	}
	return string(runes)
}

// TitleEach capitalises every separator-delimited component while preserving
// the separators. "corp.local" -> "Corp.Local", "active_directory" ->
// "Active_Directory".
func TitleEach(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	start := true
	for _, r := range s {
		if isSeparator(r) {
			b.WriteRune(r)
			start = true
			continue
		}
		if start {
			b.WriteRune(unicode.ToUpper(r))
			start = false
			continue
		}
		if r == 'İ' {
			b.WriteByte('i')
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// Camel joins separator-delimited words with the first word lowercased and the
// following words capitalised: "active_directory" -> "activeDirectory".
func Camel(s string) string {
	words := SplitName(s)
	if len(words) <= 1 {
		return Title(s)
	}
	var b strings.Builder
	b.Grow(len(s))
	for i, w := range words {
		if i == 0 {
			b.WriteString(Lower(w))
			continue
		}
		b.WriteString(Title(w))
	}
	return b.String()
}

// Pascal joins separator-delimited words with every word capitalised:
// "active_directory" -> "ActiveDirectory".
func Pascal(s string) string {
	words := SplitName(s)
	if len(words) <= 1 {
		return Title(s)
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, w := range words {
		b.WriteString(Title(w))
	}
	return b.String()
}

// Options controls Forms.
type Options struct {
	// Turkish enables Turkish dotted/dotless I variants for spellings that
	// actually contain Turkish-specific letters.
	Turkish bool
	// ForceTurkish enables Turkish variants for every spelling, including
	// ASCII-only ones. The -tr flag sets this.
	ForceTurkish bool
	// Camel adds a camelCase spelling for separator-delimited keywords.
	Camel bool
	// Pascal adds a PascalCase spelling for separator-delimited keywords.
	Pascal bool
}

// Forms returns every distinct spelling-and-case form worth trying for one
// keyword, in priority order: lowercase, Title, UPPER, source spelling, then
// ASCII-folded and Turkish variants.
func Forms(s string, opts Options) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	spellings := make([]string, 0, 3)
	addSpelling := func(v string) {
		if v == "" {
			return
		}
		for _, existing := range spellings {
			if existing == v {
				return
			}
		}
		spellings = append(spellings, v)
	}
	addSpelling(s)
	addSpelling(FoldASCII(s))
	if opts.Camel {
		addSpelling(Camel(s))
	}
	if opts.Pascal {
		addSpelling(Pascal(s))
	}

	seen := make(map[string]bool, 12)
	out := make([]string, 0, 12)
	add := func(v string) {
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}

	for _, sp := range spellings {
		add(Lower(sp))
		add(Title(sp))
		add(Upper(sp))
		add(sp)
		if opts.ForceTurkish || (opts.Turkish && HasTurkish(sp)) {
			add(TurkishLower(sp))
			add(TurkishUpper(sp))
			add(TurkishTitle(sp))
		}
	}
	return out
}

// HasTurkish reports whether s contains Turkish-specific letters (including
// dotless/dotted I) that change casing behaviour.
func HasTurkish(s string) bool {
	for _, r := range s {
		switch r {
		case 'ç', 'Ç', 'ğ', 'Ğ', 'ı', 'İ', 'ö', 'Ö', 'ş', 'Ş', 'ü', 'Ü':
			return true
		}
	}
	return false
}

// HasUpper reports whether s contains an uppercase letter.
func HasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// IsLower reports whether s contains no uppercase letters. Digits and
// separators do not make a string uppercase.
func IsLower(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

// IsASCII reports whether every byte is in the ASCII range.
func IsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// IsAlnumOnly reports whether s consists only of letters and digits. It is
// used to keep mid-word special insertion away from domain names and
// separator-containing keywords.
func IsAlnumOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// JoinWords removes whitespace between the words of a phrase. It returns the
// joined spelling and whether a change was made.
func JoinWords(s string) (string, bool) {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return s, false
	}
	return strings.Join(fields, ""), true
}

// SplitName splits a username or phrase on its separators and drops empty
// components.
func SplitName(s string) []string {
	return strings.FieldsFunc(s, isSeparator)
}

// RuneLen returns the number of Unicode code points in s.
func RuneLen(s string) int {
	return utf8.RuneCountInString(s)
}

// FirstRune returns the first rune of s, or 0 when s is empty.
func FirstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

// LastRune returns the last rune of s, or 0 when s is empty.
func LastRune(s string) rune {
	var last rune
	for _, r := range s {
		last = r
	}
	return last
}

func isSeparator(r rune) bool {
	switch r {
	case '.', '_', '-', ' ':
		return true
	}
	return false
}
