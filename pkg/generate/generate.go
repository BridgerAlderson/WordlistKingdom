// Package generate builds human-plausible password candidates from target
// keywords. It deliberately emits a prioritised, bounded set rather than a
// Cartesian product: breach-derived password shapes are dominated by
// word+digits, word+digits+symbol, capitalised words and simple two-word
// combinations. Everything is produced in a deterministic order so two runs on
// the same inputs and flags yield byte-identical wordlists.
package generate

import (
	"context"
	"strings"
	"time"

	"wordlistkingdom/pkg/config"
	"wordlistkingdom/pkg/textutil"
)

// Phase identifies a generation stage. The numeric order is also the output
// priority order: the most likely candidates are written first.
type Phase int

const (
	PhaseCore Phase = iota
	PhaseNumbers
	PhaseDates
	PhaseSepNumbers
	PhaseSpecials
	PhaseYearSpecial
	PhaseNumSpecial
	PhasePairs
	PhaseUsernames
	PhaseAffixes
	PhaseLeet
	PhaseSeasons
	PhaseDeep
	PhaseCount
)

var phaseNames = [PhaseCount]string{
	PhaseCore:        "core",
	PhaseNumbers:     "numbers",
	PhaseDates:       "dates",
	PhaseSepNumbers:  "separated numbers",
	PhaseSpecials:    "specials",
	PhaseYearSpecial: "year+special",
	PhaseNumSpecial:  "number+special",
	PhasePairs:       "word pairs",
	PhaseUsernames:   "usernames",
	PhaseAffixes:     "role affixes",
	PhaseLeet:        "leetspeak",
	PhaseSeasons:     "seasons/months",
	PhaseDeep:        "deep variants",
}

// String returns the stable phase label used in reports.
func (p Phase) String() string {
	if p < 0 || p >= PhaseCount {
		return "unknown"
	}
	return phaseNames[p]
}

// Stats counts generation attempts per phase. It counts offers to the sink,
// before policy/dedup/exclusion filtering.
type Stats struct {
	Attempts int64
	Phases   [PhaseCount]int64
}

// EmitFunc receives one candidate. Returning false stops generation (the sink
// uses this for output caps, write errors and cancellation).
type EmitFunc func(candidate string) bool

// User is an account record used by the username phase. First, Last and
// BirthYear are optional.
type User struct {
	Username  string
	First     string
	Last      string
	BirthYear int
}

// Generator holds the prepared keyword forms and curated variant sets.
type Generator struct {
	cfg   config.Config
	now   int
	terms []term
	users []User

	years    []int
	numbers  []string
	specials []string

	stats Stats
	phase Phase
	sink  EmitFunc
	ctx   context.Context
	stop  bool
}

type term struct {
	primary string
	forms   []string
	lower   []string
	title   []string
	upper   []string
}

// New prepares a generator from a plain username list.
func New(cfg config.Config, keywords, usernames []string) *Generator {
	users := make([]User, 0, len(usernames))
	for _, u := range usernames {
		users = append(users, User{Username: u})
	}
	return NewWithUsers(cfg, keywords, users)
}

// NewWithUsers prepares a generator with optional full names and birth years.
// Keywords must already be cleaned and deduplicated; usernames are normalised
// by the source package.
func NewWithUsers(cfg config.Config, keywords []string, users []User) *Generator {
	g := &Generator{cfg: cfg, now: time.Now().Year()}
	g.years = g.buildYears()
	g.numbers = g.buildNumbers()
	g.specials = g.buildSpecials()

	seen := make(map[string]bool, len(keywords))
	for _, kw := range keywords {
		t, ok := g.makeTerm(kw)
		if !ok {
			continue
		}
		key := strings.ToLower(textutil.FoldASCII(t.primary))
		if seen[key] {
			continue
		}
		seen[key] = true
		g.terms = append(g.terms, t)
	}

	seenUsers := make(map[string]bool, len(users))
	for _, u := range users {
		u.Username = strings.TrimSpace(u.Username)
		if u.Username == "" {
			continue
		}
		key := strings.ToLower(textutil.FoldASCII(u.Username))
		if seenUsers[key] {
			continue
		}
		seenUsers[key] = true
		u.First = strings.TrimSpace(u.First)
		u.Last = strings.TrimSpace(u.Last)
		g.users = append(g.users, u)
	}
	return g
}

// Terms returns the number of prepared keyword terms.
func (g *Generator) Terms() int { return len(g.terms) }

// Generate runs all phases in priority order and returns attempt statistics.
func (g *Generator) Generate(ctx context.Context, sink EmitFunc) Stats {
	g.ctx = ctx
	g.sink = sink
	g.stats = Stats{}
	g.stop = false
	if ctx != nil && ctx.Err() != nil {
		g.stop = true
		return g.stats
	}

	g.phase = PhaseCore
	for _, t := range g.terms {
		for _, f := range t.forms {
			if !g.push(f) {
				return g.stats
			}
		}
	}

	if g.cfg.Pattern.Digits {
		g.phaseNumbers()
		if g.stop {
			return g.stats
		}
		if len(g.cfg.Dates) > 0 {
			g.phaseDates()
			if g.stop {
				return g.stats
			}
		}
		if g.cfg.Pattern.Special {
			g.phaseSepNumbers()
			if g.stop {
				return g.stats
			}
		}
	}

	if g.cfg.Pattern.Special {
		g.phaseSpecials()
		if g.stop {
			return g.stats
		}
	}

	if g.cfg.Pattern.Digits && g.cfg.Pattern.Special {
		g.phaseYearSpecial()
		if g.stop {
			return g.stats
		}
		g.phaseNumSpecial()
		if g.stop {
			return g.stats
		}
	}

	if g.cfg.Perm >= 2 && g.cfg.PairLimit > 0 {
		g.phasePairs()
		if g.stop {
			return g.stats
		}
	}

	if len(g.users) > 0 {
		g.phaseUsers()
		if g.stop {
			return g.stats
		}
	}

	if g.cfg.Level >= 1 {
		g.phaseAffixes()
		if g.stop {
			return g.stats
		}
		g.phaseLeet()
		if g.stop {
			return g.stats
		}
	}

	if g.cfg.Level >= 2 {
		g.phaseSeasons()
		if g.stop {
			return g.stats
		}
	}

	if g.cfg.Perm >= 3 || g.cfg.Level >= 3 {
		g.phaseDeep()
	}
	return g.stats
}

// derivedSpellings extracts the spellings people actually type when a keyword
// is a domain, a compound or a phrase:
//
//	acme.local        -> acmelocal, acme
//	active_directory  -> activedirectory, active
//	Acme Corp         -> Acme (the joined form is already the primary)
//
// It deliberately does not split IP addresses, TLDs or "www." hosts.
func derivedSpellings(raw string) []string {
	if !strings.ContainsAny(raw, "._- ") {
		return nil
	}
	first := raw
	if i := strings.IndexAny(raw, "._- "); i >= 0 {
		first = raw[:i]
	}
	if strings.EqualFold(first, "www") || strings.EqualFold(first, "http") || strings.EqualFold(first, "https") {
		return nil
	}

	joined := strings.Map(func(r rune) rune {
		switch r {
		case '.', '_', '-', ' ':
			return -1
		}
		return r
	}, raw)

	var out []string
	if joined != raw && hasLetter(joined) && textutil.RuneLen(joined) >= 3 {
		out = append(out, joined)
	}
	if hasLetter(first) && textutil.RuneLen(first) >= 3 &&
		!strings.EqualFold(first, raw) && !strings.EqualFold(first, joined) {
		out = append(out, first)
	}
	return out
}

// makeTerm expands one raw keyword into spellings and pattern-filtered case
// forms. Multi-word input is joined ("Acme Corp" -> "AcmeCorp") unless spaces
// are explicitly allowed.
func (g *Generator) makeTerm(raw string) (term, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return term{}, false
	}
	joined, hasSpace := textutil.JoinWords(raw)
	primary := raw
	spellings := make([]string, 0, 3)
	if hasSpace {
		primary = joined
		spellings = append(spellings, joined)
		if g.cfg.AllowSpace {
			spellings = append(spellings, raw)
		}
	} else {
		spellings = append(spellings, raw)
	}
	if fold := textutil.FoldASCII(primary); fold != primary {
		spellings = append(spellings, fold)
	}
	for _, derived := range derivedSpellings(raw) {
		spellings = appendUnique(spellings, derived)
	}

	t := term{primary: primary}
	seen := make(map[string]bool, 8)
	for _, sp := range spellings {
		opts := textutil.Options{
			Turkish:      true,
			ForceTurkish: g.cfg.Turkish,
			Camel:        g.cfg.Level >= 2,
			Pascal:       g.cfg.Level >= 1,
		}
		for _, f := range textutil.Forms(sp, opts) {
			if !g.acceptForm(f) || seen[f] {
				continue
			}
			seen[f] = true
			t.forms = append(t.forms, f)
		}
	}
	if len(t.forms) == 0 {
		return term{}, false
	}

	for _, f := range t.forms {
		if textutil.IsLower(f) {
			t.lower = append(t.lower, f)
		}
		if f == textutil.Title(f) {
			t.title = append(t.title, f)
		}
		if f == textutil.Upper(f) {
			t.upper = append(t.upper, f)
		}
	}
	return t, true
}

// acceptForm applies the legacy class selector. The selector chooses mutation
// layers, not a strict character whitelist: a Title-case form belongs to the
// uppercase layer even though it contains lowercase letters.
func (g *Generator) acceptForm(s string) bool {
	p := g.cfg.Pattern
	hasUpper := textutil.HasUpper(s)
	if p.Lower && !hasUpper {
		return true
	}
	if p.Upper && hasUpper {
		return true
	}
	if !p.Lower && !p.Upper {
		return !hasUpper
	}
	return false
}

// push offers one candidate to the sink and records attempts.
func (g *Generator) push(s string) bool {
	if g.stop {
		return false
	}
	if s == "" {
		return true
	}
	g.stats.Attempts++
	g.stats.Phases[g.phase]++

	if g.stats.Attempts%4096 == 0 && g.ctx != nil {
		if err := g.ctx.Err(); err != nil {
			g.stop = true
			return false
		}
	}
	if g.sink == nil || !g.sink(s) {
		g.stop = true
		return false
	}
	return true
}

// formsForTier returns the case forms used by a suffix layer. Callers decide
// whether the UPPER forms are worth the extra candidates; lower and Title are
// always the first two.
func (g *Generator) formsForTier(t term, includeUpper bool) []string {
	out := make([]string, 0, len(t.lower)+len(t.title)+len(t.upper))
	out = append(out, t.lower...)
	out = append(out, t.title...)
	if includeUpper {
		out = append(out, t.upper...)
	}
	return dedupeStrings(out)
}

func (g *Generator) allForms(t term) []string {
	out := make([]string, 0, len(t.forms))
	out = append(out, t.lower...)
	out = append(out, t.title...)
	out = append(out, t.upper...)
	for _, f := range t.forms {
		out = append(out, f)
	}
	return dedupeStrings(out)
}

func appendUnique(dst []string, values ...string) []string {
	for _, v := range values {
		found := false
		for _, existing := range dst {
			if existing == v {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst
}

func dedupeStrings(in []string) []string {
	if len(in) < 2 {
		return in
	}
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func appendUniqueInt(dst []int, values ...int) []int {
	for _, v := range values {
		found := false
		for _, existing := range dst {
			if existing == v {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
