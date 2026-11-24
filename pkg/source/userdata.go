package source

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"wordlistkingdom/pkg/textutil"
)

// User is one account record from a user-data CSV. First, Last and BirthYear
// are optional; a bare username list is still valid.
type User struct {
	Username  string
	First     string
	Last      string
	BirthYear int
}

// ReadUserdataFile reads a user-data CSV, optionally gzip-compressed.
//
// Accepted layouts:
//
//	username,first,last,birthyear
//	jsmith,John,Smith,1990
//
// or, without a header:
//
//	jsmith,John,Smith,1990
//	asmith,Alice,Smith
//
// A single-column file is treated as a plain username list. The header may
// use common aliases (user/login/sam, firstname/givenname, lastname/surname,
// birth/year/born).
func ReadUserdataFile(path string) ([]User, Stats, error) {
	rc, err := Open(path)
	if err != nil {
		return nil, Stats{}, err
	}
	defer rc.Close()
	users, st, err := ReadUserdata(rc)
	if err != nil {
		return nil, st, fmt.Errorf("%s: %w", path, err)
	}
	return users, st, nil
}

// ReadUserdata parses user data from r.
func ReadUserdata(r io.Reader) ([]User, Stats, error) {
	var st Stats
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	cr.LazyQuotes = true

	rows, err := cr.ReadAll()
	if err != nil {
		return nil, st, fmt.Errorf("csv: %w", err)
	}
	if len(rows) > 0 && len(rows[0]) > 0 {
		rows[0][0] = strings.TrimPrefix(rows[0][0], "\ufeff")
	}

	idxUser, idxFirst, idxLast, idxBirth := 0, 1, 2, 3
	start := 0
	if len(rows) > 0 && isUserHeader(rows[0]) {
		var ok bool
		idxUser, idxFirst, idxLast, idxBirth, ok = userHeaderIndex(rows[0])
		if !ok {
			return nil, st, fmt.Errorf("header must contain a username column")
		}
		start = 1
	}

	seen := make(map[string]bool)
	out := make([]User, 0, len(rows))
	for rowNo, row := range rows[start:] {
		st.Lines++
		if len(row) == 0 {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(row[0]), "#") {
			st.Comments++
			continue
		}

		get := func(i int) string {
			if i < 0 || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}

		username := NormalizeUsername(get(idxUser))
		if username == "" || !utf8.ValidString(username) {
			st.Skipped++
			continue
		}
		key := strings.ToLower(textutil.FoldASCII(username))
		if seen[key] {
			st.Duplicates++
			continue
		}

		u := User{
			Username: username,
			First:    cleanName(get(idxFirst)),
			Last:     cleanName(get(idxLast)),
		}
		if raw := get(idxBirth); raw != "" {
			year, err := strconv.Atoi(raw)
			if err != nil || year < 1900 || year > 2100 {
				return nil, st, fmt.Errorf("row %d: invalid birth year %q", rowNo+start+1, raw)
			}
			u.BirthYear = year
		}

		seen[key] = true
		out = append(out, u)
		st.Kept++
	}
	return out, st, nil
}

func isUserHeader(row []string) bool {
	for _, cell := range row {
		switch normalizeHeader(cell) {
		case "username", "samaccountname":
			return true
		}
	}
	// Single-column values like "user" or "login" are valid usernames; only
	// treat the shorter aliases as headers when there are other columns.
	if len(row) > 1 {
		for _, cell := range row {
			switch normalizeHeader(cell) {
			case "user", "login", "sam", "account":
				return true
			}
		}
	}
	return false
}

func userHeaderIndex(row []string) (user, first, last, birth int, ok bool) {
	user, first, last, birth = -1, -1, -1, -1
	for i, cell := range row {
		switch normalizeHeader(cell) {
		case "username", "user", "login", "sam", "samaccountname", "account":
			if user < 0 {
				user = i
			}
		case "first", "firstname", "givenname", "name":
			if first < 0 {
				first = i
			}
		case "last", "lastname", "surname", "familyname":
			if last < 0 {
				last = i
			}
		case "birth", "birthyear", "year", "born", "birthdate":
			if birth < 0 {
				birth = i
			}
		}
	}
	return user, first, last, birth, user >= 0
}

func normalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("_", "", "-", "", " ", "").Replace(s)
	return s
}

func cleanName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	if textutil.RuneLen(s) > 64 || hasControl(s) || !utf8.ValidString(s) {
		return ""
	}
	return s
}
