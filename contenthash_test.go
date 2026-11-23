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

// defaultIgnorable is Default_Ignorable_Code_Point as DerivedCoreProperties
// derives it: Other_Default_Ignorable_Code_Point + Cf + Variation_Selector -
// White_Space - FFF9..FFFB - 13430..1343F - Prepended_Concatenation_Mark.
func defaultIgnorable(r rune) bool {
	if !unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Cf, unicode.Variation_Selector) {
		return false
	}
	return !unicode.In(r, unicode.White_Space, unicode.Prepended_Concatenation_Mark) &&
		!(r >= 0xFFF9 && r <= 0xFFFB) && !(r >= 0x13430 && r <= 0x1343F)
}

// TestCharacterTables checks the written-out character sets against Go's
// Unicode tables, when Go's are also Unicode 15.0.
func TestCharacterTables(t *testing.T) {
	if unicode.Version != "15.0.0" {
		t.Skipf("Go's unicode tables are %s, not 15.0.0", unicode.Version)
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		want := defaultIgnorable(r) && r != 0x200C && r != 0x200D && !(r >= 0xE0020 && r <= 0xE007F)
		if isIgnored(r) != want {
			t.Errorf("isIgnored(%U) = %v", r, !want)
		}
		if isSpace(r) != (unicode.IsSpace(r) || unicode.Is(unicode.Cc, r)) {
			t.Errorf("isSpace(%U) = %v", r, isSpace(r))
		}
	}
}

// Invalid UTF-8 can't be a JSON vector, so it's checked here: decoding left
// to right, a byte that doesn't start a valid sequence becomes U+FFFD and
// decoding resumes at the next byte. Most decoders replace a maximal subpart
// with one U+FFFD instead ("\xe2\x80" gives one); this rule gives one per
// byte.
func TestInvalidUTF8(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a\xffb", "a\ufffdb"},
		{"a\xff\xfeb", "a\ufffd\ufffdb"},
		{"caf\xc3", "caf\ufffd"},
		{"\xe2\x80", "\ufffd\ufffd"},                     // a truncated sequence
		{"\xed\xa0\x80", "\ufffd\ufffd\ufffd"},           // a surrogate
		{"\xc0\x80", "\ufffd\ufffd"},                     // an overlong encoding
		{"\xf4\x90\x80\x80", "\ufffd\ufffd\ufffd\ufffd"}, // above U+10FFFF
		{"\xe2\x80\x99s", "'s"},                          // a valid right quote is unaffected
		{"x\xe2\x80\x99\xe2\x80", "x'\ufffd\ufffd"},      // a valid sequence, then a truncated one
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
	for _, s := range []string{"  Newton\u2019s  Laws ", "Stra\u00dfe", "\ufb01nance x\u00b2", "\u200bA\u2013B", "Cafe\u0301", "\u0130stanbul", "\u01c5", "12\u2033", "\u200d a \u200c", "\u00b4x"} {
		once := V2Normalize(s)
		if twice := V2Normalize(once); twice != once {
			t.Errorf("V2Normalize(%q) = %q, but normalizing that gives %q", s, once, twice)
		}
	}
}

// FuzzV2Normalize checks that normalizing twice changes nothing.
func FuzzV2Normalize(f *testing.F) {
	for _, s := range []string{"", "Science", "  Newton\u2019s  Laws ", "Stra\u00dfe", "\ufb01 x\u00b2 12\u2033", "\u200bA\u2013B\ufe0f", "\u13a0\uab70", "\u00b4\u0301", "\xff\xe2\x80"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		once := V2Normalize(s)
		if twice := V2Normalize(once); twice != once {
			t.Errorf("V2Normalize(%q) = %q, but normalizing that gives %q", s, once, twice)
		}
	})
}
