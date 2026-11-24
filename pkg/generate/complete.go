package generate

import (
	"fmt"

	"wordlistkingdom/pkg/textutil"
)

// completeYearTokens returns the full and two-digit forms of every configured
// year.
func (g *Generator) completeYearTokens() []string {
	out := make([]string, 0, len(g.years)*2)
	for _, y := range g.years {
		out = append(out, fmt.Sprintf("%d", y))
		if short, ok := shortYear(y); ok {
			out = append(out, short)
		}
	}
	return dedupeStrings(out)
}

// completeHistoricalYearTokens is the subset used for special+year prefixes
// at complete level (for example *1999Corp). Keeping it smaller than the full
// year list prevents complete output from exploding for large keyword files.
func (g *Generator) completeHistoricalYearTokens() []string {
	out := make([]string, 0, 16)
	for _, y := range []int{g.now, g.now - 1, 1971, 1999, 2000, 2001, 2019, 2023, 2024} {
		out = append(out, fmt.Sprintf("%d", y))
		if short, ok := shortYear(y); ok {
			out = append(out, short)
		}
	}
	return dedupeStrings(out)
}

// completeNumberTokens are the short numeric prefixes the original tool used
// (01, 23, 24, 99, 123).
func (g *Generator) completeNumberTokens() []string {
	out := []string{"01", "23", "24", "99", "123"}
	if short, ok := shortYear(g.now); ok {
		out = appendUnique(out, short)
	}
	if short, ok := shortYear(g.now - 1); ok {
		out = appendUnique(out, short)
	}
	return dedupeStrings(out)
}

// completePairPrefixes are the prefixes applied to pair candidates at
// complete level: symbols, years, numbers, symbol+number and a bounded
// symbol+year set.
func (g *Generator) completePairPrefixes() []string {
	out := make([]string, 0, 128)
	out = appendUnique(out, g.specials...)
	out = appendUnique(out, g.completeYearTokens()...)
	out = appendUnique(out, g.completeNumberTokens()...)
	for _, s := range g.specials {
		for _, n := range g.completeNumberTokens() {
			out = appendUnique(out, s+n)
		}
	}
	for _, s := range []string{"!", "%", "*", "$"} {
		for _, y := range g.completeHistoricalYearTokens() {
			out = appendUnique(out, s+y)
		}
	}
	return dedupeStrings(out)
}

// phaseComplete emits the prefix-symbol / prefix-year / prefix-number shapes
// that the original wordlist_generator.exe produced. It runs only at level 3
// (complete) and never in the focused profile.
func (g *Generator) phaseComplete() {
	g.phase = PhaseComplete

	years := g.completeYearTokens()
	nums := g.completeNumberTokens()

	for _, t := range g.terms {
		for _, f := range g.formsForTier(t, true) {
			if g.cfg.Pattern.Special {
				for _, s := range g.specials {
					if !g.push(s + f) {
						return
					}
					for _, n := range nums {
						if !g.push(s + n + f) {
							return
						}
					}
					for _, y := range years {
						if !g.push(s + y + f) {
							return
						}
					}
					for _, suffix := range []string{"!", "*", "?"} {
						if s != suffix && !g.push(s+f+suffix) {
							return
						}
					}
				}
			}
			if g.cfg.Pattern.Digits {
				for _, y := range years {
					if !g.push(y + f) {
						return
					}
					if g.cfg.Pattern.Special {
						for _, suffix := range []string{"!", "*", "%", "?", "."} {
							if !g.push(y + f + suffix) {
								return
							}
						}
					}
				}
				for _, n := range nums {
					if !g.push(n + f) {
						return
					}
					if g.cfg.Pattern.Special && !g.push(n+f+"!") {
						return
					}
				}
			}
		}
	}
}

// completePairEligible limits prefix application to the pair bases that match
// the shapes seen in the original output (Corp.local.Corp, CORP-CORP.LOCAL,
// Corp.local-Corp, ...). Applying the full prefix ladder to every mixed base
// would make complete output unusable on large keyword files.
func (g *Generator) completePairEligible(a, b, sep, base string) bool {
	wa := textutil.Title(a)
	wb := textutil.Title(b)
	if base == wa+sep+wb || base == wa+sep+textutil.Upper(b) {
		return true
	}
	if base == textutil.Upper(a)+sep+wb || base == wa+sep+textutil.Lower(b) {
		return true
	}
	return false
}

// completePairExtras emits a bounded set of prefix and prefix+suffix
// combinations for an eligible pair base.
func (g *Generator) completePairExtras(base string) bool {
	specials := []string{"!", "$", "%", "*", ".", "_"}
	years := g.completeHistoricalYearTokens()
	nums := g.completeNumberTokens()

	if g.cfg.Pattern.Special || g.cfg.Pattern.Digits {
		for _, s := range specials {
			if !g.push(s + base) {
				return false
			}
		}
		for _, y := range years {
			if !g.push(y + base) {
				return false
			}
		}
		for _, n := range nums {
			if !g.push(n + base) {
				return false
			}
		}
		for _, s := range []string{"%", "$", "!", "*"} {
			for _, n := range []string{"01", "23", "24"} {
				if !g.push(s + n + base) {
					return false
				}
			}
		}
		for _, s := range []string{"*", "!", "$", "%"} {
			for _, y := range []string{"1999", "2019", "2023", "2024"} {
				if !g.push(s + y + base) {
					return false
				}
			}
		}
	}

	if g.cfg.Pattern.Special {
		for _, y := range years {
			for _, s := range []string{"!", "*", "-", "."} {
				if !g.push(y + base + s) {
					return false
				}
			}
		}
		for _, s := range []string{"!", "$", "%", "*"} {
			for _, y := range []string{"71", "2023", "2024", fmt.Sprintf("%d", g.now)} {
				if !g.push(s + base + y) {
					return false
				}
			}
		}
	}

	if g.cfg.Pattern.Digits && g.cfg.Pattern.Special {
		for _, n := range []string{"123", "99", "2024", "24"} {
			for _, s := range []string{"!", "*", "?"} {
				if !g.push(base + n + s) {
					return false
				}
				if !g.push(base + s + n) {
					return false
				}
			}
		}
		for _, p := range nums {
			for _, n := range []string{"123", "99"} {
				if !g.push(p + base + n + "!") {
					return false
				}
			}
		}
	}
	return true
}
