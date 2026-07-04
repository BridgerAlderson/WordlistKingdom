package filter

import "unicode"

type PolicyFilter struct {
	minLen  int
	enabled bool
}

func NewPolicyFilter(minLen int, enabled bool) *PolicyFilter {
	return &PolicyFilter{minLen: minLen, enabled: enabled}
}

func (pf *PolicyFilter) Passes(password string) bool {
	if !pf.enabled {
		return true
	}
	if len(password) < pf.minLen {
		return false
	}
	var upper, digit, special bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			special = true
		}
		if upper && digit && special {
			return true
		}
	}
	return upper && digit && special
}
