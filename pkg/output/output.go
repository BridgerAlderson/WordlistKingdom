// Package output writes candidates to a file or stdout with a large buffer and
// optional console echo. Write errors are sticky and surfaced by Close so a
// full disk cannot silently produce a truncated wordlist report.
package output

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
)

const (
	fileBuffer   = 1 << 20
	stdoutBuffer = 1 << 18
)

// Writer serialises one candidate per line.
type Writer struct {
	file   *os.File
	bw     *bufio.Writer
	echo   io.Writer
	count  atomic.Int64
	err    error
	closed bool
	path   string
}

// New opens path ("-" means stdout). When echo is non-nil every line is also
// written there; the CLI uses this for the -v echo behaviour.
func New(path string, stdout io.Writer, echo io.Writer) (*Writer, error) {
	if path == "-" {
		return &Writer{
			bw:   bufio.NewWriterSize(stdout, stdoutBuffer),
			echo: echo,
			path: "-",
		}, nil
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	return &Writer{
		file: f,
		bw:   bufio.NewWriterSize(f, fileBuffer),
		echo: echo,
		path: path,
	}, nil
}

// Write emits one candidate.
func (w *Writer) Write(s string) error {
	if w.closed {
		return errors.New("output: write after close")
	}
	if w.err != nil {
		return w.err
	}
	if _, err := w.bw.WriteString(s); err != nil {
		return w.setErr(fmt.Errorf("write %s: %w", w.path, err))
	}
	if err := w.bw.WriteByte('\n'); err != nil {
		return w.setErr(fmt.Errorf("write %s: %w", w.path, err))
	}
	if w.echo != nil {
		if _, err := io.WriteString(w.echo, s+"\n"); err != nil {
			return w.setErr(fmt.Errorf("echo: %w", err))
		}
	}
	w.count.Add(1)
	return nil
}

// Count returns the number of successfully written candidates.
func (w *Writer) Count() int64 { return w.count.Load() }

// Path returns the output path ("-" for stdout).
func (w *Writer) Path() string { return w.path }

// Close flushes and closes the underlying file. It is safe to call once.
func (w *Writer) Close() error {
	if w.closed {
		return w.err
	}
	w.closed = true
	if err := w.bw.Flush(); err != nil && w.err == nil {
		w.err = fmt.Errorf("flush %s: %w", w.path, err)
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil && w.err == nil {
			w.err = fmt.Errorf("close %s: %w", w.path, err)
		}
	}
	return w.err
}

func (w *Writer) setErr(err error) error {
	if w.err == nil {
		w.err = err
	}
	return w.err
}
