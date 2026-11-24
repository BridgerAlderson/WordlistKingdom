// Package dedup provides an exact (up to 64-bit hash collisions) string set for
// candidate deduplication and exclusion lists. A compact open-addressed table
// is used instead of a Bloom filter: it never drops a unique candidate because
// of a false positive, and it needs only eight bytes per slot.
package dedup

import (
	"fmt"

	"wordlistkingdom/pkg/source"
)

const (
	offset64 = 14695981039346656037
	prime64  = 1099511628211

	minCapacity = 1 << 12
	loadNum     = 7
	loadDen     = 10
)

// Set is an open-addressed set of 64-bit string hashes. The zero value is not
// ready for use; call New.
type Set struct {
	keys  []uint64
	count int
	mask  uint64
}

// Hash returns the 64-bit FNV-1a hash of s. Zero is remapped to one because
// zero is the empty-slot sentinel; the collision probability penalty is
// negligible for wordlist-sized inputs.
func Hash(s string) uint64 {
	h := uint64(offset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	if h == 0 {
		return 1
	}
	return h
}

// New creates a set with room for approximately expected entries. Pass zero to
// start small and grow automatically.
func New(expected int) *Set {
	capacity := minCapacity
	if expected > 0 {
		want := expected * loadDen / loadNum
		for capacity < want {
			capacity <<= 1
			if capacity <= 0 {
				break
			}
		}
	}
	return &Set{
		keys: make([]uint64, capacity),
		mask: uint64(capacity - 1),
	}
}

// AddHash inserts h and reports whether it was not already present.
func (s *Set) AddHash(h uint64) bool {
	if h == 0 {
		h = 1
	}
	if (s.count+1)*loadDen >= len(s.keys)*loadNum {
		s.grow()
	}
	idx := h & s.mask
	for {
		k := s.keys[idx]
		if k == 0 {
			s.keys[idx] = h
			s.count++
			return true
		}
		if k == h {
			return false
		}
		idx = (idx + 1) & s.mask
	}
}

// HasHash reports whether h is present.
func (s *Set) HasHash(h uint64) bool {
	if h == 0 {
		h = 1
	}
	idx := h & s.mask
	for {
		k := s.keys[idx]
		if k == 0 {
			return false
		}
		if k == h {
			return true
		}
		idx = (idx + 1) & s.mask
	}
}

// AddString inserts s and reports whether it was new.
func (s *Set) AddString(str string) bool { return s.AddHash(Hash(str)) }

// HasString reports whether s is present.
func (s *Set) HasString(str string) bool { return s.HasHash(Hash(str)) }

// Len returns the number of inserted entries.
func (s *Set) Len() int { return s.count }

func (s *Set) grow() {
	if len(s.keys) == 0 {
		s.keys = make([]uint64, minCapacity)
		s.mask = uint64(len(s.keys) - 1)
		return
	}
	old := s.keys
	capacity := len(old) << 1
	if capacity <= 0 {
		panic("dedup: set capacity overflow")
	}
	s.keys = make([]uint64, capacity)
	s.mask = uint64(capacity - 1)
	s.count = 0
	for _, k := range old {
		if k != 0 {
			s.AddHash(k)
		}
	}
}

// Load reads every wordlist in paths into set. It returns the number of unique
// entries added and the number of lines consumed.
func Load(paths []string, set *Set) (unique int, lines int, err error) {
	for _, path := range paths {
		n, err := source.ForEachLine(path, func(line string) error {
			if set.AddString(line) {
				unique++
			}
			return nil
		})
		if err != nil {
			return unique, lines, fmt.Errorf("exclude %s: %w", path, err)
		}
		lines += n
	}
	return unique, lines, nil
}
