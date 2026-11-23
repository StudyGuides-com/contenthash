// Package contenthash computes the hashes that identify content in a
// hierarchy (tags, questions and passages) from its text alone, so two
// systems that only share the text agree on what is the same content.
//
// Every function is named for the hash version it computes. V2 is current:
// "v2:" followed by the hex SHA-256 of an encoded list of fields, the kind of
// content first, text normalized by V2Normalize so typing differences (case,
// quote and dash styles, invisible characters, whitespace) don't make new
// content. A tag's hash chains through its parent's, so it depends on its
// owner and every name on its path.
//
// V1 functions compute the earlier, unprefixed hashes, so stored v1 hashes
// can be checked and replaced. A hash version's output never changes.
//
// Normalization is fixed to Unicode 15.0. Text containing characters
// assigned after Unicode 15.0 is not guaranteed to hash the same under a
// later version.
package contenthash

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
)

// V2Prefix starts every v2 hash.
const V2Prefix = "v2:"

// Kinds of content, the first field of every v2 hash.
const (
	kindTag      = "tag"
	kindQuestion = "question"
	kindPassage  = "passage"
)

// V2Normalize is the text a v2 hash is computed from:
//
//  1. Decoding left to right, a byte that doesn't start a valid UTF-8
//     sequence becomes U+FFFD, and decoding resumes at the next byte.
//  2. Default ignorable code points are removed, except the zero-width
//     non-joiner and joiner (U+200C, U+200D) and the tag characters
//     (U+E0020-E007F). Double primes become ".
//  3. NFKC, full case folding (Unicode CaseFolding.txt, statuses C and F),
//     then NFKC again.
//  4. Look-alike punctuation becomes ASCII: single quotes, primes and the
//     modifier letter apostrophe become ', double quotes become ", and
//     hyphens and dashes become -.
//  5. Whitespace (Unicode White_Space and control characters) is trimmed from
//     both ends, and each run inside becomes one space. Joiners left at
//     either end are trimmed too.
func V2Normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if isIgnored(r) {
			continue
		}
		if q, ok := doublePrimes[r]; ok {
			r = q
		}
		b.WriteRune(r)
	}
	s = nfkc(cases.Fold().String(nfkc(b.String())))

	b.Reset()
	space := false
	for _, r := range s {
		if isSpace(r) {
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		if a, ok := punctuation[r]; ok {
			r = a
		}
		r = foldCherokee(r)
		b.WriteRune(r)
	}
	return strings.Trim(b.String(), " \u200c\u200d")
}

// V2TagChild is the v2 hash of a tag named name under the tag whose v2 hash
// is parentHash; an empty parentHash makes it a root. owner is empty for
// public content.
func V2TagChild(owner, parentHash, name string) string {
	return hash(kindTag, owner, parentHash, V2Normalize(name))
}

// V2Tag is the v2 hash of the tag at the end of path, whose names run from
// the root to the tag. owner is empty for public content. It panics if path
// is empty: a tag has at least a name.
func V2Tag(owner string, path []string) string {
	if len(path) == 0 {
		panic("contenthash: V2Tag needs at least one name")
	}
	h := ""
	for _, name := range path {
		h = V2TagChild(owner, h, name)
	}
	return h
}

// V2Question is the v2 hash of a question. scope is empty for a question that
// may be shared across tags; otherwise it is the v2 hash of the one tag the
// question belongs to.
func V2Question(owner, scope, prompt, answer string) string {
	return hash(kindQuestion, owner, scope, V2Normalize(prompt), V2Normalize(answer))
}

// V2Passage is the v2 hash of a passage titled title in the tag whose v2 hash
// is tagHash.
func V2Passage(owner, tagHash, title string) string {
	return hash(kindPassage, owner, tagHash, V2Normalize(title))
}

// Version is the version of a stored hash: 2 for "v2:" and 64 lowercase hex
// digits, 1 for 64 lowercase hex digits alone, and 0 for anything else. A v1
// hash can't be told from any other unprefixed SHA-256; Version only checks
// its form.
func Version(h string) int {
	switch {
	case strings.HasPrefix(h, V2Prefix) && isHash(h[len(V2Prefix):]):
		return 2
	case isHash(h):
		return 1
	}
	return 0
}

func isHash(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// hash encodes the fields and returns the prefixed hex SHA-256. Each field is
// its length in bytes, a colon, its bytes and a comma.
func hash(fields ...string) string {
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(strconv.Itoa(len(f)))
		b.WriteByte(':')
		b.WriteString(f)
		b.WriteByte(',')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return V2Prefix + hex.EncodeToString(sum[:])
}

// V1Tag is the v1 hash of a tag: SHA-256 of owner + parent name + name,
// run together, unnormalized; owner is empty for public content and
// parentName empty for a root.
func V1Tag(owner, parentName, name string) string {
	return v1(owner + parentName + name)
}

// V1Question is the v1 hash of a shared question: prompt + answer.
func V1Question(prompt, answer string) string {
	return v1(prompt + answer)
}

// V1UserQuestion is the v1 hash of an owned question: owner + the topic's id
// + prompt + answer.
func V1UserQuestion(owner, topicID, prompt, answer string) string {
	return v1(owner + topicID + prompt + answer)
}

// V1Passage is the v1 hash of a passage: its title.
func V1Passage(title string) string {
	return v1(title)
}

func v1(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
