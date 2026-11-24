package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	w, err := New(path, &bytes.Buffer{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"corp", "Corp2026", "corp!"} {
		if err := w.Write(s); err != nil {
			t.Fatal(err)
		}
	}
	if w.Count() != 3 {
		t.Fatalf("Count = %d", w.Count())
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "corp\nCorp2026\ncorp!\n"
	if string(data) != want {
		t.Fatalf("file = %q, want %q", data, want)
	}
}

func TestStdout(t *testing.T) {
	var buf bytes.Buffer
	w, err := New("-", &buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write("corp"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "corp\n" {
		t.Fatalf("stdout = %q", buf.String())
	}
}

func TestEcho(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	var echo bytes.Buffer
	w, err := New(path, &bytes.Buffer{}, &echo)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Write("corp"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if echo.String() != "corp\n" {
		t.Fatalf("echo = %q", echo.String())
	}
}

func TestWriteAfterClose(t *testing.T) {
	var buf bytes.Buffer
	w, err := New("-", &buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Write("corp"); err == nil {
		t.Fatal("write after close succeeded")
	}
}

func TestCreateError(t *testing.T) {
	_, err := New(filepath.Join(t.TempDir(), "missing", "out.txt"), &bytes.Buffer{}, nil)
	if err == nil {
		t.Fatal("expected create error")
	}
}
