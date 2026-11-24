package config

import (
	"reflect"
	"testing"
)

func TestParsePattern(t *testing.T) {
	all, err := ParsePattern("")
	if err != nil || !all.Lower || !all.Upper || !all.Digits || !all.Special {
		t.Fatalf("empty pattern = %+v, %v", all, err)
	}
	p, err := ParsePattern("q%")
	if err != nil || !p.Lower || !p.Digits || p.Upper || p.Special {
		t.Fatalf("q%% = %+v, %v", p, err)
	}
	p, err = ParsePattern("q @ % !")
	if err != nil || !p.Lower || !p.Upper || !p.Digits || !p.Special {
		t.Fatalf("spaced pattern = %+v, %v", p, err)
	}
	if _, err := ParsePattern("x"); err == nil {
		t.Fatal("expected invalid pattern error")
	}
}

func TestPatternString(t *testing.T) {
	if got := AllClasses().String(); got != "q@%! (all classes)" {
		t.Fatalf("all = %q", got)
	}
	p, _ := ParsePattern("q%")
	if got := p.String(); got != "q%" {
		t.Fatalf("q%% = %q", got)
	}
}

func TestParseYears(t *testing.T) {
	got, err := ParseYears("1923, 1974,1923")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []int{1923, 1974}) {
		t.Fatalf("years = %#v", got)
	}
	if _, err := ParseYears("23"); err == nil {
		t.Fatal("expected invalid year error")
	}
}

func TestParseDates(t *testing.T) {
	got, err := ParseDates("2000-01-02, 17.05.1990, 1999/12/31, 20200102, 01022000")
	if err != nil {
		t.Fatal(err)
	}
	want := []Date{
		{Year: 2000, Month: 1, Day: 2},
		{Year: 1990, Month: 5, Day: 17},
		{Year: 1999, Month: 12, Day: 31},
		{Year: 2020, Month: 1, Day: 2},
		{Year: 2000, Month: 2, Day: 1}, // DDMMYYYY fallback
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dates = %#v, want %#v", got, want)
	}
	for _, bad := range []string{"2000-02-30", "2000-13-01", "0000.01.1", "17/05/90", "not-a-date"} {
		if _, err := ParseDates(bad); err == nil {
			t.Errorf("ParseDates(%q) succeeded, want error", bad)
		}
	}
}

func TestParseNumbers(t *testing.T) {
	got, err := ParseNumbers("123,007, 1881")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"123", "007", "1881"}) {
		t.Fatalf("numbers = %#v", got)
	}
	if _, err := ParseNumbers("12a"); err == nil {
		t.Fatal("expected invalid number error")
	}
}

func TestParseSpecials(t *testing.T) {
	got, err := ParseSpecials("!@#._")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("specials = %#v", got)
	}
	got, err = ParseSpecials("!,!123")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"!", "!123"}) {
		t.Fatalf("token specials = %#v", got)
	}
}

func TestParsePaths(t *testing.T) {
	got := ParsePaths(" rockyou.txt , , extra.txt")
	if !reflect.DeepEqual(got, []string{"rockyou.txt", "extra.txt"}) {
		t.Fatalf("paths = %#v", got)
	}
}

func TestValidate(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatalf("default invalid: %v", err)
	}

	bad := Default()
	bad.MinLen = 20
	bad.MaxLen = 10
	if err := bad.Validate(); err == nil {
		t.Fatal("expected min/max error")
	}

	bad = Default()
	bad.Level = 9
	if err := bad.Validate(); err == nil {
		t.Fatal("expected level error")
	}

	bad = Default()
	bad.Perm = 0
	if err := bad.Validate(); err == nil {
		t.Fatal("expected perm error")
	}

	bad = Default()
	bad.ADPolicy = true
	bad.ADMinLen = 64
	bad.MaxLen = 32
	if err := bad.Validate(); err == nil {
		t.Fatal("expected AD min/max error")
	}

	c = Default()
	c.MaxLen = 0
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.MaxLen != 128 {
		t.Fatalf("MaxLen = %d, want 128", c.MaxLen)
	}
}
