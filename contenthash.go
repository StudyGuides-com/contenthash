// Package contenthash computes the hashes that identify content in a
// hierarchy (tags, questions and passages) from its text alone, so two
// systems that only share the text agree on what is the same content.
//
// A v2 hash is "v2:" followed by the hex SHA-256 of an encoded list of
// fields: the kind of content first, then its fields in a fixed order. Each
// field is encoded as its length in bytes (decimal), a colon, the bytes, and
// a comma, so no two lists of fields encode the same way. Text fields are
// normalized first (see Normalize); owner, scope and hash fields are used as
// given.
//
// A tag's hash chains through its parent's: the root's parent hash is
// empty, so a tag's hash depends on its owner and every name on its path.
//
// The V1 functions compute the earlier, unprefixed hashes, so stored v1
// hashes can be checked and replaced.
package contenthash

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Prefix starts every v2 hash.
const Prefix = "v2:"

// Kinds of content, the first field of every v2 hash.
const (
	kindTag      = "tag"
	kindQuestion = "question"
	kindPassage  = "passage"
)

// Normalize is the text a hash is computed from: Unicode NFC, with leading
// and trailing whitespace removed and every run of whitespace inside made a
// single space. Case is kept.
func Normalize(s string) string {
	return strings.Join(strings.Fields(norm.NFC.String(s)), " ")
}

// TagChild is the v2 hash of a tag named name under the tag whose hash is
// parentHash; an empty parentHash makes it a root. owner is empty for public
// content.
func TagChild(owner, parentHash, name string) string {
	return hash(kindTag, owner, parentHash, Normalize(name))
}

// Tag is the v2 hash of the tag at the end of path, whose names run from the
// root to the tag. owner is empty for public content. Tag with no names is
// "".
func Tag(owner string, path ...string) string {
	h := ""
	for _, name := range path {
		h = TagChild(owner, h, name)
	}
	return h
}

// Question is the v2 hash of a question. scope is empty for a question that
// may be shared across tags (public content); otherwise it is the hash of the
// one tag the question belongs to.
func Question(owner, scope, prompt, answer string) string {
	return hash(kindQuestion, owner, scope, Normalize(prompt), Normalize(answer))
}

// Passage is the v2 hash of a passage titled title in the tag whose hash is
// tagHash.
func Passage(owner, tagHash, title string) string {
	return hash(kindPassage, owner, tagHash, Normalize(title))
}

// IsV2 reports whether h is a v2 hash.
func IsV2(h string) bool {
	return strings.HasPrefix(h, Prefix) && len(h) == len(Prefix)+sha256.Size*2
}

// hash encodes the fields and returns the prefixed hex SHA-256.
func hash(fields ...string) string {
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(strconv.Itoa(len(f)))
		b.WriteByte(':')
		b.WriteString(f)
		b.WriteByte(',')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return Prefix + hex.EncodeToString(sum[:])
}

// V1Tag is the v1 hash of a tag: SHA-256 of owner + parent name + name,
// run together, unnormalized; owner is empty for public content and
// parentName empty for a root.
func V1Tag(owner, parentName, name string) string {
	return v1(owner + parentName + name)
}

// V1Question is the v1 hash of a public question: prompt + answer.
func V1Question(prompt, answer string) string {
	return v1(prompt + answer)
}

// V1UserQuestion is the v1 hash of a user's question: owner + the topic's id
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
