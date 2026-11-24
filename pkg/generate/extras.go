package generate

import (
	"fmt"
	"strings"
	"unicode"

	"wordlistkingdom/pkg/config"
	"wordlistkingdom/pkg/textutil"
)

// ---------------------------------------------------------------------------
// Usernames
// ---------------------------------------------------------------------------

// phaseUsers turns a user list into account-flavoured candidates and crosses
// the strongest few keywords with each user. It does not cross every user with
// every keyword: a 5000-user engagement would otherwise produce millions of
// low-value lines.
func (g *Generator) phaseUsers() {
	g.phase = PhaseUsernames

	specials := g.userSpecials()
	year := fmt.Sprintf("%d", g.now)
	short, hasShort := shortYear(g.now)

	for _, u := range g.users {
		numbers := g.userNumbers()
		if u.BirthYear > 0 {
			numbers = appendUnique(numbers, fmt.Sprintf("%d", u.BirthYear))
			if by, ok := shortYear(u.BirthYear); ok {
				numbers = appendUnique(numbers, by)
			}
		}

		forms := g.usernameForms(u.Username)
		forms = append(forms, g.personForms(u)...)
		for _, f := range forms {
			if !g.acceptForm(f) {
				continue
			}
			if !g.push(f) {
				return
			}
			if g.cfg.Pattern.Digits && canAppendNumber(f) {
				for _, n := range numbers {
					if !g.push(f + n) {
						return
					}
				}
				if g.cfg.Pattern.Special {
					if !g.push(f + year + "!") {
						return
					}
					if !g.push(f + "!" + year) {
						return
					}
					if u.BirthYear > 0 {
						by := fmt.Sprintf("%d", u.BirthYear)
						if !g.push(f + by + "!") {
							return
						}
						if !g.push(f + "!" + by) {
							return
						}
					}
				}
			}
			if g.cfg.Pattern.Special && canAppendSpecial(f) {
				for _, s := range specials {
					if !g.push(f + s) {
						return
					}
				}
			}
		}

		if g.cfg.Perm < 2 || g.cfg.UserKeywordLimit <= 0 {
			continue
		}
		limit := minInt(len(g.terms), g.cfg.UserKeywordLimit)
		for k := 0; k < limit; k++ {
			kw := g.terms[k].primary
			for _, sep := range g.userSeps() {
				for _, base := range g.pairBases(u.Username, kw, sep) {
					if !g.acceptForm(base) {
						continue
					}
					if !g.push(base) {
						return
					}
					if g.cfg.Pattern.Digits && canAppendNumber(base) {
						if !g.push(base + year) {
							return
						}
						if hasShort && !g.push(base+short) {
							return
						}
					}
					if g.cfg.Pattern.Special && canAppendSpecial(base) {
						if !g.push(base + "!") {
							return
						}
					}
					if g.cfg.Pattern.Digits && g.cfg.Pattern.Special && canAppendNumber(base) {
						if !g.push(base + year + "!") {
							return
						}
					}
				}
			}
		}
	}
}

// usernameForms expands a username into spellings and name-order variants.
func (g *Generator) usernameForms(u string) []string {
	spellings := []string{u}
	if fold := textutil.FoldASCII(u); fold != u {
		spellings = append(spellings, fold)
	}

	parts := textutil.SplitName(u)
	if len(parts) >= 2 {
		first := parts[0]
		last := parts[len(parts)-1]
		initial := string(textutil.FirstRune(first))
		spellings = appendUnique(spellings,
			strings.Join(parts, ""),
			first+last,
			last+first,
			initial+last,
			last+initial,
			first+"."+last,
			initial+"."+last,
			last+"."+first,
		)
	}

	out := make([]string, 0, len(spellings)*4)
	seen := make(map[string]bool, len(spellings)*4)
	for _, sp := range spellings {
		opts := textutil.Options{Turkish: g.cfg.Turkish || textutil.HasTurkish(sp)}
		for _, f := range textutil.Forms(sp, opts) {
			if f == "" || seen[f] || textutil.RuneLen(f) > 64 {
				continue
			}
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// personForms expands optional full-name data into the name shapes people use
// as passwords: john.smith, johnsmith, jsmith, smithj, John.Smith, JohnSmith.
func (g *Generator) personForms(u User) []string {
	if u.First == "" || u.Last == "" {
		return nil
	}
	first := textutil.Lower(u.First)
	last := textutil.Lower(u.Last)
	initial := string(textutil.FirstRune(first))
	if initial == "" || first == "" || last == "" {
		return nil
	}

	spellings := []string{
		first,
		last,
		first + last,
		last + first,
		first + "." + last,
		first + "_" + last,
		first + "-" + last,
		initial + "." + last,
		initial + last,
		last + initial,
		last + "." + first,
		textutil.Pascal(u.First + " " + u.Last),
		textutil.Pascal(initial + " " + last),
		textutil.Pascal(last + " " + first),
		textutil.TitleEach(first + "." + last),
	}
	seen := make(map[string]bool, len(spellings)*4)
	out := make([]string, 0, len(spellings)*4)
	for _, sp := range spellings {
		if sp == "" {
			continue
		}
		opts := textutil.Options{Turkish: true, ForceTurkish: g.cfg.Turkish}
		for _, f := range textutil.Forms(sp, opts) {
			if f == "" || seen[f] || textutil.RuneLen(f) > 64 {
				continue
			}
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

func (g *Generator) userNumbers() []string {
	out := []string{
		fmt.Sprintf("%d", g.now),
		fmt.Sprintf("%d", g.now-1),
		fmt.Sprintf("%02d", g.now%100),
		"123",
	}
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "1234", "1")
	}
	return dedupeStrings(out)
}

func (g *Generator) userSpecials() []string {
	out := []string{"!", "@", "."}
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "#")
	}
	allowed := make(map[string]bool, len(g.specials))
	for _, s := range g.specials {
		allowed[s] = true
	}
	filtered := out[:0]
	for _, s := range out {
		if allowed[s] {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (g *Generator) userSeps() []string {
	out := []string{""}
	if !g.cfg.Pattern.Special {
		return out
	}
	out = append(out, ".", "_")
	if g.cfg.Level >= 1 {
		out = append(out, "-", "@")
	}
	return out
}

// ---------------------------------------------------------------------------
// Role affixes
// ---------------------------------------------------------------------------

// phaseAffixes emits service-account shapes: AdminCorp, corp_backup,
// IT_Support. Role words are generic and do not depend on the target language
// unless the operator adds their own keyword.
func (g *Generator) phaseAffixes() {
	g.phase = PhaseAffixes

	seps := []string{""}
	if g.cfg.Level >= 2 && g.cfg.Pattern.Special {
		seps = append(seps, "_", ".")
	}
	year := fmt.Sprintf("%d", g.now)

	for _, t := range g.terms {
		for _, role := range g.roles() {
			if strings.EqualFold(role, t.primary) {
				continue
			}
			for _, sep := range seps {
				for _, order := range [][2]string{{role, t.primary}, {t.primary, role}} {
					for _, base := range g.affixBases(order[0], order[1], sep) {
						candidates := []string{base}
						if g.cfg.Pattern.Digits && canAppendNumber(base) {
							candidates = append(candidates, base+year)
						}
						for _, candidate := range candidates {
							if !g.acceptForm(candidate) {
								continue
							}
							if !g.push(candidate) {
								return
							}
						}
					}
				}
			}
		}
	}
}

func (g *Generator) affixBases(a, b, sep string) []string {
	la := textutil.Lower(a)
	lb := textutil.Lower(b)
	ta := textutil.TitleEach(a)
	tb := textutil.TitleEach(b)
	ua := textutil.Upper(a)
	ub := textutil.Upper(b)

	full := la + sep + lb
	return dedupeStrings([]string{
		full,
		textutil.Title(full),
		ta + sep + tb,
		ua + sep + ub,
	})
}

// ---------------------------------------------------------------------------
// Leetspeak
// ---------------------------------------------------------------------------

type leetSub struct {
	from    rune
	special rune
	digit   rune
}

var leetPatterns = [][]leetSub{
	{{from: 'a', special: '@'}},
	{{from: 'o', digit: '0'}},
	{{from: 'i', digit: '1'}},
	{{from: 'e', digit: '3'}},
	{{from: 's', special: '$', digit: '5'}},
	{{from: 'a', special: '@'}, {from: 'o', digit: '0'}},
	{{from: 'a', special: '@'}, {from: 'e', digit: '3'}, {from: 'o', digit: '0'}},
	{{from: 'a', special: '@'}, {from: 'e', digit: '3'}, {from: 'i', digit: '1'}, {from: 'o', digit: '0'}, {from: 's', special: '$', digit: '5'}},
	{{from: 'e', digit: '3'}, {from: 'o', digit: '0'}},
}

// phaseLeet applies partial leetspeak. Real leet passwords usually substitute
// one or two characters, not every possible character; full substitution is
// kept for deep level only.
func (g *Generator) phaseLeet() {
	g.phase = PhaseLeet

	limit := 3
	if g.cfg.Level >= 2 {
		limit = 6
	}
	if g.cfg.Level >= 3 {
		limit = len(leetPatterns)
	}
	year := fmt.Sprintf("%d", g.now)

	for _, t := range g.terms {
		for _, base := range t.lower {
			for _, pattern := range leetPatterns[:limit] {
				leet, changed := applyLeet(base, pattern, g.cfg.Pattern)
				if !changed {
					continue
				}
				if !g.push(leet) {
					return
				}
				if g.cfg.Pattern.Upper {
					if !g.push(textutil.Title(leet)) {
						return
					}
				}
				if g.cfg.Level >= 2 && g.cfg.Pattern.Digits {
					if !g.push(leet + year) {
						return
					}
				}
				if g.cfg.Level >= 2 && g.cfg.Pattern.Special {
					if !g.push(leet + "!") {
						return
					}
				}
				if g.cfg.Level >= 3 && g.cfg.Pattern.Digits && g.cfg.Pattern.Special {
					if !g.push(leet + year + "!") {
						return
					}
				}
			}
		}
	}
}

func applyLeet(s string, pattern []leetSub, classes config.Pattern) (string, bool) {
	runes := []rune(s)
	changed := false
	for _, sub := range pattern {
		to, ok := sub.target(classes)
		if !ok {
			continue
		}
		for i, r := range runes {
			if unicode.ToLower(r) != sub.from {
				continue
			}
			runes[i] = to
			changed = true
			break
		}
	}
	if !changed {
		return s, false
	}
	return string(runes), true
}

func (s leetSub) target(classes config.Pattern) (rune, bool) {
	if s.special != 0 && classes.Special {
		return s.special, true
	}
	if s.digit != 0 && classes.Digits {
		return s.digit, true
	}
	return 0, false
}

// ---------------------------------------------------------------------------
// Seasons and months
// ---------------------------------------------------------------------------

// phaseSeasons emits keyword+season and season+keyword shapes. Seasons are a
// real but comparatively rare corporate pattern, so they only appear from
// extended level onward; months are deep-level only.
func (g *Generator) phaseSeasons() {
	g.phase = PhaseSeasons
	tokens := g.seasonTokens()
	year := fmt.Sprintf("%d", g.now)

	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, g.cfg.Level >= 3) {
			for _, season := range tokens {
				for _, base := range []string{f + season, season + f} {
					if !g.push(base) {
						return
					}
					if g.cfg.Pattern.Digits && !g.push(base+year) {
						return
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Deep variants
// ---------------------------------------------------------------------------

// phaseDeep contains the low-probability shapes that are still useful for
// offline cracking: prefix years, numeric prefixes, safe mid-word symbols and
// three-word combinations. It is gated behind level 3 or -perm 3.
func (g *Generator) phaseDeep() {
	g.phase = PhaseDeep

	if g.cfg.Level >= 3 {
		g.deepPrefixes()
		if g.stop {
			return
		}
		g.deepMidSymbols()
		if g.stop {
			return
		}
	}
	if g.cfg.Perm >= 3 {
		g.deepTriples()
	}
}

func (g *Generator) deepPrefixes() {
	years := []string{fmt.Sprintf("%d", g.now), fmt.Sprintf("%d", g.now-1)}
	if short, ok := shortYear(g.now); ok {
		years = append(years, short)
	}
	numbers := []string{"123", "1"}

	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, false) {
			if g.cfg.Pattern.Digits {
				for _, y := range years {
					if !g.push(y + f) {
						return
					}
				}
				for _, n := range numbers {
					if !g.push(n + f) {
						return
					}
				}
			}
		}
	}
}

func (g *Generator) deepMidSymbols() {
	if !g.cfg.Pattern.Special {
		return
	}
	chars := make([]string, 0, 5)
	for _, c := range []string{"!", "@", ".", "_", "-"} {
		if g.hasSpecial(c) {
			chars = append(chars, c)
		}
	}
	if len(chars) == 0 {
		return
	}

	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, false) {
			// Only mutate plain alphanumeric words. Inserting a symbol into
			// corp.local would produce broken domain-shaped strings.
			if !textutil.IsAlnumOnly(f) {
				continue
			}
			runes := []rune(f)
			if len(runes) < 5 {
				continue
			}
			mid := len(runes) / 2
			left := string(runes[:mid])
			right := string(runes[mid:])
			for _, c := range chars {
				if !g.push(left + c + right) {
					return
				}
			}
		}
	}
}

func (g *Generator) deepTriples() {
	limit := minInt(len(g.terms), g.cfg.TripleLimit)
	if limit < 3 {
		return
	}
	seps := []string{""}
	if g.cfg.Pattern.Special {
		seps = append(seps, ".", "_")
	}

	for i := 0; i < limit; i++ {
		for j := 0; j < limit; j++ {
			if j == i {
				continue
			}
			for k := 0; k < limit; k++ {
				if k == i || k == j {
					continue
				}
				a := g.terms[i].primary
				b := g.terms[j].primary
				c := g.terms[k].primary
				if strings.EqualFold(a, b) || strings.EqualFold(b, c) || strings.EqualFold(a, c) {
					continue
				}
				for _, sep := range seps {
					lower := textutil.Lower(a) + sep + textutil.Lower(b) + sep + textutil.Lower(c)
					title := textutil.Title(lower)
					upper := textutil.Upper(a) + sep + textutil.Upper(b) + sep + textutil.Upper(c)
					for _, base := range []string{lower, title, upper} {
						if !g.acceptForm(base) {
							continue
						}
						if !g.push(base) {
							return
						}
					}
				}
			}
		}
	}
}
