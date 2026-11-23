package contenthash

import (
	"encoding/json"
	"os"
	"testing"
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
		"Tag":            func(a []string) string { return Tag(a[0], a[1:]...) },
		"TagChild":       func(a []string) string { return TagChild(a[0], a[1], a[2]) },
		"Question":       func(a []string) string { return Question(a[0], a[1], a[2], a[3]) },
		"Passage":        func(a []string) string { return Passage(a[0], a[1], a[2]) },
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

func TestSameAndDifferent(t *testing.T) {
	const owner = "owner1"
	same := []struct {
		name string
		a, b string
	}{
		{"whitespace", Tag("", "Biology", "Unit 1"), Tag("", " Biology ", "Unit\t\t1")},
		{"NFC", Tag("", "Café"), Tag("", "Café")},
		{"Tag is TagChild down the path", Tag("", "A", "B", "C"), TagChild("", TagChild("", TagChild("", "", "A"), "B"), "C")},
		{"question whitespace", Question("", "", "What is it?", "This"), Question("", "", " What  is it? ", "This ")},
	}
	for _, c := range same {
		if c.a != c.b {
			t.Errorf("%s: %s != %s", c.name, c.a, c.b)
		}
	}
	different := []struct {
		name string
		a, b string
	}{
		{"same names under different parents", Tag("", "Biology", "Unit 1", "Vocabulary"), Tag("", "Chemistry", "Unit 1", "Vocabulary")},
		{"owner", Tag("", "Biology"), Tag(owner, "Biology")},
		{"case", Tag("", "Unit 1"), Tag("", "unit 1")},
		{"names run together differently", Tag("", "AB", "C"), Tag("", "A", "BC")},
		{"a tag and a passage with the same text", TagChild("", "", "X"), Passage("", "", "X")},
		{"question scope", Question(owner, "", "Q", "A"), Question(owner, Tag(owner, "T"), "Q", "A")},
		{"prompt and answer run together differently", Question("", "", "QA", "B"), Question("", "", "Q", "AB")},
	}
	for _, c := range different {
		if c.a == c.b {
			t.Errorf("%s: both %s", c.name, c.a)
		}
	}
}

func TestIsV2(t *testing.T) {
	if !IsV2(Tag("", "Library")) {
		t.Error("a v2 tag hash")
	}
	for _, h := range []string{V1Tag("", "", "Library"), "", "v2:", "v2:abc", Prefix + V1Tag("", "", "Library") + "0"} {
		if IsV2(h) {
			t.Errorf("IsV2(%q)", h)
		}
	}
}

func TestNoNames(t *testing.T) {
	if got := Tag("owner"); got != "" {
		t.Errorf("Tag with no names = %q", got)
	}
}
