package cli

import (
	"bytes"
	"reflect"
	"testing"
)

func TestParseLegacyAliases(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := Parse([]string{
		"-L", "in.txt",
		"-o", "out.txt",
		"-lvl", "2",
		"-perm", "3",
		"-pattern", "q%",
		"-v",
	}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	cfg := opts.Config
	if cfg.Input != "in.txt" || cfg.Output != "out.txt" {
		t.Fatalf("input/output = %q/%q", cfg.Input, cfg.Output)
	}
	if cfg.Level != 2 || cfg.Perm != 3 || !cfg.Verbose {
		t.Fatalf("level/perm/verbose = %d/%d/%v", cfg.Level, cfg.Perm, cfg.Verbose)
	}
	if !cfg.Pattern.Lower || !cfg.Pattern.Digits || cfg.Pattern.Upper || cfg.Pattern.Special {
		t.Fatalf("pattern = %+v", cfg.Pattern)
	}
}

func TestMinLenAlias(t *testing.T) {
	opts, err := Parse([]string{"-input", "in.txt", "-min-len", "10"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Config.MinLen != 10 {
		t.Fatalf("MinLen = %d", opts.Config.MinLen)
	}
}

func TestDefaults(t *testing.T) {
	opts, err := Parse([]string{"-input", "in.txt"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := opts.Config
	if cfg.Output != "custom_wordlist.txt" {
		t.Fatalf("output = %q", cfg.Output)
	}
	if cfg.Level != 0 || cfg.Perm != 2 {
		t.Fatalf("level/perm = %d/%d", cfg.Level, cfg.Perm)
	}
	if cfg.PairLimit != 25 || cfg.MaxCandidates != 5_000_000 {
		t.Fatalf("limits = %d/%d", cfg.PairLimit, cfg.MaxCandidates)
	}
}

func TestUserdataFlag(t *testing.T) {
	opts, err := Parse([]string{"-input", "in.txt", "-userdata", "people.csv"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Config.Userdata != "people.csv" {
		t.Fatalf("userdata = %q", opts.Config.Userdata)
	}
}

func TestDatesFlag(t *testing.T) {
	opts, err := Parse([]string{"-input", "in.txt", "-dates", "2000-01-02,17.05.1990"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Config.Dates) != 2 {
		t.Fatalf("dates = %#v", opts.Config.Dates)
	}
	if _, err := Parse([]string{"-input", "in.txt", "-dates", "2000-02-30"}, &bytes.Buffer{}); err == nil {
		t.Fatal("invalid date was accepted")
	}
}

func TestDeprecatedFlagsAccepted(t *testing.T) {
	_, err := Parse([]string{
		"-input", "in.txt",
		"-workers", "4",
		"-no-filter",
		"-bloom-size", "1000000",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("deprecated flags rejected: %v", err)
	}
}

func TestInvalidInputs(t *testing.T) {
	cases := [][]string{
		{"-input", "in.txt", "-pattern", "x"},
		{"-input", "in.txt", "-level", "9"},
		{"-input", "in.txt", "-perm", "0"},
		{"-input", "in.txt", "-years", "23"},
		{"-input", "in.txt", "-min", "20", "-max", "10"},
		{"-input", "in.txt", "extra"},
	}
	for _, args := range cases {
		if _, err := Parse(args, &bytes.Buffer{}); err == nil {
			t.Errorf("Parse(%v) succeeded, want error", args)
		}
	}
}

func TestHelp(t *testing.T) {
	var stderr bytes.Buffer
	_, err := Parse([]string{"-h"}, &stderr)
	if !IsHelp(err) {
		t.Fatalf("err = %v, want help", err)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage:")) {
		t.Fatalf("usage missing: %q", stderr.String())
	}
}

func TestVersion(t *testing.T) {
	opts, err := Parse([]string{"-version", "-input", "in.txt"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Version {
		t.Fatal("Version flag not set")
	}
}

func TestNormalizeAliases(t *testing.T) {
	in := []string{"-L", "in.txt", "-o", "-", "-lvl=2", "--min-len", "8", "-pattern", "!"}
	want := []string{"-input", "in.txt", "-output", "-", "-level=2", "--min", "8", "-pattern", "!"}
	got := normalizeAliases(in)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAliases = %#v, want %#v", got, want)
	}
}

func TestQuietAndProgress(t *testing.T) {
	opts, err := Parse([]string{"-input", "in.txt", "-quiet", "-progress"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Config.Quiet || !opts.Config.Progress {
		t.Fatal("quiet/progress not parsed")
	}
}

func TestNoFilterDoesNotDisableExplicitAD(t *testing.T) {
	opts, err := Parse([]string{"-input", "in.txt", "-ad-policy", "-no-filter"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Config.ADPolicy {
		t.Fatal("-no-filter overrode an explicit -ad-policy")
	}
}
