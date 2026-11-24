// Package source reads keyword, username and exclusion wordlists. It handles
// BOMs, CRLF, gzip-compressed files, stdin, comments, quoting and duplicates,
// and it keeps the original spelling while deduplicating case-insensitively.
package source

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"wordlistkingdom/pkg/textutil"
)

const (
	maxKeywordRunes  = 64
	maxUsernameRunes = 128
	scannerBuffer    = 1 << 20
)

// Stats describes what happened while reading a list.
type Stats struct {
	Lines      int
	Kept       int
	Skipped    int
	Duplicates int
	Comments   int
}

// Options changes list-specific reading rules.
type Options struct {
	// Usernames enables SAMAccountName normalisation and the larger length
	// limit.
	Usernames bool
}

// ReadFile opens a path (or stdin for "-") and reads it.
func ReadFile(path string, opts Options) ([]string, Stats, error) {
	rc, err := Open(path)
	if err != nil {
		return nil, Stats{}, err
	}
	defer rc.Close()
	list, st, err := Read(rc, opts)
	if err != nil {
		return nil, st, fmt.Errorf("%s: %w", path, err)
	}
	return list, st, nil
}

// Read parses a list from r.
func Read(r io.Reader, opts Options) ([]string, Stats, error) {
	var st Stats
	seen := make(map[string]bool)
	out := make([]string, 0, 64)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), scannerBuffer)

	first := true
	for scanner.Scan() {
		st.Lines++
		line := scanner.Text()
		if first {
			line = strings.TrimPrefix(line, "\ufeff")
			first = false
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			st.Comments++
			continue
		}

		line = trimQuotes(line)
		if opts.Usernames {
			line = NormalizeUsername(line)
		}
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			st.Skipped++
			continue
		}
		if hasControl(line) || !utf8.ValidString(line) {
			st.Skipped++
			continue
		}

		limit := maxKeywordRunes
		if opts.Usernames {
			limit = maxUsernameRunes
		}
		if textutil.RuneLen(line) > limit {
			st.Skipped++
			continue
		}

		key := strings.ToLower(textutil.FoldASCII(line))
		if seen[key] {
			st.Duplicates++
			continue
		}
		seen[key] = true
		out = append(out, line)
		st.Kept++
	}
	if err := scanner.Err(); err != nil {
		return nil, st, fmt.Errorf("scan %s: %w", describe(r), err)
	}
	return out, st, nil
}

// ForEachLine streams a wordlist without building a slice. Lines are stripped
// of a UTF-8 BOM on the first line and of trailing CR; leading/trailing spaces
// are preserved because they can be part of a password.
func ForEachLine(path string, fn func(line string) error) (int, error) {
	rc, err := Open(path)
	if err != nil {
		return 0, err
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 64*1024), scannerBuffer)

	count := 0
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			line = strings.TrimPrefix(line, "\ufeff")
			first = false
		}
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		count++
		if err := fn(line); err != nil {
			return count, err
		}
	}
	if err := scanner.Err(); err != nil {
		return count, fmt.Errorf("scan %s: %w", path, err)
	}
	return count, nil
}

// Open opens a path, transparently decompressing gzip files by magic bytes so
// a renamed rockyou.txt.gz still works. "-" means stdin.
func Open(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	br := bufio.NewReaderSize(f, 64*1024)
	magic, _ := br.Peek(2)
	if len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("gzip %s: %w", path, err)
		}
		return &readerCloser{Reader: gz, closers: []io.Closer{gz, f}}, nil
	}
	return &readerCloser{Reader: br, closers: []io.Closer{f}}, nil
}

// NormalizeUsername reduces common login formats to a SAMAccountName:
//
//	CORP\jsmith      -> jsmith
//	jsmith@corp.local -> jsmith
//
// It does not alter a username that is already bare.
func NormalizeUsername(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexAny(s, `\/`); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.IndexByte(s, '@'); i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func trimQuotes(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func describe(r io.Reader) string {
	if f, ok := r.(*os.File); ok {
		return f.Name()
	}
	return "input"
}

type readerCloser struct {
	io.Reader
	closers []io.Closer
}

func (r *readerCloser) Close() error {
	var first error
	for _, c := range r.closers {
		if err := c.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
