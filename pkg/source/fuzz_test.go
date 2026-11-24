package source

import (
	"strings"
	"testing"
)

func FuzzReadKeywords(f *testing.F) {
	for _, seed := range []string{
		"corp\ncorp.local\n",
		"\ufeff# comment\nAcme\r\n",
		"\"quoted word\"\n",
		"bad\x00line\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		list, _, err := Read(strings.NewReader(s), Options{})
		if err != nil {
			return
		}
		for _, item := range list {
			if item == "" {
				t.Fatal("empty keyword accepted")
			}
		}
	})
}

func FuzzReadUserdata(f *testing.F) {
	for _, seed := range []string{
		"username,first,last,birthyear\njsmith,John,Smith,1990\n",
		"jsmith,John,Smith,1990\n",
		"jsmith\n",
		"jsmith,John,Smith,not-a-year\n",
		"\ufeffusername,first,last,birthyear\njsmith,John,Smith,1990\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		users, _, err := ReadUserdata(strings.NewReader(s))
		if err != nil {
			return
		}
		for _, u := range users {
			if u.Username == "" {
				t.Fatal("empty username accepted")
			}
			if u.BirthYear != 0 && (u.BirthYear < 1900 || u.BirthYear > 2100) {
				t.Fatalf("accepted birth year %d", u.BirthYear)
			}
		}
	})
}
