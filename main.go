package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
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
  ╚══╝╚══╝  ╚═╝  ╚═╝   WordlistKingdom v1.0
  ─────────────────────────────────────────────────────────────────
  Target-Specific Custom Wordlist Generator
  For Authorised Penetration Testing & Red Team Operations Only
  ─────────────────────────────────────────────────────────────────
`

func main() {
	inputFile  := flag.String("input",      "keywords.txt",       "Input keywords file path")
	outputFile := flag.String("output",     "custom_wordlist.txt","Output wordlist file path")
	workers    := flag.Int("workers",       runtime.NumCPU(),     "Parallel worker count (default: NumCPU)")
	minLen     := flag.Int("min-len",       8,                    "Minimum password length for AD policy filter")
	noFilter   := flag.Bool("no-filter",    false,                "Disable AD policy filter (output all mutations)")
	bloomSize  := flag.Uint("bloom-size",   5_000_000,            "Expected unique candidates (tunes bloom filter accuracy)")
	verbose    := flag.Bool("v",            false,                "Print live progress counter")
	flag.Parse()

	fmt.Print(banner)

	keywords, err := reader.ReadKeywords(*inputFile)
	if err != nil {
		log.Fatalf("[-] %v", err)
	}
	fmt.Printf("[*] Keywords loaded : %d  (from %s)\n", len(keywords), *inputFile)
	fmt.Printf("[*] Workers         : %d\n", *workers)
	fmt.Printf("[*] AD policy filter: %v  (min-len=%d)\n", !*noFilter, *minLen)
	fmt.Printf("[*] Bloom filter    : ~%d expected unique entries\n\n", *bloomSize)

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
					fmt.Printf("\r[~] Written: %8d | Elapsed: %s",
						w.Count(), time.Since(start).Round(time.Millisecond))
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	start := time.Now()
	count, err := engine.Run(keywords)
	if err != nil {
		log.Fatalf("[-] Engine error: %v", err)
	}
	cancel()

	if err := w.Close(); err != nil {
		log.Fatalf("[-] Failed to flush output: %v", err)
	}

	elapsed := time.Since(start)

	if *verbose {
		fmt.Println()
	}
	fmt.Printf("[+] Done!  %d unique passwords written to '%s'\n", count, *outputFile)
	fmt.Printf("[+] Time   : %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("[+] Speed  : %.0f entries/sec\n", float64(count)/elapsed.Seconds())

	if info, err := os.Stat(*outputFile); err == nil {
		fmt.Printf("[+] Size   : %s\n", fmtBytes(info.Size()))
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
