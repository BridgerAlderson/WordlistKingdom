package generate

import (
	"unicode"

	"wordlistkingdom/pkg/textutil"
)

// phaseNumbers emits word+digits shapes such as Corp2026 and corp1234.
func (g *Generator) phaseNumbers() {
	g.phase = PhaseNumbers
	for _, t := range g.terms {
		for _, f := range g.allForms(t) {
			if !canAppendNumber(f) {
				continue
			}
			for _, n := range g.numbers {
				if !g.push(f + n) {
					return
				}
			}
		}
	}
}

// phaseDates emits birthday/founding-date shapes such as Corp02012000,
// corp02.01.2000 and corp2000!. Dates are only generated when -dates is given.
func (g *Generator) phaseDates() {
	g.phase = PhaseDates
	tokens := g.dateTokens()
	if len(tokens) == 0 {
		return
	}
	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, true) {
			if !canAppendNumber(f) {
				continue
			}
			for _, token := range tokens {
				if !g.cfg.Pattern.Special && !allDigits(token) {
					continue
				}
				if !g.push(f + token) {
					return
				}
				if g.cfg.Pattern.Special {
					if !g.push(f + token + "!") {
						return
					}
					if !g.push(f + "!" + token) {
						return
					}
				}
			}
		}
	}
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// phaseSepNumbers emits word+separator+digits shapes such as corp_24,
// corp.local.2026 and Corp@99. This is the pattern the original tool produced
// heavily, and it is genuinely common in corporate passwords.
func (g *Generator) phaseSepNumbers() {
	g.phase = PhaseSepNumbers
	seps := g.sepChars()
	if len(seps) == 0 {
		return
	}
	nums := g.sepNumberTokens()
	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, true) {
			if !canAppendNumber(f) {
				continue
			}
			for _, sep := range seps {
				for _, n := range nums {
					if !g.push(f + sep + n) {
						return
					}
				}
			}
		}
	}
}

// phaseSpecials emits word+symbol shapes such as Corp!, corp@ and admin#.
func (g *Generator) phaseSpecials() {
	g.phase = PhaseSpecials
	for _, t := range g.terms {
		for _, f := range g.allForms(t) {
			if !canAppendSpecial(f) {
				continue
			}
			for _, s := range g.specials {
				if !g.push(f + s) {
					return
				}
			}
		}
	}
}

// phaseYearSpecial emits year/symbol sandwiches in both orders:
// Corp2026!, Corp!2026, corp26@, corp@26.
func (g *Generator) phaseYearSpecial() {
	g.phase = PhaseYearSpecial
	years := g.yearSpecialTokens()
	chars := g.yearSpecialChars()
	if len(years) == 0 || len(chars) == 0 {
		return
	}
	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, true) {
			if !canAppendNumber(f) {
				continue
			}
			for _, y := range years {
				for _, s := range chars {
					if !g.push(f + y + s) {
						return
					}
					if !g.push(f + s + y) {
						return
					}
				}
			}
		}
	}
}

// phaseNumSpecial emits common number/symbol sandwiches in both orders:
// Corp123!, Corp!123, admin1@.
func (g *Generator) phaseNumSpecial() {
	g.phase = PhaseNumSpecial
	nums := g.numSpecialTokens()
	chars := g.numSpecialChars()
	if len(nums) == 0 || len(chars) == 0 {
		return
	}
	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, true) {
			if !canAppendNumber(f) {
				continue
			}
			for _, n := range nums {
				for _, s := range chars {
					if !g.push(f + n + s) {
						return
					}
					if !g.push(f + s + n) {
						return
					}
				}
			}
		}
	}
}

// canAppendNumber reports whether appending digits reads naturally. Bases that
// already end in a digit or a symbol are handled by other phases.
func canAppendNumber(s string) bool {
	r := textutil.LastRune(s)
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// canAppendSpecial reports whether appending a symbol reads naturally.
func canAppendSpecial(s string) bool {
	r := textutil.LastRune(s)
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
