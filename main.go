package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"wordlistkingdom/pkg/bloom"
	"wordlistkingdom/pkg/filter"
	"wordlistkingdom/pkg/mutations"
	"wordlistkingdom/pkg/reader"
	"wordlistkingdom/pkg/writer"
)

const banner = `
 ██╗    ██╗ ██╗  ██╗
 ██║    ██║ ██║ ██╔╝
 ██║ █╗ ██║ █████╔╝
 ██║███╗██║ ██╔═██╗
 ╚███╔███╔╝ ██║  ██╗
  ╚══╝╚══╝  ╚═╝  ╚═╝   WordlistKingdom v1.1
  ─────────────────────────────────────────────────────────────────
  Target-Specific Custom Wordlist Generator
  For Authorised Penetration Testing & Red Team Operations Only
  ─────────────────────────────────────────────────────────────────
`

func main() {
	inputFile  := flag.String("input",      "keywords.txt",       "Input keywords file path")
	outputFile := flag.String("output",     "custom_wordlist.txt","Output file path (use - for stdout)")
	userlist   := flag.String("userlist",   "",                   "Username list for pivot mutations (SAMAccountName format)")
	yearsFlag  := flag.String("years",      "",                   "Extra years to include, comma-separated (e.g. 1938,2010)")
	workers    := flag.Int("workers",       runtime.NumCPU(),     "Parallel worker count (default: NumCPU)")
	minLen     := flag.Int("min-len",       8,                    "Minimum password length for AD policy filter")
	noFilter   := flag.Bool("no-filter",    false,                "Disable AD policy filter (output all mutations)")
	bloomSize  := flag.Uint("bloom-size",   5_000_000,            "Expected unique candidates (tunes bloom filter accuracy)")
	verbose    := flag.Bool("v",            false,                "Print live progress counter")
	flag.Parse()

	stdout := *outputFile == "-"

	var status io.Writer = os.Stdout
	if stdout {
		status = os.Stderr
	}

	fmt.Fprint(status, banner)

	keywords, err := reader.ReadKeywords(*inputFile)
	if err != nil {
		log.Fatalf("[-] %v", err)
	}

	var usernames []string
	if *userlist != "" {
		usernames, err = reader.ReadKeywords(*userlist)
		if err != nil {
			log.Fatalf("[-] userlist: %v", err)
		}
	}

	if *yearsFlag != "" {
		var custom []int
		for _, s := range strings.Split(*yearsFlag, ",") {
			if y, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				custom = append(custom, y)
			}
		}
		mutations.SetExtraYears(custom)
	}

	fmt.Fprintf(status, "[*] Keywords loaded : %d  (from %s)\n", len(keywords), *inputFile)
	if len(usernames) > 0 {
		fmt.Fprintf(status, "[*] Usernames loaded: %d  (from %s)\n", len(usernames), *userlist)
	}
	fmt.Fprintf(status, "[*] Workers         : %d\n", *workers)
	fmt.Fprintf(status, "[*] AD policy filter: %v  (min-len=%d)\n", !*noFilter, *minLen)
	fmt.Fprintf(status, "[*] Bloom filter    : ~%d expected unique entries\n\n", *bloomSize)

	bf := bloom.New(*bloomSize, 0.01)

	w, err := writer.New(*outputFile)
	if err != nil {
		log.Fatalf("[-] %v", err)
	}

	pf     := filter.NewPolicyFilter(*minLen, !*noFilter)
	engine := mutations.NewEngine(*workers, bf, pf, w)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if *verbose {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			start := time.Now()
			for {
				select {
				case <-ticker.C:
					fmt.Fprintf(status, "\r[~] Written: %8d | Elapsed: %s",
						w.Count(), time.Since(start).Round(time.Millisecond))
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	start := time.Now()
	count, err := engine.Run(keywords, usernames)
	if err != nil {
		log.Fatalf("[-] Engine error: %v", err)
	}
	cancel()

	if err := w.Close(); err != nil {
		log.Fatalf("[-] Failed to flush output: %v", err)
	}

	elapsed := time.Since(start)

	if *verbose {
		fmt.Fprintln(status)
	}
	fmt.Fprintf(status, "[+] Done!  %d unique passwords written to '%s'\n", count, *outputFile)
	fmt.Fprintf(status, "[+] Time   : %s\n", elapsed.Round(time.Millisecond))
	fmt.Fprintf(status, "[+] Speed  : %.0f entries/sec\n", float64(count)/elapsed.Seconds())

	if !stdout {
		if info, err := os.Stat(*outputFile); err == nil {
			fmt.Fprintf(status, "[+] Size   : %s\n", fmtBytes(info.Size()))
		}
	}

	fp := bf.EstimatedFP()
	if fp > 0.05 {
		fmt.Fprintf(status, "[!] Warning: bloom FP ~%.1f%% — increase -bloom-size to reduce duplicates\n", fp*100)
	} else {
		fmt.Fprintf(status, "[+] Bloom FP: ~%.5f%%\n", fp*100)
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
