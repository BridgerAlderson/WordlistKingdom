// Package cli parses command-line flags, including the legacy flags from the
// original wordlist_generator.exe (-L, -o, -lvl, -perm, -pattern, -v).
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"wordlistkingdom/pkg/config"
)

// Options is the result of a successful parse.
type Options struct {
	Config  config.Config
	Version bool
}

// Version is injected by main; kept here so usage can print it.
var Version = "dev"

// Parse parses args. On -h/-help it returns flag.ErrHelp after printing usage
// to stderr.
func Parse(args []string, stderr io.Writer) (Options, error) {
	args = normalizeAliases(args)

	cfg := config.Default()
	fs := flag.NewFlagSet("wordlistkingdom", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(stderr, usageText()) }

	var (
		input    = fs.String("input", cfg.Input, "keyword file path, or - for stdin")
		output   = fs.String("output", cfg.Output, "output file path, or - for stdout")
		userlist = fs.String("userlist", cfg.Userlist, "username list for account-flavoured candidates")
		userdata = fs.String("userdata", cfg.Userdata, "CSV with username,first,last,birthyear for name/date pivots")
		level    = fs.Int("level", cfg.Level, "generation depth: 0 focused, 1 standard, 2 extended, 3 deep")
		perm     = fs.Int("perm", cfg.Perm, "maximum terms per candidate: 1, 2 or 3")
		pattern  = fs.String("pattern", "", "class selector: q lower, @ upper, % digits, ! specials")
		minLen   = fs.Int("min", cfg.MinLen, "minimum candidate length")
		maxLen   = fs.Int("max", cfg.MaxLen, "maximum candidate length (0 = 128)")
		adPolicy = fs.Bool("ad-policy", cfg.ADPolicy, "enforce AD complexity (3 of 4 classes)")
		adMin    = fs.Int("ad-min", cfg.ADMinLen, "minimum length enforced by -ad-policy")
		ascii    = fs.Bool("ascii", cfg.ASCIIOnly, "drop non-ASCII candidates (ASCII-folded forms remain)")
		space    = fs.Bool("allow-space", cfg.AllowSpace, "allow spaces in candidates")
		turkish  = fs.Bool("tr", cfg.Turkish, "add Turkish case variants for every keyword")
		years    = fs.String("years", "", "extra years, comma-separated (e.g. 1923,1974)")
		dates    = fs.String("dates", "", "target dates, comma-separated (e.g. 2000-01-02,1990-05-17)")
		numbers  = fs.String("numbers", "", "extra numeric suffixes, comma-separated (e.g. 1881,34)")
		specials = fs.String("specials", "", "extra special suffixes (e.g. '!@#._' or '!,!123')")
		exclude  = fs.String("exclude", "", "wordlists to exclude, comma-separated (gzip supported)")
		pair     = fs.Int("pair-limit", cfg.PairLimit, "keywords eligible for two-word combinations (0 disables)")
		triple   = fs.Int("triple-limit", cfg.TripleLimit, "keywords eligible for three-word combinations")
		userKW   = fs.Int("user-keywords", cfg.UserKeywordLimit, "keywords crossed with each username (0 disables)")
		maxCand  = fs.Int64("max-candidates", cfg.MaxCandidates, "hard output cap (0 = unlimited)")
		verbose  = fs.Bool("v", cfg.Verbose, "echo every candidate to stdout (legacy behaviour)")
		quiet    = fs.Bool("quiet", cfg.Quiet, "suppress banner and summary")
		progress = fs.Bool("progress", cfg.Progress, "show a live counter on stderr")
		version  = fs.Bool("version", false, "print version and exit")

		// Accepted for compatibility with v1.1 scripts. They have no effect:
		// deduplication is exact and the AD policy is opt-in.
		_ = fs.Int("workers", 0, "deprecated: ignored")
		_ = fs.Bool("no-filter", false, "deprecated: AD policy is opt-in")
		_ = fs.Uint("bloom-size", 0, "deprecated: exact dedup is used")
	)

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}
	if fs.NArg() > 0 {
		return Options{}, fmt.Errorf("unexpected argument %q (flags only; see -h)", fs.Arg(0))
	}

	opts := Options{Version: *version}
	cfg.Input = *input
	cfg.Output = *output
	cfg.Userlist = *userlist
	cfg.Userdata = *userdata
	cfg.Level = *level
	cfg.Perm = *perm
	cfg.MinLen = *minLen
	cfg.MaxLen = *maxLen
	cfg.ADPolicy = *adPolicy
	cfg.ADMinLen = *adMin
	cfg.ASCIIOnly = *ascii
	cfg.AllowSpace = *space
	cfg.Turkish = *turkish
	cfg.PairLimit = *pair
	cfg.TripleLimit = *triple
	cfg.UserKeywordLimit = *userKW
	cfg.MaxCandidates = *maxCand
	cfg.Verbose = *verbose
	cfg.Quiet = *quiet
	cfg.Progress = *progress

	p, err := config.ParsePattern(*pattern)
	if err != nil {
		return Options{}, err
	}
	cfg.Pattern = p

	if cfg.Years, err = config.ParseYears(*years); err != nil {
		return Options{}, err
	}
	if cfg.Dates, err = config.ParseDates(*dates); err != nil {
		return Options{}, err
	}
	if cfg.ExtraNumbers, err = config.ParseNumbers(*numbers); err != nil {
		return Options{}, err
	}
	if cfg.ExtraSpecials, err = config.ParseSpecials(*specials); err != nil {
		return Options{}, err
	}
	cfg.Exclude = config.ParsePaths(*exclude)

	// The legacy default input was inputs.txt; the project default is
	// keywords.txt. Fall back only when the operator did not choose a path.
	if !wasSet(fs, "input") && !fileExists(cfg.Input) && fileExists("inputs.txt") {
		cfg.Input = "inputs.txt"
	}

	if err := cfg.Validate(); err != nil {
		return Options{}, err
	}
	opts.Config = cfg
	return opts, nil
}

var aliases = map[string]string{
	"L":       "input",
	"o":       "output",
	"lvl":     "level",
	"min-len": "min",
}

// normalizeAliases rewrites legacy single-letter/long aliases to canonical
// names before flag parsing. Values are never inspected.
func normalizeAliases(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		var prefix string
		switch {
		case strings.HasPrefix(a, "--"):
			prefix = "--"
		case strings.HasPrefix(a, "-"):
			prefix = "-"
		default:
			out = append(out, a)
			continue
		}

		body := a[len(prefix):]
		if body == "" {
			out = append(out, a)
			continue
		}
		if eq := strings.IndexByte(body, '='); eq >= 0 {
			if canon, ok := aliases[body[:eq]]; ok {
				out = append(out, prefix+canon+"="+body[eq+1:])
				continue
			}
			out = append(out, a)
			continue
		}
		if canon, ok := aliases[body]; ok {
			out = append(out, prefix+canon)
			continue
		}
		out = append(out, a)
	}
	return out
}

func wasSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func usageText() string {
	return `WordlistKingdom ` + Version + ` - target-specific password candidate generator

Usage:
  wordlistkingdom [flags]

Input / output:
  -input, -L string     keyword file, or - for stdin (default keywords.txt,
                        falls back to inputs.txt when present)
  -output, -o string    output file, or - for stdout (default custom_wordlist.txt)
  -userlist string      username list (DOMAIN\user and user@domain are normalised)
  -userdata string      CSV: username,first,last,birthyear (header optional)
  -exclude string       comma-separated wordlists to exclude; gzip is auto-detected

Generation:
  -level, -lvl int       0 focused (default), 1 standard, 2 extended, 3 deep
  -perm int              maximum terms per candidate: 1, 2 (default) or 3
  -pattern string        q lower, @ upper, % digits, ! specials (default: all)
  -pair-limit int        keywords used for two-word combinations (default 25)
  -triple-limit int      keywords used for three-word combinations (default 8)
  -user-keywords int     keywords crossed with each username (default 3)
  -years string          extra years, comma-separated (e.g. 1923,1974)
  -dates string          target dates (YYYY-MM-DD or DD.MM.YYYY), comma-separated
  -numbers string        extra numeric suffixes, comma-separated
  -specials string       extra special suffixes (e.g. '!@#._' or '!,!123')
  -tr                    add Turkish case variants for every keyword

Output policy:
  -min int               minimum candidate length (default 1)
  -max int               maximum candidate length (default 32; 0 = 128)
  -ad-policy             require Active Directory complexity (3 of 4 classes)
  -ad-min int            minimum length for -ad-policy (default 8)
  -ascii                 drop non-ASCII candidates
  -allow-space           allow spaces in candidates
  -max-candidates int    hard output cap (default 5000000; 0 = unlimited)

Interface:
  -v                     echo every candidate to stdout (legacy behaviour)
  -progress              live counter on stderr
  -quiet                 no banner or summary
  -version               print version
  -h, -help              this help

Levels:
  0 focused    base words, years/numbers, symbols, year+symbol, word pairs.
               Best default for password spraying; a small keyword file yields
               hundreds of candidates.
  1 standard   adds role affixes, partial leetspeak, PascalCase spellings
               (ActiveDirectory) and more numeric suffixes.
  2 extended   adds seasons, more symbols, extended roles and camelCase.
  3 deep       adds prefix years, numeric prefixes, mid-word symbols, months and
               three-word combinations (also enabled by -perm 3).
  All levels are supersets of the previous one.

Legacy flags:
  -L, -o, -lvl, -min-len, -perm, -pattern, -v are supported. -workers,
  -no-filter and -bloom-size are accepted but ignored (v1.1 compatibility).

Examples:
  wordlistkingdom -L inputs.txt -o wordlist.txt -v
  wordlistkingdom -input corp.txt -o spray.txt -min 8 -ad-policy -level 1
  wordlistkingdom -L corp.txt -exclude rockyou.txt -o new.txt
  wordlistkingdom -L corp.txt -userlist users.txt -o users.txt
  wordlistkingdom -L corp.txt -userdata people.csv -o people.txt
  wordlistkingdom -L corp.txt -pattern 'q%!' -level 2 -o strict.txt
  wordlistkingdom -L corp.txt -years 1923 -tr -o turkish.txt
  wordlistkingdom -L corp.txt -dates 1990-05-17,2000-01-02 -o dates.txt
`
}

// ErrHelp is re-exported so main can distinguish a help request from a real
// parse failure without importing flag.
var ErrHelp = flag.ErrHelp

// IsHelp reports whether err is a help request.
func IsHelp(err error) bool { return errors.Is(err, flag.ErrHelp) }
