package textutil

import (
	"reflect"
	"testing"
)

func TestFormsBasic(t *testing.T) {
	got := Forms("corp", Options{})
	want := []string{"corp", "Corp", "CORP"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Forms(corp) = %#v, want %#v", got, want)
	}
}

func TestFormsDotted(t *testing.T) {
	got := Forms("corp.local", Options{})
	want := []string{"corp.local", "Corp.local", "CORP.LOCAL"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Forms(corp.local) = %#v, want %#v", got, want)
	}
}

func TestFormsPreservesMixedSource(t *testing.T) {
	got := Forms("CoRp", Options{})
	want := []string{"corp", "Corp", "CORP", "CoRp"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Forms(CoRp) = %#v, want %#v", got, want)
	}
}

func TestFormsTurkish(t *testing.T) {
	got := Forms("İstanbul", Options{Turkish: true})
	for _, want := range []string{"istanbul", "İstanbul", "İSTANBUL", "Istanbul", "ISTANBUL"} {
		if !contains(got, want) {
			t.Errorf("Forms(İstanbul) missing %q: %#v", want, got)
		}
	}
	if len(got) != 5 {
		t.Fatalf("Forms(İstanbul) has %d entries, want 5: %#v", len(got), got)
	}
}

func TestFormsTurkishForcedASCII(t *testing.T) {
	got := Forms("Istanbul", Options{Turkish: true, ForceTurkish: true})
	if !contains(got, "ıstanbul") {
		t.Fatalf("Turkish lower variant missing: %#v", got)
	}
	got = Forms("admin", Options{Turkish: true, ForceTurkish: true})
	if !contains(got, "ADMİN") {
		t.Fatalf("Turkish upper variant missing: %#v", got)
	}
	got = Forms("Istanbul", Options{Turkish: true})
	if contains(got, "ıstanbul") {
		t.Fatalf("forced variant generated without ForceTurkish: %#v", got)
	}
}

func TestFormsPascal(t *testing.T) {
	got := Forms("active_directory", Options{Pascal: true})
	if !contains(got, "ActiveDirectory") {
		t.Fatalf("Pascal form missing: %#v", got)
	}
}

func TestFormsCamel(t *testing.T) {
	got := Forms("active_directory", Options{Camel: true})
	if !contains(got, "activeDirectory") {
		t.Fatalf("camel form missing: %#v", got)
	}
	if !contains(got, "activedirectory") {
		t.Fatalf("lower camel form missing: %#v", got)
	}
}

func TestFoldASCII(t *testing.T) {
	cases := map[string]string{
		"İstanbul":     "Istanbul",
		"Şirket Kış":   "Sirket Kis",
		"çğıöşü":       "cgiosu",
		"crème brûlée": "creme brulee",
		"plain":        "plain",
	}
	for in, want := range cases {
		if got := FoldASCII(in); got != want {
			t.Errorf("FoldASCII(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTurkishCasing(t *testing.T) {
	if got := TurkishLower("Iİ"); got != "ıi" {
		t.Errorf("TurkishLower(Iİ) = %q, want ıi", got)
	}
	if got := TurkishUpper("iı"); got != "İI" {
		t.Errorf("TurkishUpper(iı) = %q, want İI", got)
	}
	if got := Lower("İSTANBUL"); got != "istanbul" {
		t.Errorf("Lower(İSTANBUL) = %q, want istanbul", got)
	}
}

func TestTitleEach(t *testing.T) {
	cases := map[string]string{
		"corp.local":       "Corp.Local",
		"active_directory": "Active_Directory",
		"corp-local":       "Corp-Local",
	}
	for in, want := range cases {
		if got := TitleEach(in); got != want {
			t.Errorf("TitleEach(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCamel(t *testing.T) {
	if got := Camel("corp.local"); got != "corpLocal" {
		t.Errorf("Camel(corp.local) = %q, want corpLocal", got)
	}
	if got := Camel("corp"); got != "Corp" {
		t.Errorf("Camel(corp) = %q, want Corp", got)
	}
}

func TestJoinWords(t *testing.T) {
	if got, changed := JoinWords("Acme  Corp"); !changed || got != "AcmeCorp" {
		t.Fatalf("JoinWords = %q, %v", got, changed)
	}
	if _, changed := JoinWords("corp"); changed {
		t.Fatal("JoinWords changed a single word")
	}
}

func TestPredicates(t *testing.T) {
	if !IsASCII("corp.local") || IsASCII("şirket") {
		t.Error("IsASCII mismatch")
	}
	if !IsAlnumOnly("Admin123") || IsAlnumOnly("corp.local") {
		t.Error("IsAlnumOnly mismatch")
	}
	if !HasTurkish("İstanbul") || HasTurkish("Istanbul") {
		t.Error("HasTurkish mismatch")
	}
	if !HasUpper("Corp") || HasUpper("corp") {
		t.Error("HasUpper mismatch")
	}
	if got := LastRune("corp!"); got != '!' {
		t.Errorf("LastRune = %q", got)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
