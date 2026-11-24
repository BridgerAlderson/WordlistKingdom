// Package policy applies output-length and character-class rules before a
// candidate reaches the wordlist. It intentionally has no notion of "common
// passwords": quality is decided by the generator; this package only rejects
// strings that cannot be used or that the operator explicitly excluded.
package policy

import (
	"unicode"
	"unicode/utf8"

	"wordlistkingdom/pkg/config"
)

// Reason describes why a candidate was rejected.
type Reason int

const (
	OK Reason = iota
	Empty
	TooShort
	TooLong
	Whitespace
	Control
	Separator
	NonASCII
	ADComplexity
	reasonCount
)

var reasonNames = [reasonCount]string{
	OK:           "ok",
	Empty:        "empty",
	TooShort:     "too short",
	TooLong:      "too long",
	Whitespace:   "whitespace",
	Control:      "control character",
	Separator:    "repeated separator",
	NonASCII:     "non-ASCII",
	ADComplexity: "AD complexity",
}

// String returns a short human-readable rejection reason.
func (r Reason) String() string {
	if r < 0 || r >= reasonCount {
		return "unknown"
	}
	return reasonNames[r]
}

// Count is the number of distinct reasons (for statistics arrays).
const Count = int(reasonCount)

// Policy is an immutable filter built from a config.
type Policy struct {
	minLen     int
	maxLen     int
	allowSpace bool
	asciiOnly  bool
	ad         bool
	adMinLen   int
}

// New builds a policy from cfg.
func New(cfg config.Config) *Policy {
	maxLen := cfg.MaxLen
	if maxLen == 0 {
		maxLen = 128
	}
	return &Policy{
		minLen:     cfg.MinLen,
		maxLen:     maxLen,
		allowSpace: cfg.AllowSpace,
		asciiOnly:  cfg.ASCIIOnly,
		ad:         cfg.ADPolicy,
		adMinLen:   cfg.ADMinLen,
	}
}

// Check validates one candidate and returns the first failing reason.
func (p *Policy) Check(s string) Reason {
	if s == "" {
		return Empty
	}

	n := utf8.RuneCountInString(s)
	if n < p.minLen {
		return TooShort
	}
	if n > p.maxLen {
		return TooLong
	}

	var upper, lower, digit, special bool
	prevSep := false
	first := true
	for _, r := range s {
		if unicode.IsControl(r) {
			return Control
		}
		if unicode.IsSpace(r) {
			if !p.allowSpace {
				return Whitespace
			}
			special = true
		}
		if p.asciiOnly && r > unicode.MaxASCII {
			return NonASCII
		}

		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		default:
			special = true
		}

		if isSeparator(r) {
			if first || prevSep {
				return Separator
			}
			prevSep = true
		} else {
			prevSep = false
		}
		first = false
	}

	if p.ad {
		if n < p.adMinLen {
			return ADComplexity
		}
		classes := 0
		for _, ok := range []bool{upper, lower, digit, special} {
			if ok {
				classes++
			}
		}
		// Active Directory complexity requires three of the four classes,
		// not all four. Requiring a symbol on every password is a common
		// source of missed candidates.
		if classes < 3 {
			return ADComplexity
		}
	}
	return OK
}

func isSeparator(r rune) bool {
	switch r {
	case '.', '_', '-', '@':
		return true
	}
	return false
}
