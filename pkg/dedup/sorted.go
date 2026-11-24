package dedup

import (
	"fmt"
	"sort"

	"wordlistkingdom/pkg/source"
)

// Sorted is an exact exclusion set backed by a sorted []uint64. It uses eight
// bytes per hash instead of the open-addressed Set's slot overhead, which
// matters for multi-million-line lists such as rockyou. Lookups are binary
// searches; for wordlist-sized candidate streams that is still cheap, and the
// memory saving is substantial.
type Sorted struct {
	keys []uint64
}

// NewSorted creates an empty sorted set with room for expected hashes.
func NewSorted(expected int) *Sorted {
	if expected < 0 {
		expected = 0
	}
	return &Sorted{keys: make([]uint64, 0, expected)}
}

// AddString appends s to the set. Call Build after the last add.
func (s *Sorted) AddString(str string) {
	s.keys = append(s.keys, Hash(str))
}

// Build sorts and deduplicates the hashes. It is idempotent.
func (s *Sorted) Build() {
	if len(s.keys) < 2 {
		return
	}
	sort.Slice(s.keys, func(i, j int) bool { return s.keys[i] < s.keys[j] })

	n := 1
	for i := 1; i < len(s.keys); i++ {
		if s.keys[i] != s.keys[n-1] {
			s.keys[n] = s.keys[i]
			n++
		}
	}
	s.keys = s.keys[:n]
}

// HasHash reports whether h is present. Build must have been called.
func (s *Sorted) HasHash(h uint64) bool {
	if h == 0 {
		h = 1
	}
	i := sort.Search(len(s.keys), func(i int) bool { return s.keys[i] >= h })
	return i < len(s.keys) && s.keys[i] == h
}

// HasString reports whether str is present.
func (s *Sorted) HasString(str string) bool { return s.HasHash(Hash(str)) }

// Len returns the number of unique hashes.
func (s *Sorted) Len() int { return len(s.keys) }

// LoadSorted streams all paths into a sorted exclusion set. The expected hint
// only controls the initial allocation; it is not a correctness limit.
func LoadSorted(paths []string, expected int) (*Sorted, int, int, error) {
	set := NewSorted(expected)
	lines := 0
	for _, path := range paths {
		n, err := source.ForEachLine(path, func(line string) error {
			set.AddString(line)
			return nil
		})
		if err != nil {
			return nil, 0, lines, fmt.Errorf("exclude %s: %w", path, err)
		}
		lines += n
	}
	set.Build()
	return set, set.Len(), lines, nil
}
