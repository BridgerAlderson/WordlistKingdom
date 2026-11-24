package source

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadKeywords(t *testing.T) {
	in := "\ufeff# comment\r\n" +
		"Acme\r\n" +
		"\"Acme Corp\"\r\n" +
		"acme\r\n" +
		"corp.local\r\n" +
		"\r\n" +
		"  Network  \r\n"

	list, st, err := Read(strings.NewReader(in), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Lines != 7 {
		t.Fatalf("Lines = %d, want 7", st.Lines)
	}
	if st.Duplicates != 1 {
		t.Fatalf("Duplicates = %d, want 1", st.Duplicates)
	}
	want := []string{"Acme", "Acme Corp", "corp.local", "Network"}
	if len(list) != len(want) {
		t.Fatalf("list = %#v, want %#v", list, want)
	}
	for i := range want {
		if list[i] != want[i] {
			t.Errorf("list[%d] = %q, want %q", i, list[i], want[i])
		}
	}
}

func TestReadUsernames(t *testing.T) {
	in := "CORP\\jsmith\njsmith@corp.local\nJSMITH\njohn.smith\n"
	list, st, err := Read(strings.NewReader(in), Options{Usernames: true})
	if err != nil {
		t.Fatal(err)
	}
	if st.Duplicates != 2 {
		t.Fatalf("Duplicates = %d, want 2", st.Duplicates)
	}
	want := []string{"jsmith", "john.smith"}
	if len(list) != len(want) {
		t.Fatalf("list = %#v", list)
	}
	for i := range want {
		if list[i] != want[i] {
			t.Errorf("list[%d] = %q, want %q", i, list[i], want[i])
		}
	}
}

func TestReadSkipsControlAndLong(t *testing.T) {
	in := "corp\nbad\x00line\n" + strings.Repeat("x", 100) + "\nok\n"
	list, st, err := Read(strings.NewReader(in), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Skipped != 2 {
		t.Fatalf("Skipped = %d, want 2", st.Skipped)
	}
	if len(list) != 2 || list[0] != "corp" || list[1] != "ok" {
		t.Fatalf("list = %#v", list)
	}
}

func TestReadGzipFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "words.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write([]byte("corp\ncorp.local\n")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	list, _, err := ReadFile(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %#v", list)
	}
}

func TestForEachLinePreservesPasswordSpaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exclude.txt")
	if err := os.WriteFile(path, []byte("\ufeff  spaced  \r\nplain\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got []string
	n, err := ForEachLine(path, func(line string) error {
		got = append(got, line)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || len(got) != 2 {
		t.Fatalf("n=%d got=%#v", n, got)
	}
	if got[0] != "  spaced  " {
		t.Fatalf("spaces not preserved: %q", got[0])
	}
}

func TestNormalizeUsername(t *testing.T) {
	cases := map[string]string{
		`CORP\jsmith`:       "jsmith",
		`CORP/jsmith`:       "jsmith",
		"jsmith@corp.local": "jsmith",
		"jsmith":            "jsmith",
	}
	for in, want := range cases {
		if got := NormalizeUsername(in); got != want {
			t.Errorf("NormalizeUsername(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadUserdataHeader(t *testing.T) {
	in := "username,first,last,birthyear\n" +
		"jsmith,John,Smith,1990\n" +
		"CORP\\asmith,Alice,Smith,\n" +
		"jsmith,Duplicate,Record,1991\n"
	users, st, err := ReadUserdata(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("users = %#v", users)
	}
	if users[0].Username != "jsmith" || users[0].First != "John" || users[0].Last != "Smith" || users[0].BirthYear != 1990 {
		t.Fatalf("users[0] = %#v", users[0])
	}
	if users[1].Username != "asmith" || users[1].BirthYear != 0 {
		t.Fatalf("users[1] = %#v", users[1])
	}
	if st.Duplicates != 1 {
		t.Fatalf("duplicates = %d", st.Duplicates)
	}
}

func TestReadUserdataHeaderlessAndAliases(t *testing.T) {
	users, _, err := ReadUserdata(strings.NewReader("jsmith,John,Smith,1990\nasmith\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Username != "jsmith" || users[0].First != "John" {
		t.Fatalf("users = %#v", users)
	}
	if users[1].Username != "asmith" || users[1].First != "" {
		t.Fatalf("users = %#v", users)
	}

	users, _, err = ReadUserdata(strings.NewReader("login,givenname,surname,born\njsmith,John,Smith,1990\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].BirthYear != 1990 || users[0].Last != "Smith" {
		t.Fatalf("users = %#v", users)
	}
}

func TestReadUserdataBOM(t *testing.T) {
	users, _, err := ReadUserdata(strings.NewReader("\ufeffusername,first,last,birthyear\njsmith,John,Smith,1990\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Username != "jsmith" || users[0].BirthYear != 1990 {
		t.Fatalf("users = %#v", users)
	}
}

func TestReadUserdataInvalidBirthYear(t *testing.T) {
	if _, _, err := ReadUserdata(strings.NewReader("jsmith,John,Smith,not-a-year\n")); err == nil {
		t.Fatal("expected invalid birth year error")
	}
	if _, _, err := ReadUserdata(strings.NewReader("jsmith,John,Smith,1700\n")); err == nil {
		t.Fatal("expected out-of-range birth year error")
	}
}
