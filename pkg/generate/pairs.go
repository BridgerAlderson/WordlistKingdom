package generate

import (
	"fmt"
	"strings"
	"unicode"

	"wordlistkingdom/pkg/textutil"
)

// phasePairs emits two-keyword combinations. Only the first PairLimit keywords
// take part so a large keyword file cannot produce a quadratic explosion.
// Ordered pairs are produced because corp.localCorp and Corpcorp.local are
// different guesses, and both appear in real password dumps.
func (g *Generator) phasePairs() {
	limit := minInt(len(g.terms), g.cfg.PairLimit)
	if limit < 1 {
		return
	}
	g.phase = PhasePairs

	numbers := g.pairNumbers()
	specials := g.pairSpecials()
	year := fmt.Sprintf("%d", g.now)

	// A doubled keyword (corpcorp, Corpcorp, corpCorp) is a real pattern and
	// was present in the legacy generator. Keep it to alphanumeric terms and
	// only apply the two most common suffixes.
	for _, t := range g.terms[:limit] {
		if !textutil.IsAlnumOnly(t.primary) || !hasLetter(t.primary) {
			continue
		}
		for _, base := range g.pairBases(t.primary, t.primary, "") {
			if !g.acceptForm(base) {
				continue
			}
			for _, candidate := range g.doubledSuffixes(base, year) {
				if !g.push(candidate) {
					return
				}
			}
		}
	}

	if limit < 2 {
		return
	}

	for i := 0; i < limit; i++ {
		for j := 0; j < limit; j++ {
			if i == j {
				continue
			}
			a := g.terms[i].primary
			b := g.terms[j].primary
			if strings.EqualFold(a, b) {
				continue
			}
			for _, sep := range g.pairSeps() {
				for _, base := range g.pairBases(a, b, sep) {
					if !g.acceptForm(base) {
						continue
					}
					if !g.push(base) {
						return
					}
					if g.cfg.Pattern.Digits && canAppendNumber(base) {
						for _, n := range numbers {
							if !g.push(base + n) {
								return
							}
						}
					}
					if g.cfg.Pattern.Special && canAppendSpecial(base) {
						for _, s := range specials {
							if !g.push(base + s) {
								return
							}
						}
					}
					if g.cfg.Pattern.Digits && g.cfg.Pattern.Special && canAppendNumber(base) {
						// Only the most common symbol in year/symbol shapes;
						// two orders because both occur in real lists.
						if !g.push(base + year + "!") {
							return
						}
						if !g.push(base + "!" + year) {
							return
						}
					}
				}
			}
		}
	}
}

// pairBases returns the case-consistent forms of a two-word combination.
// TitleWhole differs from TitleEach for dotted/underscored keywords:
//
//	corp.local + corp + "." -> Corp.local.corp   (TitleWhole)
//	                         -> Corp.Local.Corp   (TitleEach)
func (g *Generator) pairBases(a, b, sep string) []string {
	la := textutil.Lower(a)
	lb := textutil.Lower(b)
	wa := textutil.Title(a)
	wb := textutil.Title(b)
	ua := textutil.Upper(a)
	ub := textutil.Upper(b)

	full := la + sep + lb
	out := make([]string, 0, 14)
	out = append(out,
		full,
		textutil.Title(full),
		ua+sep+ub,
		la+sep+wb,
	)

	if sep != "" {
		ta := textutil.TitleEach(a)
		tb := textutil.TitleEach(b)
		out = append(out, ta+sep+tb)
		if g.cfg.Level >= 3 {
			for _, x := range []string{la, ta, ua} {
				for _, y := range []string{lb, tb, ub} {
					out = append(out, x+sep+y)
				}
			}
		}
	} else if g.cfg.Level >= 3 {
		// With no separator, TitleEach would capitalise after an internal dot
		// (CorpCorp.Local), so use whole-string title forms instead.
		for _, x := range []string{la, wa, ua} {
			for _, y := range []string{lb, wb, ub} {
				out = append(out, x+y)
			}
		}
	}
	return dedupeStrings(out)
}

// doubledSuffixes keeps self-combinations useful without adding another full
// suffix ladder.
func (g *Generator) doubledSuffixes(base, year string) []string {
	out := []string{base}
	if g.cfg.Pattern.Digits && canAppendNumber(base) {
		out = append(out, base+year)
	}
	if g.cfg.Pattern.Special && canAppendSpecial(base) {
		out = append(out, base+"!")
	}
	if g.cfg.Pattern.Digits && g.cfg.Pattern.Special && canAppendNumber(base) {
		out = append(out, base+year+"!")
	}
	return out
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
