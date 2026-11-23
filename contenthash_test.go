package contenthash

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type vector struct {
	Name string   `json:"name"`
	Func string   `json:"func"`
	Args []string `json:"args"`
	Hash string   `json:"hash"`
}

// TestVectors checks every function against testdata/vectors.json, which an
// independent implementation of the spec generates.
func TestVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []vector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) == 0 {
		t.Fatal("no vectors")
	}
	funcs := map[string]func(a []string) string{
		"Tag":            func(a []string) string { return V2Tag(a[0], a[1:]) },
		"TagChild":       func(a []string) string { return V2TagChild(a[0], a[1], a[2]) },
		"Question":       func(a []string) string { return V2Question(a[0], a[1], a[2], a[3]) },
		"Passage":        func(a []string) string { return V2Passage(a[0], a[1], a[2]) },
		"V1Tag":          func(a []string) string { return V1Tag(a[0], a[1], a[2]) },
		"V1Question":     func(a []string) string { return V1Question(a[0], a[1]) },
		"V1UserQuestion": func(a []string) string { return V1UserQuestion(a[0], a[1], a[2], a[3]) },
		"V1Passage":      func(a []string) string { return V1Passage(a[0]) },
	}
	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			f, ok := funcs[v.Func]
			if !ok {
				t.Fatalf("unknown func %q", v.Func)
			}
			if got := f(v.Args); got != v.Hash {
				t.Errorf("%s%q = %s, want %s", v.Func, v.Args, got, v.Hash)
			}
		})
	}
}

// TestUnicodeVersion fails when a dependency bump changes the Unicode data
// normalization uses, so that change is made deliberately: it changes v2
// hashes of characters assigned after Unicode 15.0.
func TestUnicodeVersion(t *testing.T) {
	if norm.Version != "15.0.0" {
		t.Errorf("x/text/unicode/norm is Unicode %s; v2 is defined on 15.0.0", norm.Version)
	}
	if cases.UnicodeVersion != "15.0.0" {
		t.Errorf("x/text/cases is Unicode %s; v2 is defined on 15.0.0", cases.UnicodeVersion)
	}
}

// TestFormatTable checks the written-out format characters against Go's
// Unicode tables, when Go's are also Unicode 15.0.
func TestFormatTable(t *testing.T) {
	if unicode.Version != "15.0.0" {
		t.Skipf("Go's unicode tables are %s, not 15.0.0", unicode.Version)
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		want := unicode.Is(unicode.Cf, r) && r != 0x200C && r != 0x200D
		if isFormat(r) != want {
			t.Errorf("isFormat(%U) = %v", r, !want)
		}
		if isSpace(r) != (unicode.IsSpace(r) || unicode.Is(unicode.Cc, r)) {
			t.Errorf("isSpace(%U) = %v", r, isSpace(r))
		}
	}
}

// Invalid UTF-8 can't be a JSON vector, so it's checked here: each invalid
// byte becomes U+FFFD before normalizing.
func TestInvalidUTF8(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a\xffb", "a\ufffdb"},
		{"a\xff\xfeb", "a\ufffd\ufffdb"},
		{"caf\xc3", "caf\ufffd"},
		{"\xe2\x80", "\ufffd\ufffd"},
	}
	for _, tt := range tests {
		if got := V2Normalize(tt.in); got != tt.want {
			t.Errorf("V2Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if V2Tag("", []string{"\xff"}) != V2Tag("", []string{"\ufffd"}) {
		t.Error("an invalid byte hashes as U+FFFD")
	}
}

func TestV2TagPanicsWithNoNames(t *testing.T) {
	for _, path := range [][]string{nil, {}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("V2Tag(%q) didn't panic", path)
				}
			}()
			V2Tag("owner", path)
		}()
	}
}

func TestVersion(t *testing.T) {
	v2 := V2Tag("", []string{"Library"})
	v1 := V1Tag("", "", "Library")
	tests := []struct {
		h    string
		want int
	}{
		{v2, 2},
		{v1, 1},
		{"", 0},
		{"v2:", 0},
		{"v2:abc", 0},
		{V2Prefix + v1 + "0", 0},
		{V2Prefix + strings.Repeat("z", 64), 0},
		{V2Prefix + strings.ToUpper(v1), 0},
		{strings.ToUpper(v1), 0},
		{"v3:" + v1, 0},
	}
	for _, tt := range tests {
		if got := Version(tt.h); got != tt.want {
			t.Errorf("Version(%q) = %d, want %d", tt.h, got, tt.want)
		}
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	for _, s := range []string{"  Newton\u2019s  Laws ", "Stra\u00dfe", "\ufb01nance x\u00b2", "\u200bA\u2013B", "Cafe\u0301", "\u0130stanbul", "\u01c5"} {
		once := V2Normalize(s)
		if twice := V2Normalize(once); twice != once {
			t.Errorf("V2Normalize(%q) = %q, but normalizing that gives %q", s, once, twice)
		}
	}
}
