package generate_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"wordlistkingdom/pkg/config"
	"wordlistkingdom/pkg/generate"
	"wordlistkingdom/pkg/textutil"
)

func collect(t *testing.T, cfg config.Config, keywords, users []string) ([]string, map[string]bool, generate.Stats) {
	t.Helper()
	g := generate.New(cfg, keywords, users)
	var order []string
	set := make(map[string]bool)
	st := g.Generate(context.Background(), func(s string) bool {
		order = append(order, s)
		set[s] = true
		return true
	})
	return order, set, st
}

func collectUsers(t *testing.T, cfg config.Config, keywords []string, users []generate.User) ([]string, map[string]bool, generate.Stats) {
	t.Helper()
	g := generate.NewWithUsers(cfg, keywords, users)
	var order []string
	set := make(map[string]bool)
	st := g.Generate(context.Background(), func(s string) bool {
		order = append(order, s)
		set[s] = true
		return true
	})
	return order, set, st
}

func TestFocusedQuality(t *testing.T) {
	cfg := config.Default()
	cfg.Level = 0
	cfg.Perm = 2
	order, set, st := collect(t, cfg, []string{"corp", "corp.local"}, nil)

	year := time.Now().Year()
	short := fmt.Sprintf("%02d", year%100)

	want := []string{
		"corp", "Corp", "CORP",
		"corp.local", "Corp.local", "CORP.LOCAL",
		fmt.Sprintf("corp%d", year),
		fmt.Sprintf("corp%d", year-1),
		"corp" + short,
		"corp123", "corp!", "corp@", "corp_", "corp?",
		"corp99!", "Corp99*", "CORP99?", "corp99.",
		fmt.Sprintf("Corp%d!", year),
		fmt.Sprintf("corp%d", year) + "*",
		fmt.Sprintf("corp%d", year) + "?",
		"corp_" + short,
		"corp.local." + fmt.Sprintf("%d", year),
		"corp.local_99",
		"corp.local26*",
		"Corp@99",
		"corpcorp", "Corpcorp", "corpCorp", "CORPCORP",
		"corpCorp.local",
		"corp.localCorp",
		"CORP.LOCAL" + fmt.Sprintf("%d", year),
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("missing expected candidate %q", w)
		}
	}

	if len(set) < 900 || len(set) > 1300 {
		t.Fatalf("focused unique output size = %d, want 900..1300", len(set))
	}
	bad := []string{
		"!corp", "2026corp", "corp.loca!l", "AdminCorp", "corpAdmin",
		"corpYaz", "c0rp", "corpOcak", "corp..local", ".corp",
		"AcmeCorp",
	}
	for _, w := range bad {
		if set[w] {
			t.Errorf("unexpected junk candidate %q", w)
		}
	}

	if st.Attempts < int64(len(order)) {
		t.Fatalf("attempts = %d, emitted = %d", st.Attempts, len(order))
	}
}

func TestDeterministic(t *testing.T) {
	cfg := config.Default()
	first, _, _ := collect(t, cfg, []string{"corp", "corp.local", "admin"}, []string{"jsmith"})
	second, _, _ := collect(t, cfg, []string{"corp", "corp.local", "admin"}, []string{"jsmith"})
	if len(first) != len(second) {
		t.Fatalf("run lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("candidate %d differs: %q vs %q", i, first[i], second[i])
		}
	}
}

func TestPermutations(t *testing.T) {
	kws := []string{"corp", "acme", "test"}
	cfg := config.Default()

	cfg.Perm = 1
	_, set, _ := collect(t, cfg, kws, nil)
	if set["corpacme"] || set["corpacmetest"] {
		t.Fatal("perm 1 produced a combination")
	}

	cfg.Perm = 2
	_, set, _ = collect(t, cfg, kws, nil)
	if !set["corpacme"] {
		t.Fatal("perm 2 did not produce a pair")
	}
	if set["corpacmetest"] {
		t.Fatal("perm 2 produced a triple")
	}

	cfg.Perm = 3
	_, set, _ = collect(t, cfg, kws, nil)
	if !set["corpacmetest"] {
		t.Fatal("perm 3 did not produce a triple")
	}
}

func TestPatternClasses(t *testing.T) {
	kws := []string{"corp", "acme"}
	cfg := config.Default()

	p, err := config.ParsePattern("q%")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Pattern = p
	order, _, _ := collect(t, cfg, kws, nil)
	for _, c := range order {
		if textutil.HasUpper(c) {
			t.Fatalf("q%% pattern produced uppercase: %q", c)
		}
		if strings.ContainsAny(c, "!@#*?_.") {
			t.Fatalf("q%% pattern produced a special: %q", c)
		}
	}

	p, err = config.ParsePattern("@!")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Pattern = p
	order, _, _ = collect(t, cfg, kws, nil)
	for _, c := range order {
		if strings.ContainsAny(c, "0123456789") {
			t.Fatalf("@! pattern produced a digit: %q", c)
		}
		if !textutil.HasUpper(c) {
			t.Fatalf("@! pattern produced a lowercase-only candidate: %q", c)
		}
	}
}

func TestCandidateCap(t *testing.T) {
	cfg := config.Default()
	g := generate.New(cfg, []string{"corp", "acme"}, nil)
	var count int
	st := g.Generate(context.Background(), func(string) bool {
		count++
		return count < 5
	})
	if count != 5 {
		t.Fatalf("sink saw %d candidates, want 5", count)
	}
	if st.Attempts != 5 {
		t.Fatalf("attempts = %d, want 5", st.Attempts)
	}
}

func TestUsernamePivots(t *testing.T) {
	cfg := config.Default()
	order, set, _ := collect(t, cfg, []string{"corp", "acme"}, []string{"jsmith", "john.smith"})

	year := time.Now().Year()
	want := []string{
		"jsmith", "JSMITH", "jsmith" + fmt.Sprintf("%d", year), "jsmith!", "jsmith.corp",
		"johnsmith", "smithj", "j.smith", "smith.john",
		"john.smith" + fmt.Sprintf("%d", year),
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("missing username candidate %q (emitted %d lines)", w, len(order))
		}
	}
}

func TestTurkishForms(t *testing.T) {
	cfg := config.Default()
	cfg.Turkish = true
	_, set, _ := collect(t, cfg, []string{"İstanbul"}, nil)
	for _, want := range []string{"istanbul", "İstanbul", "İSTANBUL", "Istanbul", "ISTANBUL"} {
		if !set[want] {
			t.Errorf("missing Turkish form %q", want)
		}
	}
}

func TestDomainDerivation(t *testing.T) {
	cfg := config.Default()
	_, set, _ := collect(t, cfg, []string{"acme.local"}, nil)
	year := time.Now().Year()

	for _, want := range []string{
		"acme", "Acme", "ACME",
		"acme" + fmt.Sprintf("%d", year),
		"acmelocal", "Acmelocal", "ACMELOCAL",
		"acmelocal" + fmt.Sprintf("%d", year),
		"acme.local",
	} {
		if !set[want] {
			t.Errorf("domain derivation missing %q", want)
		}
	}
	for _, unwanted := range []string{"local", "local" + fmt.Sprintf("%d", year)} {
		if set[unwanted] {
			t.Errorf("domain derivation produced the TLD component %q", unwanted)
		}
	}
}

func TestDateShapes(t *testing.T) {
	cfg := config.Default()
	cfg.Dates = []config.Date{{Year: 1990, Month: 5, Day: 17}}
	_, set, st := collect(t, cfg, []string{"corp"}, nil)

	for _, want := range []string{
		"corp17051990", "corp170590", "corp19900517", "corp900517",
		"corp17.05.1990", "corp17-05-1990", "corp17/05/1990",
		"corp1705", "corp0517",
		"Corp17051990", "CORP17051990",
		"corp17051990!", "corp!17051990",
		"corp1990", "corp90", "corp1990!", "corp90!",
	} {
		if !set[want] {
			t.Errorf("date generation missing %q", want)
		}
	}
	if st.Phases[generate.PhaseDates] == 0 {
		t.Fatal("date phase produced no attempts")
	}
}

func TestDatesRequireFlag(t *testing.T) {
	cfg := config.Default()
	_, set, _ := collect(t, cfg, []string{"corp"}, nil)
	if set["corp17051990"] {
		t.Fatal("date shape generated without -dates")
	}
}

func TestDeepLevel(t *testing.T) {
	cfg := config.Default()
	cfg.Level = 3
	cfg.Perm = 3
	_, set, _ := collect(t, cfg, []string{"corp", "Admin", "Test"}, nil)

	year := time.Now().Year()
	checks := []string{
		fmt.Sprintf("%dcorp", year),
		"Ad!min",
		"corpOcak",
		"CorpAdmin",
		"AdminCorp",
		"corpadmintest",
	}
	for _, w := range checks {
		if !set[w] {
			t.Errorf("deep level missing %q", w)
		}
	}
}

func TestLevelsAreSupersets(t *testing.T) {
	kws := []string{"corp", "acme", "test"}
	var levels []map[string]bool
	for lvl := 0; lvl <= 3; lvl++ {
		cfg := config.Default()
		cfg.Level = lvl
		_, set, _ := collect(t, cfg, kws, nil)
		levels = append(levels, set)
	}
	for lvl := 0; lvl < 3; lvl++ {
		for candidate := range levels[lvl] {
			if !levels[lvl+1][candidate] {
				t.Fatalf("level %d candidate %q is missing from level %d", lvl, candidate, lvl+1)
			}
		}
	}
}

func TestPairLimit(t *testing.T) {
	kws := []string{"corp", "acme", "test"}

	cfg := config.Default()
	cfg.PairLimit = 0
	_, set, _ := collect(t, cfg, kws, nil)
	if set["corpacme"] || set["corpcorp"] {
		t.Fatal("pair-limit 0 produced pairs")
	}

	cfg.PairLimit = 1
	_, set, _ = collect(t, cfg, kws, nil)
	if set["corpacme"] {
		t.Fatal("pair-limit 1 produced a cross-keyword pair")
	}
	if !set["corpcorp"] {
		t.Fatal("pair-limit 1 should still allow the first keyword's doubled form")
	}
}

func TestNoJunkAtFocused(t *testing.T) {
	cfg := config.Default()
	_, set, _ := collect(t, cfg, []string{"corp", "corp.local", "Admin"}, nil)

	prefixYear := regexp.MustCompile(`^(19|20)[0-9]{2}`)
	doubleSep := regexp.MustCompile(`\.\.|__|--|\._|_\.|\.-|-\.`)
	leadingSep := regexp.MustCompile(`^[._@-]`)

	for c := range set {
		if prefixYear.MatchString(c) {
			t.Errorf("focused output contains prefix year: %q", c)
		}
		if doubleSep.MatchString(c) {
			t.Errorf("focused output contains double separator: %q", c)
		}
		if leadingSep.MatchString(c) {
			t.Errorf("focused output starts with a separator: %q", c)
		}
		if strings.Contains(c, "Ad!min") {
			t.Errorf("focused output contains a mid-word symbol: %q", c)
		}
	}
}

func TestPhaseStatsSum(t *testing.T) {
	cfg := config.Default()
	_, _, st := collect(t, cfg, []string{"corp", "acme", "test"}, []string{"jsmith"})
	var sum int64
	for _, n := range st.Phases {
		sum += n
	}
	if sum != st.Attempts {
		t.Fatalf("phase sum = %d, attempts = %d", sum, st.Attempts)
	}
}

func BenchmarkGenerate11Keywords(b *testing.B) {
	cfg := config.Default()
	keywords := []string{
		"Acme", "Corp", "Network", "Admin", "Istanbul", "Ankara",
		"Turkey", "IT", "Helpdesk", "Password", "Welcome",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := generate.New(cfg, keywords, nil)
		g.Generate(context.Background(), func(string) bool { return true })
	}
}

func TestRichUserData(t *testing.T) {
	cfg := config.Default()
	users := []generate.User{{
		Username:  "jsmith",
		First:     "John",
		Last:      "Smith",
		BirthYear: 1990,
	}}
	_, set, _ := collectUsers(t, cfg, []string{"corp"}, users)
	for _, want := range []string{
		"jsmith", "jsmith1990", "jsmith1990!", "jsmith!1990",
		"john.smith", "John.Smith", "johnsmith", "JohnSmith",
		"JSmith", "SmithJohn", "smithj", "j.smith",
		"john.smith1990", "JSmith2026",
	} {
		if !set[want] {
			t.Errorf("rich user data missing %q", want)
		}
	}
}

func TestCompleteOutputParity(t *testing.T) {
	cfg := config.Default()
	cfg.Level = 3
	_, set, _ := collect(t, cfg, []string{"corp", "corp.local"}, nil)

	// Representative shapes from the original wordlist_generator.exe output.
	want := []string{
		"corp", "Corp", "CORP", "corp.local", "Corp.local", "CORP.LOCAL",
		"corp2024*", "corp_24", "corp.local2024.", "Corp@24",
		"CORP.LOCAL24*", "corp.local.2024", "corp.local_99",
		"corp99!", "corp.local123!",
		"!Corp", ".Corp", "_Corp", "$Corp", "%01Corp", "*1999Corp",
		"2001Corp.local%", "!Corp.Corp.local71",
		"CORPcorp", "corpCORP", "corp.localCORP",
		"corp.localCorp.local", "Corp.localCorp.local", "CORP.LOCALCORP.LOCAL",
		"%01Corp.local.Corp", "2019CORP.LOCAL_Corp.",
		"01Corp-CORP.LOCAL123!", "2000CORPCorp.local-", "23Corp.local-Corp",
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("complete output missing %q", w)
		}
	}
}

func TestFocusedHasNoPrefixes(t *testing.T) {
	cfg := config.Default()
	_, set, _ := collect(t, cfg, []string{"corp", "corp.local"}, nil)
	for _, w := range []string{"!Corp", ".Corp", "%01Corp", "*1999Corp"} {
		if set[w] {
			t.Errorf("default focused output contains prefix shape %q", w)
		}
	}
}
