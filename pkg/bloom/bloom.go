package bloom

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sync"
)

type Filter struct {
	bits  []uint64
	k     uint
	m     uint
	mu    sync.Mutex
	count uint64
}

func New(n uint, fp float64) *Filter {
	m := calcM(n, fp)
	k := calcK(m, n)
	return &Filter{
		bits: make([]uint64, (m+63)/64),
		k:    k,
		m:    m,
	}
}

func (f *Filter) TestAndAdd(item string) bool {
	h1, h2 := hashPair(item)

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := uint(0); i < f.k; i++ {
		pos := (h1 + uint64(i)*h2) % uint64(f.m)
		if f.bits[pos>>6]&(1<<(pos&63)) == 0 {
			for j := uint(0); j < f.k; j++ {
				p := (h1 + uint64(j)*h2) % uint64(f.m)
				f.bits[p>>6] |= 1 << (p & 63)
			}
			f.count++
			return false
		}
	}
	return true
}

func (f *Filter) Count() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count
}

func (f *Filter) EstimatedFP() float64 {
	f.mu.Lock()
	n := float64(f.count)
	m := float64(f.m)
	k := float64(f.k)
	f.mu.Unlock()
	return math.Pow(1-math.Exp(-k*n/m), k)
}

func calcM(n uint, fp float64) uint {
	m := uint(math.Ceil(-float64(n) * math.Log(fp) / (math.Ln2 * math.Ln2)))
	if m < 64 {
		return 64
	}
	return m
}

func calcK(m, n uint) uint {
	k := uint(math.Round(float64(m) / float64(n) * math.Ln2))
	if k < 1 {
		return 1
	}
	if k > 20 {
		return 20
	}
	return k
}

func hashPair(s string) (uint64, uint64) {
	b := []byte(s)

	h1 := fnv.New64()
	h1.Write(b)
	v1 := h1.Sum64()

	h2 := fnv.New64a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(len(s)))
	h2.Write(buf[:])
	h2.Write(b)
	v2 := h2.Sum64()

	return v1, v2
}
