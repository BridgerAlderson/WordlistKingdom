package config

import (
	"testing"
)

func FuzzParseDates(f *testing.F) {
	for _, seed := range []string{
		"2000-01-02",
		"17.05.1990",
		"1999/12/31",
		"20200102",
		"01022000",
		"2000-02-30",
		"not-a-date",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		dates, err := ParseDates(s)
		if err != nil {
			return
		}
		for _, d := range dates {
			if d.Year < 1900 || d.Year > 2999 || d.Month < 1 || d.Month > 12 || d.Day < 1 || d.Day > 31 {
				t.Fatalf("accepted invalid date: %#v", d)
			}
		}
	})
}

func FuzzParsePattern(f *testing.F) {
	for _, seed := range []string{"", "q", "q@%!", "q%", "q @ % !", "x"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ParsePattern(s)
	})
}

func FuzzParseSpecials(f *testing.F) {
	for _, seed := range []string{"!@#._", "!,!123", "", " ", "abc"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ParseSpecials(s)
	})
}
