package writer

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

type Writer struct {
	f     *os.File
	bw    *bufio.Writer
	mu    sync.Mutex
	count atomic.Int64
}

func New(path string) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	return &Writer{
		f:  f,
		bw: bufio.NewWriterSize(f, 64<<20),
	}, nil
}

func (w *Writer) Write(s string) {
	w.mu.Lock()
	w.bw.WriteString(s)
	w.bw.WriteByte('\n')
	w.mu.Unlock()
	w.count.Add(1)
}

func (w *Writer) Count() int64 { return w.count.Load() }

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.bw.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return w.f.Close()
}
