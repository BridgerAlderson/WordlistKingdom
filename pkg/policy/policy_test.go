package policy

import (
	"testing"

	"wordlistkingdom/pkg/config"
)

func testCfg() config.Config {
	c := config.Default()
	c.MinLen = 1
	c.MaxLen = 32
	c.ADPolicy = false
	return c
}

func TestBasicChecks(t *testing.T) {
	p := New(testCfg())
	cases := []struct {
		in   string
		want Reason
	}{
		{"", Empty},
		{"corp2026", OK},
		{"corp..local", Separator},
		{".corp", Separator},
		{"corp local", Whitespace},
		{"corp\x00local", Control},
		{"şifre", OK},
	}
	for _, tc := range cases {
		if got := p.Check(tc.in); got != tc.want {
			t.Errorf("Check(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestLengthBounds(t *testing.T) {
	cfg := testCfg()
	cfg.MinLen = 8
	cfg.MaxLen = 12
	p := New(cfg)
	if got := p.Check("corp"); got != TooShort {
		t.Fatalf("short = %v", got)
	}
	if got := p.Check("corp2026"); got != OK {
		t.Fatalf("min = %v", got)
	}
	if got := p.Check("corp2026202620"); got != TooLong {
		t.Fatalf("long = %v", got)
	}
}

func TestSpaceAndASCII(t *testing.T) {
	cfg := testCfg()
	cfg.AllowSpace = true
	p := New(cfg)
	if got := p.Check("corp local"); got != OK {
		t.Fatalf("space allowed = %v", got)
	}
	cfg.ASCIIOnly = true
	p = New(cfg)
	if got := p.Check("şifre"); got != NonASCII {
		t.Fatalf("ascii = %v", got)
	}
}

func TestADComplexity(t *testing.T) {
	cfg := testCfg()
	cfg.ADPolicy = true
	cfg.ADMinLen = 8
	p := New(cfg)

	cases := []struct {
		in   string
		want Reason
	}{
		{"Corp2026", OK},            // upper + lower + digit
		{"corp2026", ADComplexity},  // lower + digit only
		{"CORP2026", ADComplexity},  // upper + digit only
		{"Corp2026!", OK},           // all four
		{"Corp!", ADComplexity},     // too short for AD
		{"CorpLocal", ADComplexity}, // upper + lower only
	}
	for _, tc := range cases {
		if got := p.Check(tc.in); got != tc.want {
			t.Errorf("AD Check(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}

	cfg.ADMinLen = 12
	p = New(cfg)
	if got := p.Check("Corp2026!"); got != ADComplexity {
		t.Fatalf("AD min = %v", got)
	}
}

func TestReasonStrings(t *testing.T) {
	if TooShort.String() != "too short" {
		t.Fatalf("reason string = %q", TooShort.String())
	}
	if Reason(999).String() != "unknown" {
		t.Fatal("unknown reason mismatch")
	}
	if Count != 9 {
		t.Fatalf("Count = %d", Count)
	}
}
