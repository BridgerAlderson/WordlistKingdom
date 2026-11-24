package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"wordlistkingdom/pkg/config"
)

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func baseCfg(input, output string) config.Config {
	cfg := config.Default()
	cfg.Input = input
	cfg.Output = output
	cfg.Quiet = true
	return cfg
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSuffix(string(data), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\ncorp.local\n")
	output := filepath.Join(dir, "out.txt")

	cfg := baseCfg(input, output)
	var stdout, stderr bytes.Buffer
	st, err := Run(context.Background(), cfg, "test", &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if st.Written < 600 {
		t.Fatalf("written = %d, want at least 600", st.Written)
	}
	lines := readLines(t, output)
	if int64(len(lines)) != st.Written {
		t.Fatalf("lines = %d, stats = %d", len(lines), st.Written)
	}
	if st.Bytes == 0 {
		t.Fatalf("byte count not populated")
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected echo without -v: %q", stdout.String())
	}

	seen := make(map[string]bool, len(lines))
	hasYear := false
	yearPrefix := fmt.Sprintf("corp%d", time.Now().Year())
	for _, line := range lines {
		if seen[line] {
			t.Fatalf("duplicate line %q", line)
		}
		seen[line] = true
		if strings.HasPrefix(line, yearPrefix) {
			hasYear = true
		}
	}
	if !hasYear {
		t.Fatal("expected a dynamic-year candidate")
	}
	if seen["AdminCorp"] {
		t.Fatal("focused output contains an affix mutation")
	}
}

func TestVerboseEcho(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\n")
	output := filepath.Join(dir, "out.txt")
	cfg := baseCfg(input, output)
	cfg.Verbose = true

	var stdout, stderr bytes.Buffer
	st, err := Run(context.Background(), cfg, "test", &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	echoLines := strings.Count(stdout.String(), "\n")
	if int64(echoLines) != st.Written {
		t.Fatalf("echo lines = %d, written = %d", echoLines, st.Written)
	}
	if st.Written != int64(len(readLines(t, output))) {
		t.Fatal("echo count and file count differ")
	}
}

func TestExclude(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\ncorp.local\n")
	exclude := write(t, dir, "exclude.txt", "corp2026\nCorp!\n")
	output := filepath.Join(dir, "out.txt")

	cfg := baseCfg(input, output)
	cfg.Exclude = []string{exclude}
	_, err := Run(context.Background(), cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	lines := readLines(t, output)
	for _, line := range lines {
		if line == "corp2026" || line == "Corp!" {
			t.Fatalf("excluded value present: %q", line)
		}
	}
}

func TestNoCandidates(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\n")
	output := filepath.Join(dir, "out.txt")
	cfg := baseCfg(input, output)
	cfg.MinLen = 100
	cfg.MaxLen = 128

	st, err := Run(context.Background(), cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, ErrNoCandidates) {
		t.Fatalf("err = %v, want ErrNoCandidates", err)
	}
	if st.Written != 0 {
		t.Fatalf("written = %d", st.Written)
	}
}

func TestSamePathRejected(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\n")
	cfg := baseCfg(input, input)
	_, err := Run(context.Background(), cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "same as an input") {
		t.Fatalf("err = %v", err)
	}
}

func TestStdoutOutput(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\n")
	cfg := baseCfg(input, "-")
	cfg.Quiet = false

	var stdout, stderr bytes.Buffer
	st, err := Run(context.Background(), cfg, "test", &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "WordlistKingdom") {
		t.Fatal("banner leaked into stdout")
	}
	if int64(strings.Count(stdout.String(), "\n")) != st.Written {
		t.Fatalf("stdout lines and written count differ")
	}
	if !strings.Contains(stderr.String(), "WordlistKingdom") {
		t.Fatal("banner missing from stderr")
	}
}

func TestCandidateCap(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\ncorp.local\n")
	output := filepath.Join(dir, "out.txt")
	cfg := baseCfg(input, output)
	cfg.MaxCandidates = 7

	st, err := Run(context.Background(), cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Written != 7 || !st.Truncated {
		t.Fatalf("written=%d truncated=%v", st.Written, st.Truncated)
	}
	if len(readLines(t, output)) != 7 {
		t.Fatal("file does not contain exactly the capped number of lines")
	}
}

func TestCanceledContext(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\n")
	output := filepath.Join(dir, "out.txt")
	cfg := baseCfg(input, output)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	st, err := Run(ctx, cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, ErrInterrupted) {
		t.Fatalf("err = %v, want ErrInterrupted", err)
	}
	if st.Written != 0 {
		t.Fatalf("written = %d", st.Written)
	}
	if _, statErr := os.Stat(output); statErr != nil {
		t.Fatalf("partial output file missing: %v", statErr)
	}
}

func TestUserlistRun(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\n")
	users := write(t, dir, "users.txt", `CORP\jsmith`+"\n")
	output := filepath.Join(dir, "out.txt")
	cfg := baseCfg(input, output)
	cfg.Userlist = users

	st, err := Run(context.Background(), cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Usernames != 1 {
		t.Fatalf("usernames = %d", st.Usernames)
	}
	seen := make(map[string]bool)
	for _, line := range readLines(t, output) {
		seen[line] = true
	}
	if !seen["jsmith"] || !seen["jsmith.corp"] {
		t.Fatal("username pivot candidates missing")
	}
}

func TestUserdataRun(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\n")
	data := write(t, dir, "people.csv", "username,first,last,birthyear\njsmith,John,Smith,1990\n")
	output := filepath.Join(dir, "out.txt")
	cfg := baseCfg(input, output)
	cfg.Userdata = data

	st, err := Run(context.Background(), cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Usernames != 1 {
		t.Fatalf("usernames = %d", st.Usernames)
	}
	seen := make(map[string]bool)
	for _, line := range readLines(t, output) {
		seen[line] = true
	}
	for _, want := range []string{"jsmith1990!", "john.smith", "JohnSmith"} {
		if !seen[want] {
			t.Errorf("userdata output missing %q", want)
		}
	}
}

func TestCompleteRun(t *testing.T) {
	dir := t.TempDir()
	input := write(t, dir, "keywords.txt", "corp\ncorp.local\n")
	output := filepath.Join(dir, "complete.txt")
	cfg := baseCfg(input, output)
	cfg.Level = 3

	st, err := Run(context.Background(), cfg, "test", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Written < 1000 {
		t.Fatalf("complete output too small: %d", st.Written)
	}
	seen := make(map[string]bool)
	for _, line := range readLines(t, output) {
		seen[line] = true
	}
	for _, want := range []string{".Corp", "_Corp", "!Corp", "%01Corp", "corp.localCorp.local"} {
		if !seen[want] {
			t.Errorf("complete app output missing %q", want)
		}
	}
}
