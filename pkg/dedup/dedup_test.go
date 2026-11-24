package dedup

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSetAddHas(t *testing.T) {
	s := New(0)
	if !s.AddString("corp") {
		t.Fatal("first add was not new")
	}
	if s.AddString("corp") {
		t.Fatal("duplicate add was reported new")
	}
	if !s.HasString("corp") {
		t.Fatal("HasString missed value")
	}
	if s.HasString("Corp") {
		t.Fatal("HasString is expected to be case-sensitive")
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d", s.Len())
	}
}

func TestSetGrows(t *testing.T) {
	s := New(0)
	const n = 100_000
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("word-%06d", i)
		if !s.AddString(key) {
			t.Fatalf("unexpected duplicate at %d", i)
		}
	}
	if s.Len() != n {
		t.Fatalf("Len = %d after %d adds", s.Len(), n)
	}
	for i := 0; i < n; i += 997 {
		if !s.HasString(fmt.Sprintf("word-%06d", i)) {
			t.Fatalf("value %d missing after growth", i)
		}
	}
}

func TestHashNonZero(t *testing.T) {
	for _, s := range []string{"", "a", "corp", "corp.local", "İstanbul"} {
		if Hash(s) == 0 {
			t.Fatalf("Hash(%q) returned the zero sentinel", s)
		}
	}
}

func TestLoadPlainAndGzip(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "exclude.txt")
	if err := os.WriteFile(plain, []byte("corp2026\nCorp!\ncorp2026\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gzPath := filepath.Join(dir, "exclude.txt.gz")
	f, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write([]byte("corp.local\n")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	s := New(64)
	unique, lines, err := Load([]string{plain, gzPath}, s)
	if err != nil {
		t.Fatal(err)
	}
	if unique != 3 {
		t.Fatalf("unique = %d, want 3", unique)
	}
	if lines != 4 {
		t.Fatalf("lines = %d, want 4", lines)
	}
	if !s.HasString("corp2026") || !s.HasString("corp.local") {
		t.Fatal("loaded entries missing")
	}
}

func TestSortedSet(t *testing.T) {
	s := NewSorted(8)
	values := []string{"corp", "corp2026", "Corp!", "corp", "corp.local"}
	for _, v := range values {
		s.AddString(v)
	}
	s.Build()
	if s.Len() != 4 {
		t.Fatalf("Len = %d, want 4", s.Len())
	}
	for _, v := range values {
		if !s.HasString(v) {
			t.Fatalf("missing %q", v)
		}
	}
	if s.HasString("missing") {
		t.Fatal("unexpected membership")
	}
}

func TestSortedSetMany(t *testing.T) {
	s := NewSorted(0)
	const n = 100_000
	for i := 0; i < n; i++ {
		s.AddString(fmt.Sprintf("word-%06d", i))
	}
	s.Build()
	if s.Len() != n {
		t.Fatalf("Len = %d, want %d", s.Len(), n)
	}
	for i := 0; i < n; i += 997 {
		if !s.HasString(fmt.Sprintf("word-%06d", i)) {
			t.Fatalf("missing %d", i)
		}
	}
}

func TestLoadSortedPlainAndGzip(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "exclude.txt")
	if err := os.WriteFile(plain, []byte("corp2026\nCorp!\ncorp2026\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gzPath := filepath.Join(dir, "exclude.txt.gz")
	f, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write([]byte("corp.local\n")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	set, unique, lines, err := LoadSorted([]string{plain, gzPath}, 64)
	if err != nil {
		t.Fatal(err)
	}
	if unique != 3 || lines != 4 {
		t.Fatalf("unique=%d lines=%d", unique, lines)
	}
	if !set.HasString("corp2026") || !set.HasString("corp.local") {
		t.Fatal("loaded entries missing")
	}
}
