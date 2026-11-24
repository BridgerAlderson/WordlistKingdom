// Package config defines the runtime configuration shared by the CLI, the
// generator and the output policy. It has no dependencies outside the standard
// library so it can be used from tests without spinning up a pipeline.
package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Pattern is the legacy-compatible character-class selector.
//
//	q  lowercase words
//	@  uppercase / capitalised words
//	%  numeric suffixes
//	!  special-character suffixes
//
// An empty Pattern means "all classes". The selector chooses which mutation
// layers are generated; it is not a strict whitelist of characters in the
// final password.
type Pattern struct {
	Lower   bool
	Upper   bool
	Digits  bool
	Special bool
}

// AllClasses returns a pattern that enables every mutation layer.
func AllClasses() Pattern {
	return Pattern{Lower: true, Upper: true, Digits: true, Special: true}
}

// ParsePattern parses the four legacy class characters. An empty string means
// "all classes". Unknown characters are rejected instead of being ignored so a
// typo cannot silently shrink the wordlist.
func ParsePattern(s string) (Pattern, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return AllClasses(), nil
	}

	p := Pattern{}
	for _, r := range s {
		switch r {
		case 'q':
			p.Lower = true
		case '@':
			p.Upper = true
		case '%':
			p.Digits = true
		case '!':
			p.Special = true
		case ' ', '\t', ',', '|':
			// Tolerate separators users commonly type between classes.
		default:
			return Pattern{}, fmt.Errorf("invalid pattern character %q (valid: q, @, %%, !)", r)
		}
	}
	if !p.Lower && !p.Upper && !p.Digits && !p.Special {
		return Pattern{}, fmt.Errorf("pattern %q enables no classes", s)
	}
	return p, nil
}

// String renders a stable canonical form of the pattern.
func (p Pattern) String() string {
	var b strings.Builder
	if p.Lower {
		b.WriteByte('q')
	}
	if p.Upper {
		b.WriteByte('@')
	}
	if p.Digits {
		b.WriteByte('%')
	}
	if p.Special {
		b.WriteByte('!')
	}
	if b.Len() == 4 {
		return "q@%! (all classes)"
	}
	if b.Len() == 0 {
		return "none"
	}
	return b.String()
}

// Any reports whether at least one class is enabled.
func (p Pattern) Any() bool {
	return p.Lower || p.Upper || p.Digits || p.Special
}

// Date is a target-specific date used to build birthday/founding-date shapes.
type Date struct {
	Year  int
	Month int
	Day   int
}

// Config is the complete, validated description of one generator run.
type Config struct {
	Input    string
	Output   string
	Userlist string
	Userdata string

	Level int // 0 focused, 1 standard, 2 extended, 3 deep
	Perm  int // maximum terms per candidate: 1, 2 or 3

	Pattern Pattern

	MinLen   int // minimum output length in characters
	MaxLen   int // maximum output length in characters (0 = hard cap)
	ADPolicy bool
	ADMinLen int

	ASCIIOnly  bool
	AllowSpace bool
	Turkish    bool

	Years         []int
	Dates         []Date
	ExtraNumbers  []string
	ExtraSpecials []string
	Exclude       []string

	PairLimit        int
	TripleLimit      int
	UserKeywordLimit int

	MaxCandidates int64

	Verbose  bool
	Quiet    bool
	Progress bool
}

// Default returns the defaults used by the CLI. They are deliberately
// conservative: a focused wordlist for a small keyword file should contain
// hundreds of candidates, not hundreds of thousands.
func Default() Config {
	return Config{
		Input:            "keywords.txt",
		Output:           "custom_wordlist.txt",
		Level:            0,
		Perm:             2,
		Pattern:          AllClasses(),
		MinLen:           1,
		MaxLen:           32,
		ADMinLen:         8,
		PairLimit:        25,
		TripleLimit:      8,
		UserKeywordLimit: 3,
		MaxCandidates:    5_000_000,
	}
}

// Validate checks every field that can be checked without touching the
// filesystem. It returns the first problem with enough context to fix it.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Input) == "" {
		return fmt.Errorf("input path is empty")
	}
	if strings.TrimSpace(c.Output) == "" {
		return fmt.Errorf("output path is empty")
	}
	if c.Level < 0 || c.Level > 3 {
		return fmt.Errorf("level must be 0..3 (got %d)", c.Level)
	}
	if c.Perm < 1 || c.Perm > 3 {
		return fmt.Errorf("perm must be 1..3 (got %d)", c.Perm)
	}
	if !c.Pattern.Any() {
		return fmt.Errorf("pattern enables no character classes")
	}
	if c.MinLen < 1 {
		return fmt.Errorf("min length must be at least 1 (got %d)", c.MinLen)
	}
	if c.MaxLen == 0 {
		c.MaxLen = 128
	}
	if c.MaxLen < c.MinLen {
		return fmt.Errorf("max length (%d) is smaller than min length (%d)", c.MaxLen, c.MinLen)
	}
	if c.MaxLen > 128 {
		return fmt.Errorf("max length cannot exceed 128 (got %d)", c.MaxLen)
	}
	if c.ADMinLen < 1 || c.ADMinLen > 128 {
		return fmt.Errorf("AD minimum length must be 1..128 (got %d)", c.ADMinLen)
	}
	if c.ADPolicy && c.ADMinLen > c.MaxLen {
		return fmt.Errorf("AD minimum length (%d) exceeds max length (%d)", c.ADMinLen, c.MaxLen)
	}
	if c.PairLimit < 0 || c.TripleLimit < 0 || c.UserKeywordLimit < 0 {
		return fmt.Errorf("pair/triple/user keyword limits cannot be negative")
	}
	if c.MaxCandidates < 0 {
		return fmt.Errorf("max candidates cannot be negative")
	}
	for _, y := range c.Years {
		if y < 1000 || y > 2999 {
			return fmt.Errorf("year %d is outside 1000..2999", y)
		}
	}
	for _, d := range c.Dates {
		if d.Year < 1900 || d.Year > 2999 || d.Month < 1 || d.Month > 12 || d.Day < 1 || d.Day > 31 {
			return fmt.Errorf("invalid date %04d-%02d-%02d", d.Year, d.Month, d.Day)
		}
		t := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
		if t.Year() != d.Year || int(t.Month()) != d.Month || t.Day() != d.Day {
			return fmt.Errorf("invalid date %04d-%02d-%02d", d.Year, d.Month, d.Day)
		}
	}
	for _, n := range c.ExtraNumbers {
		if n == "" || len(n) > 8 {
			return fmt.Errorf("invalid numeric suffix %q (1..8 digits)", n)
		}
		for _, r := range n {
			if r < '0' || r > '9' {
				return fmt.Errorf("invalid numeric suffix %q (digits only)", n)
			}
		}
	}
	for _, s := range c.ExtraSpecials {
		if s == "" || len([]rune(s)) > 4 {
			return fmt.Errorf("invalid special suffix %q (1..4 characters)", s)
		}
		for _, r := range s {
			if unicode.IsSpace(r) || unicode.IsControl(r) {
				return fmt.Errorf("special suffix %q contains whitespace or control characters", s)
			}
		}
	}
	return nil
}

// ParseDates parses comma-separated target dates. Accepts
// YYYY-MM-DD, YYYY/MM/DD, YYYY.MM.DD, DD-MM-YYYY, DD/MM/YYYY, DD.MM.YYYY,
// YYYYMMDD and DDMMYYYY. Invalid calendar dates are rejected.
func ParseDates(s string) ([]Date, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	seen := make(map[Date]bool)
	var out []Date
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		d, err := parseDate(part)
		if err != nil {
			return nil, err
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out, nil
}

func parseDate(s string) (Date, error) {
	if len(s) == 8 && allDigits(s) {
		if y, m, d, ok := dateFromDigits(s); ok {
			return Date{Year: y, Month: m, Day: d}, nil
		}
		return Date{}, fmt.Errorf("invalid date %q (use YYYY-MM-DD or DD.MM.YYYY)", s)
	}

	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '/' || r == '.'
	})
	if len(fields) != 3 {
		return Date{}, fmt.Errorf("invalid date %q (use YYYY-MM-DD or DD.MM.YYYY)", s)
	}

	var year, month, day int
	var err error
	switch {
	case len(fields[0]) == 4:
		year, err = strconv.Atoi(fields[0])
		if err == nil {
			month, err = strconv.Atoi(fields[1])
		}
		if err == nil {
			day, err = strconv.Atoi(fields[2])
		}
	case len(fields[2]) == 4:
		day, err = strconv.Atoi(fields[0])
		if err == nil {
			month, err = strconv.Atoi(fields[1])
		}
		if err == nil {
			year, err = strconv.Atoi(fields[2])
		}
	default:
		return Date{}, fmt.Errorf("invalid date %q (one side must be a four-digit year)", s)
	}
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q", s)
	}
	if year < 1900 || year > 2999 {
		return Date{}, fmt.Errorf("invalid date %q (year must be 1900..2999)", s)
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return Date{}, fmt.Errorf("invalid calendar date %q", s)
	}
	return Date{Year: year, Month: month, Day: day}, nil
}

func dateFromDigits(s string) (year, month, day int, ok bool) {
	for _, candidate := range []struct {
		y, m, d string
	}{
		{s[0:4], s[4:6], s[6:8]}, // YYYYMMDD
		{s[4:8], s[2:4], s[0:2]}, // DDMMYYYY
	} {
		y, _ := strconv.Atoi(candidate.y)
		m, _ := strconv.Atoi(candidate.m)
		d, _ := strconv.Atoi(candidate.d)
		if y < 1900 || y > 2999 {
			continue
		}
		t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
		if t.Year() == y && int(t.Month()) == m && t.Day() == d {
			return y, m, d, true
		}
	}
	return 0, 0, 0, false
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// ParseYears parses a comma-separated list of four-digit years.
func ParseYears(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	seen := make(map[int]bool)
	var out []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		y, err := strconv.Atoi(part)
		if err != nil || y < 1000 || y > 2999 {
			return nil, fmt.Errorf("invalid year %q (use four digits, e.g. 1923)", part)
		}
		if !seen[y] {
			seen[y] = true
			out = append(out, y)
		}
	}
	return out, nil
}

// ParseNumbers parses a comma-separated list of numeric suffixes.
func ParseNumbers(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	seen := make(map[string]bool)
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(part) > 8 {
			return nil, fmt.Errorf("numeric suffix %q is longer than 8 digits", part)
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return nil, fmt.Errorf("numeric suffix %q must contain digits only", part)
			}
		}
		if !seen[part] {
			seen[part] = true
			out = append(out, part)
		}
	}
	return out, nil
}

// ParseSpecials parses special suffixes. A comma-separated form allows
// multi-character tokens ("!,!123"); the bare form splits into characters
// ("!@#._").
func ParseSpecials(s string) ([]string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var parts []string
	if strings.Contains(s, ",") {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				parts = append(parts, part)
			}
		}
	} else {
		for _, r := range strings.TrimSpace(s) {
			if unicode.IsSpace(r) {
				continue
			}
			parts = append(parts, string(r))
		}
	}

	seen := make(map[string]bool)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if len([]rune(p)) > 4 {
			return nil, fmt.Errorf("special suffix %q is longer than 4 characters", p)
		}
		for _, r := range p {
			if unicode.IsSpace(r) || unicode.IsControl(r) {
				return nil, fmt.Errorf("special suffix %q contains whitespace or control characters", p)
			}
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out, nil
}

// ParsePaths parses comma-separated filesystem paths.
func ParsePaths(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
