// Package app wires the reader, generator, policy, deduplication and writer
// into one deterministic pipeline. Run is deliberately side-effect free apart
// from the files it is told to touch and the writers it is given, which keeps
// the end-to-end behaviour testable.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"wordlistkingdom/pkg/config"
	"wordlistkingdom/pkg/dedup"
	"wordlistkingdom/pkg/generate"
	"wordlistkingdom/pkg/output"
	"wordlistkingdom/pkg/policy"
	"wordlistkingdom/pkg/source"
)

// ErrInterrupted is returned when generation was stopped by the caller's
// context (typically Ctrl+C) after the partial wordlist was flushed.
var ErrInterrupted = errors.New("generation interrupted")

// ErrNoCandidates is returned when every generated candidate was filtered out.
var ErrNoCandidates = errors.New("no candidates survived filtering")

// Stats is the report for one run.
type Stats struct {
	Keywords       int
	UniqueKeywords int
	Usernames      int
	ExcludeWords   int
	ExcludeLines   int

	Attempts   int64
	Written    int64
	Duplicates int64
	Excluded   int64
	Filtered   int64
	Rejected   [policy.Count]int64
	Phases     [generate.PhaseCount]int64
	Truncated  bool

	Elapsed time.Duration
	Bytes   int64
	Output  string
}

// Run executes one generator run. stdout is used for -v echo; stderr receives
// the banner, progress and summary.
func Run(ctx context.Context, cfg config.Config, version string, stdout, stderr io.Writer) (Stats, error) {
	var st Stats
	if err := cfg.Validate(); err != nil {
		return st, err
	}
	stdinInputs := 0
	for _, path := range []string{cfg.Input, cfg.Userlist, cfg.Userdata} {
		if path == "-" {
			stdinInputs++
		}
	}
	if stdinInputs > 1 {
		return st, fmt.Errorf("stdin can only be used for one of -input, -userlist or -userdata")
	}
	if err := checkPaths(cfg); err != nil {
		return st, err
	}

	if !cfg.Quiet {
		printBanner(stderr, version)
	}

	keywords, kst, err := source.ReadFile(cfg.Input, source.Options{})
	if err != nil {
		return st, err
	}
	if len(keywords) == 0 {
		return st, fmt.Errorf("no usable keywords in %s", cfg.Input)
	}
	st.Keywords = kst.Lines
	st.UniqueKeywords = len(keywords)

	var users []generate.User
	if cfg.Userdata != "" {
		records, _, err := source.ReadUserdataFile(cfg.Userdata)
		if err != nil {
			return st, fmt.Errorf("userdata: %w", err)
		}
		for _, record := range records {
			users = append(users, generate.User{
				Username:  record.Username,
				First:     record.First,
				Last:      record.Last,
				BirthYear: record.BirthYear,
			})
		}
	}
	if cfg.Userlist != "" {
		names, _, err := source.ReadFile(cfg.Userlist, source.Options{Usernames: true})
		if err != nil {
			return st, fmt.Errorf("userlist: %w", err)
		}
		users = mergeUsers(users, names)
	}
	st.Usernames = len(users)

	if !cfg.Quiet {
		fmt.Fprintf(stderr, "[*] Keywords : %d unique (%d lines, %d skipped, %d duplicates)\n",
			len(keywords), kst.Lines, kst.Skipped, kst.Duplicates)
		if len(users) > 0 {
			fmt.Fprintf(stderr, "[*] Usernames: %d unique\n", len(users))
		}
		fmt.Fprintf(stderr, "[*] Level    : %d (%s), max terms: %d\n",
			cfg.Level, levelName(cfg.Level), cfg.Perm)
		fmt.Fprintf(stderr, "[*] Pattern  : %s\n", cfg.Pattern.String())
		fmt.Fprintf(stderr, "[*] Policy   : min=%d, max=%d, ascii=%v, AD=%v\n",
			cfg.MinLen, cfg.MaxLen, cfg.ASCIIOnly, cfg.ADPolicy)
	}

	var excludeSet *dedup.Sorted
	if len(cfg.Exclude) > 0 {
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "[*] Loading %d exclusion list(s)...\n", len(cfg.Exclude))
		}
		var unique, lines int
		excludeSet, unique, lines, err = dedup.LoadSorted(cfg.Exclude, excludeEstimate(cfg.Exclude))
		if err != nil {
			return st, err
		}
		st.ExcludeWords = unique
		st.ExcludeLines = lines
		if !cfg.Quiet {
			fmt.Fprintf(stderr, "[*] Exclude  : %d unique words (%d lines)\n", unique, lines)
		}
	}

	gen := generate.NewWithUsers(cfg, keywords, users)
	if gen.Terms() == 0 {
		return st, fmt.Errorf("no keywords survived normalisation")
	}

	pol := policy.New(cfg)
	seen := dedup.New(1 << 16)
	var echo io.Writer
	if cfg.Verbose && cfg.Output != "-" {
		echo = stdout
	}
	w, err := output.New(cfg.Output, stdout, echo)
	if err != nil {
		return st, err
	}

	var (
		written  int64
		writeErr error
	)

	emit := func(candidate string) bool {
		if cfg.MaxCandidates > 0 && written >= cfg.MaxCandidates {
			st.Truncated = true
			return false
		}
		reason := pol.Check(candidate)
		if reason != policy.OK {
			st.Rejected[reason]++
			st.Filtered++
			return true
		}
		h := dedup.Hash(candidate)
		if excludeSet != nil && excludeSet.Len() > 0 && excludeSet.HasHash(h) {
			st.Excluded++
			return true
		}
		if !seen.AddHash(h) {
			st.Duplicates++
			return true
		}
		if err := w.Write(candidate); err != nil {
			writeErr = err
			return false
		}
		written++
		return true
	}

	start := time.Now()
	stopProgress := func() {}
	if cfg.Progress && !cfg.Quiet {
		done := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					fmt.Fprintf(stderr, "\r[~] %d written | %s elapsed     ",
						w.Count(), time.Since(start).Round(time.Millisecond))
				case <-done:
					fmt.Fprint(stderr, "\r"+strings.Repeat(" ", 64)+"\r")
					return
				}
			}
		}()
		stopProgress = func() {
			close(done)
			wg.Wait()
		}
	}

	gstats := gen.Generate(ctx, emit)
	stopProgress()

	st.Attempts = gstats.Attempts
	st.Phases = gstats.Phases
	st.Elapsed = time.Since(start)
	st.Output = cfg.Output

	closeErr := w.Close()
	st.Written = w.Count()
	if cfg.Output != "-" {
		if info, err := os.Stat(cfg.Output); err == nil {
			st.Bytes = info.Size()
		}
	}

	if !cfg.Quiet {
		printSummary(st, cfg, stderr)
	}

	if writeErr != nil {
		return st, writeErr
	}
	if closeErr != nil {
		return st, closeErr
	}
	if ctx.Err() != nil {
		return st, ErrInterrupted
	}
	if st.Written == 0 {
		return st, ErrNoCandidates
	}
	return st, nil
}

func printBanner(w io.Writer, version string) {
	fmt.Fprintf(w, `
 ██╗    ██╗ ██╗  ██╗
 ██║    ██║ ██║ ██╔╝
 ██║ █╗ ██║ █████╔╝
 ██║███╗██║ ██╔═██╗
 ╚███╔███╔╝ ██║  ██╗
  ╚══╝╚══╝  ╚═╝  ╚═╝   WordlistKingdom %s
  ─────────────────────────────────────────────────────
  target-specific password candidate generator
  ─────────────────────────────────────────────────────
`, version)
}

func printSummary(st Stats, cfg config.Config, w io.Writer) {
	fmt.Fprintf(w, "\n[+] Total created words: %d\n", st.Written)
	if st.Output != "-" {
		fmt.Fprintf(w, "[+] Total size of wordlist: %s (%.6f MB)\n", fmtBytes(st.Bytes), float64(st.Bytes)/(1<<20))
	}
	fmt.Fprintf(w, "[+] Generated: %d candidates | duplicates: %d | policy-filtered: %d | excluded: %d\n",
		st.Attempts, st.Duplicates, st.Filtered, st.Excluded)
	if st.Output != "-" {
		fmt.Fprintf(w, "[+] Output: %s\n", st.Output)
	} else {
		fmt.Fprintln(w, "[+] Output: stdout")
	}
	secs := st.Elapsed.Seconds()
	rate := 0.0
	if secs > 0 {
		rate = float64(st.Written) / secs
	}
	fmt.Fprintf(w, "[+] Time: %s | %.0f candidates/sec\n", st.Elapsed.Round(time.Millisecond), rate)

	if st.Truncated {
		fmt.Fprintf(w, "[!] Candidate cap (-max-candidates %d) reached; run was stopped early.\n", cfg.MaxCandidates)
	}

	if cfg.Verbose {
		fmt.Fprintln(w, "\n[+] Phase breakdown (attempts before filtering):")
		for p := generate.Phase(0); p < generate.PhaseCount; p++ {
			if st.Phases[p] == 0 {
				continue
			}
			fmt.Fprintf(w, "    %-18s %8d\n", p.String(), st.Phases[p])
		}
		if st.Filtered > 0 {
			fmt.Fprintln(w, "\n[+] Filter reasons:")
			for r := policy.Reason(0); r < policy.Reason(policy.Count); r++ {
				if st.Rejected[r] == 0 {
					continue
				}
				fmt.Fprintf(w, "    %-18s %8d\n", r.String(), st.Rejected[r])
			}
		}
	}
}

func levelName(level int) string {
	switch level {
	case 0:
		return "focused"
	case 1:
		return "standard"
	case 2:
		return "extended"
	case 3:
		return "complete"
	default:
		return "unknown"
	}
}

func fmtBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// mergeUsers keeps rich userdata records and appends bare usernames that are
// not already present. Looking at only the username is intentional: a plain
// -userlist entry should enrich, not replace, a CSV record.
func mergeUsers(rich []generate.User, bare []string) []generate.User {
	seen := make(map[string]bool, len(rich)+len(bare))
	out := make([]generate.User, 0, len(rich)+len(bare))
	for _, u := range rich {
		key := strings.ToLower(strings.TrimSpace(u.Username))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, u)
	}
	for _, name := range bare {
		name = strings.TrimSpace(name)
		key := strings.ToLower(name)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, generate.User{Username: name})
	}
	return out
}

// excludeEstimate guesses an exclusion-set size from file sizes so a large
// rockyou load does not rehash repeatedly. It is only a hint.
func excludeEstimate(paths []string) int {
	total := 0
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		// Average wordlist line is roughly 9-14 bytes; divide small files
		// by 8 and cap the preallocation to keep memory predictable.
		n := int(info.Size() / 10)
		if n < 1024 {
			n = 1024
		}
		if n > 40_000_000 {
			n = 40_000_000
		}
		total += n
	}
	return total
}

func checkPaths(cfg config.Config) error {
	if cfg.Output == "-" {
		return nil
	}
	out, err := filepath.Abs(cfg.Output)
	if err != nil {
		return err
	}
	outInfo, outStatErr := os.Stat(cfg.Output)

	same := func(path string) bool {
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		if abs == out {
			return true
		}
		if outStatErr != nil {
			return false
		}
		inInfo, err := os.Stat(path)
		return err == nil && os.SameFile(outInfo, inInfo)
	}

	for _, in := range []string{cfg.Input, cfg.Userlist, cfg.Userdata} {
		if in == "" || in == "-" {
			continue
		}
		if same(in) {
			return fmt.Errorf("output path %q is the same as an input file", cfg.Output)
		}
	}
	for _, ex := range cfg.Exclude {
		if same(ex) {
			return fmt.Errorf("output path %q is also used as an exclusion list", cfg.Output)
		}
	}
	return nil
}
