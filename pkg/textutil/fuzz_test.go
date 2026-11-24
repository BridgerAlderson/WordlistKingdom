package textutil

import "testing"

func FuzzForms(f *testing.F) {
	for _, seed := range []string{
		"corp",
		"corp.local",
		"İstanbul",
		"active_directory",
		"Acme Corp",
		"",
		"coRp",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		forms := Forms(s, Options{Turkish: true, ForceTurkish: true, Camel: true, Pascal: true})
		seen := make(map[string]bool, len(forms))
		for _, form := range forms {
			if form == "" {
				t.Fatal("empty form accepted")
			}
			if seen[form] {
				t.Fatalf("duplicate form %q", form)
			}
			seen[form] = true
		}
	})
}
