package policy

import (
	"testing"

	"wordlistkingdom/pkg/config"
)

func FuzzCheck(f *testing.F) {
	for _, seed := range []string{
		"corp2026",
		"Corp2026!",
		"corp..local",
		"corp local",
		"şifre",
		"",
	} {
		f.Add(seed)
	}
	cfg := config.Default()
	cfg.ADPolicy = true
	cfg.AllowSpace = true
	p := New(cfg)
	f.Fuzz(func(t *testing.T, s string) {
		reason := p.Check(s)
		if reason < 0 || reason >= reasonCount {
			t.Fatalf("reason out of range: %d", reason)
		}
	})
}
