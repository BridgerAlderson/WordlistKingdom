package generate

import (
	"fmt"
)

// buildYears returns the years used for numeric suffixes, in priority order.
// The default set is intentionally tiny: current and previous year cover most
// corporate passwords. Older/newer years are opt-in through -years.
func (g *Generator) buildYears() []int {
	years := []int{g.now, g.now - 1}
	switch {
	case g.cfg.Level == 2:
		years = append(years, g.now-2, g.now-3, g.now+1)
	case g.cfg.Level >= 3:
		years = append(years, g.now-2, g.now-3, g.now-4, g.now+1, g.now+2)
	case g.cfg.Level == 1:
		years = append(years, g.now-2)
	}
	years = appendUniqueInt(years, g.cfg.Years...)
	if g.cfg.Level >= 3 {
		// The original tool shipped a fixed year list; complete output keeps
		// those shapes, while focused/standard stay dynamic.
		years = appendUniqueInt(years, 1971, 1999, 2000, 2001, 2019, 2023, 2024)
	}
	for _, d := range g.cfg.Dates {
		years = appendUniqueInt(years, d.Year)
	}
	return years
}

// dateTokens expands configured dates into the digit and separator shapes
// people use for birthdays and founding dates.
func (g *Generator) dateTokens() []string {
	if len(g.cfg.Dates) == 0 {
		return nil
	}
	out := make([]string, 0, len(g.cfg.Dates)*9)
	for _, d := range g.cfg.Dates {
		out = append(out,
			fmt.Sprintf("%02d%02d%04d", d.Day, d.Month, d.Year),
			fmt.Sprintf("%02d%02d%02d", d.Day, d.Month, d.Year%100),
			fmt.Sprintf("%04d%02d%02d", d.Year, d.Month, d.Day),
			fmt.Sprintf("%02d%02d%02d", d.Year%100, d.Month, d.Day),
			fmt.Sprintf("%02d.%02d.%04d", d.Day, d.Month, d.Year),
			fmt.Sprintf("%02d-%02d-%04d", d.Day, d.Month, d.Year),
			fmt.Sprintf("%02d/%02d/%04d", d.Day, d.Month, d.Year),
			fmt.Sprintf("%02d%02d", d.Day, d.Month),
			fmt.Sprintf("%02d%02d", d.Month, d.Day),
		)
	}
	return dedupeStrings(out)
}

func (g *Generator) extraYearTokens() []string {
	var out []string
	add := func(year int) {
		out = append(out, fmt.Sprintf("%d", year))
		if short, ok := shortYear(year); ok {
			out = append(out, short)
		}
	}
	for _, y := range g.cfg.Years {
		add(y)
	}
	for _, d := range g.cfg.Dates {
		add(d.Year)
	}
	if g.cfg.Level >= 3 {
		for _, y := range []int{1971, 1999, 2000, 2001, 2019, 2023, 2024} {
			add(y)
		}
	}
	return dedupeStrings(out)
}

// buildNumbers combines year tokens with the curated numeric suffixes.
func (g *Generator) buildNumbers() []string {
	out := make([]string, 0, 24)
	for _, y := range g.years {
		out = append(out, fmt.Sprintf("%d", y))
		if short, ok := shortYear(y); ok {
			out = append(out, short)
		}
	}
	out = appendUnique(out, commonNumbers(g.cfg.Level)...)
	out = appendUnique(out, g.cfg.ExtraNumbers...)
	return out
}

// buildSpecials returns the special-character suffixes for this level.
func (g *Generator) buildSpecials() []string {
	out := []string{"!", "@", ".", "_", "*", "?"}
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "#", "$", "-", "%")
	}
	if g.cfg.Level >= 2 {
		out = appendUnique(out, "&", "+", "=")
	}
	if g.cfg.Level >= 3 {
		out = appendUnique(out, "~", "^", "|", ":", ";")
	}
	out = appendUnique(out, g.cfg.ExtraSpecials...)
	return out
}

// commonNumbers is the curated numeric suffix ladder. Order is priority order.
func commonNumbers(level int) []string {
	out := []string{"123", "1234", "12345", "1", "12", "99"}
	if level >= 1 {
		out = appendUnique(out, "0", "01", "007", "111", "321", "69", "100", "123456")
	}
	if level >= 2 {
		out = appendUnique(out, "000", "666", "777", "123456789")
	}
	return out
}

// sepNumberTokens are the numbers used in base+separator+number shapes such as
// corp_24, corp.local.2026 and Corp@99.
func (g *Generator) sepNumberTokens() []string {
	out := make([]string, 0, 16)
	// The first two configured years are the high-priority ones.
	limit := minInt(len(g.years), 2)
	if g.cfg.Level >= 1 {
		limit = minInt(len(g.years), 3)
	}
	for i := 0; i < limit; i++ {
		y := g.years[i]
		out = append(out, fmt.Sprintf("%d", y))
		if short, ok := shortYear(y); ok {
			out = append(out, short)
		}
	}
	// Explicit -years values and date years matter even at focused level.
	out = appendUnique(out, g.extraYearTokens()...)

	out = appendUnique(out, "123", "99")
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "1234", "1")
	}
	if g.cfg.Level >= 2 {
		out = appendUnique(out, "12", "12345")
	}
	out = appendUnique(out, g.cfg.ExtraNumbers...)
	return dedupeStrings(out)
}

// numSpecialTokens are numbers combined with symbols in both orders
// (Corp123!, Corp!123).
func (g *Generator) numSpecialTokens() []string {
	out := []string{"123", "1234", "99"}
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "12345", "1", "12")
	}
	if g.cfg.Level >= 2 {
		out = appendUnique(out, "99", "007", "2020")
	}
	out = appendUnique(out, g.cfg.ExtraNumbers...)
	return dedupeStrings(out)
}

// yearSpecialTokens are full and two-digit year tokens used for year+symbol
// shapes.
func (g *Generator) yearSpecialTokens() []string {
	limit := 2
	if g.cfg.Level == 1 {
		limit = 3
	}
	if g.cfg.Level >= 2 {
		limit = 4
	}
	limit = minInt(limit, len(g.years))
	out := make([]string, 0, limit*2+4)
	for i := 0; i < limit; i++ {
		y := g.years[i]
		out = append(out, fmt.Sprintf("%d", y))
		if short, ok := shortYear(y); ok {
			out = append(out, short)
		}
	}
	// Extra years supplied by the operator are always high priority.
	out = appendUnique(out, g.extraYearTokens()...)
	return dedupeStrings(out)
}

// yearSpecialChars are the symbols used in year+symbol shapes. The focused
// set mirrors the shapes that actually show up in the original tool and in
// breach data: ! * . and ?. The @ shape is covered by separated numbers.
func (g *Generator) yearSpecialChars() []string {
	out := []string{"!", "*", ".", "?"}
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "@", "#")
	}
	if g.cfg.Level >= 2 {
		out = appendUnique(out, "_")
	}
	return g.filterSpecials(out)
}

// numSpecialChars are the symbols used in number+symbol shapes such as
// Corp123! and Corp!123. Kept smaller than the year set because these shapes
// are less frequent.
func (g *Generator) numSpecialChars() []string {
	out := []string{"!", "*", ".", "?"}
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "@")
	}
	if g.cfg.Level >= 2 {
		out = appendUnique(out, "#")
	}
	return g.filterSpecials(out)
}

func (g *Generator) filterSpecials(in []string) []string {
	allowed := make(map[string]bool, len(g.specials))
	for _, s := range g.specials {
		allowed[s] = true
	}
	out := in[:0]
	for _, s := range in {
		if allowed[s] {
			out = append(out, s)
		}
	}
	return out
}

// pairNumbers are numeric suffixes applied to two-word combinations.
func (g *Generator) pairNumbers() []string {
	out := []string{fmt.Sprintf("%d", g.now), fmt.Sprintf("%02d", g.now%100), fmt.Sprintf("%d", g.now-1), "123"}
	if g.cfg.Level >= 3 {
		out = appendUnique(out, g.completeYearTokens()...)
	}
	if g.cfg.Level >= 1 {
		if short, ok := shortYear(g.now - 1); ok {
			out = appendUnique(out, short)
		}
		out = appendUnique(out, "1234", "99")
	}
	if g.cfg.Level >= 3 {
		out = appendUnique(out, "1", "12", "12345")
	}
	return dedupeStrings(out)
}

// pairSpecials are symbol suffixes applied to two-word combinations.
func (g *Generator) pairSpecials() []string {
	if g.cfg.Level >= 3 {
		return append([]string(nil), g.specials...)
	}
	out := []string{"!", "@"}
	if g.cfg.Level >= 1 {
		out = appendUnique(out, "#", ".")
	}
	return dedupeStrings(out)
}

// pairSeps are the separators used between two keywords. Separators are
// special characters, so they disappear when the ! class is disabled.
func (g *Generator) pairSeps() []string {
	out := []string{""}
	if !g.cfg.Pattern.Special {
		return out
	}
	candidates := []string{".", "_"}
	if g.cfg.Level >= 2 {
		candidates = append(candidates, "-", "@")
	}
	allowed := make(map[string]bool, len(g.specials))
	for _, s := range g.specials {
		allowed[s] = true
	}
	for _, s := range candidates {
		if s == "" || allowed[s] {
			out = append(out, s)
		}
	}
	return out
}

// sepChars are the separators used in base+sep+number shapes.
func (g *Generator) sepChars() []string {
	if !g.cfg.Pattern.Special {
		return nil
	}
	allowed := make(map[string]bool, len(g.specials))
	for _, s := range g.specials {
		allowed[s] = true
	}
	out := make([]string, 0, 4)
	for _, s := range []string{".", "_", "@", "-"} {
		if allowed[s] {
			out = append(out, s)
		}
	}
	return out
}

func shortYear(y int) (string, bool) {
	if y < 1950 || y > 2099 {
		return "", false
	}
	return fmt.Sprintf("%02d", y%100), true
}

func intsToStrings(in []int) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, fmt.Sprintf("%d", v))
	}
	return out
}

// Role words are used for service-account-style combinations (AdminCorp,
// Corp_Backup). They are deliberately small and generic.
var baseRoles = []string{
	"Admin", "Test", "Dev", "IT", "Support", "Backup", "Helpdesk", "Root",
}

var extendedRoles = []string{
	"Sys", "Sec", "Net", "VPN", "Mail", "Web", "Cloud", "Sistem",
}

// Seasons and months are English plus ASCII-encoded Turkish. They are only
// used at extended and deep levels because they are rare in corporate
// passwords compared with years and digits.
var seasons = []string{
	"Spring", "Summer", "Autumn", "Winter",
	"Bahar", "Yaz", "Sonbahar", "Kis",
}

var months = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
	"Ocak", "Subat", "Mart", "Nisan", "Mayis", "Haziran",
	"Temmuz", "Agustos", "Eylul", "Ekim", "Kasim", "Aralik",
}

func (g *Generator) roles() []string {
	out := append([]string(nil), baseRoles...)
	if g.cfg.Level >= 2 {
		out = append(out, extendedRoles...)
	}
	return out
}

func (g *Generator) seasonTokens() []string {
	out := append([]string(nil), seasons...)
	if g.cfg.Level >= 3 {
		out = append(out, months...)
	}
	return out
}

// hasSpecial reports whether s is in the configured special set.
func (g *Generator) hasSpecial(s string) bool {
	for _, v := range g.specials {
		if v == s {
			return true
		}
	}
	return false
}
