# contenthash

Deterministic SHA-256 content hashes for hierarchical content, computed from text alone, with golden test vectors.

Use it when two systems need to agree on whether a piece of content already exists, and the only thing they share is its text. An importer that sees `Science › Biology › Cells` in a file doesn't know what id the database gave that node last time, but it can compute the node's hash and look it up. The application that lets people rename and move nodes computes the same hash for the same path, so the importer still finds them.

It hashes three kinds of content:

- **Tags**: nodes in a hierarchy (folders, categories, topics), identified by their path of names.
- **Questions**: a prompt and its answer, shared across tags or scoped to one.
- **Passages**: titled text belonging to a tag.

## Install

```
go get github.com/studyguides-com/contenthash
```

Requires Go 1.24 or later. The only dependency is `golang.org/x/text`, for Unicode normalization and case folding.

## Quick start

```go
import "github.com/studyguides-com/contenthash"

// A tag, from its path of names, root first.
cells := contenthash.V2Tag("", []string{"Science", "Biology", "Cells"})

// The same name under another parent is a different tag.
other := contenthash.V2Tag("", []string{"Health", "Biology", "Cells"}) // != cells

// Typing differences don't make a different tag.
same := contenthash.V2Tag("", []string{"science", "BIOLOGY", " Cells "}) // == cells

// The first argument is the owner: "" for shared, public content, or a
// user's id for their own. Each owner's content is separate.
mine := contenthash.V2Tag("user-123", []string{"Science", "Biology", "Cells"}) // != cells

// A question that may appear under many tags.
q := contenthash.V2Question("", "", "What is a cell?", "The basic unit of life")

// A passage in the Cells tag.
p := contenthash.V2Passage("", cells, "Inside a Cell")
```

Every v2 hash is a string like `v2:3f9a…`: the version, a colon, and 64 lowercase hex digits.

## How it works

**Text only.** Every input is text the caller already has: names, prompts, answers, titles, and an owner string. Nothing depends on database ids, timestamps or order of insertion, so any system with the same text gets the same hash.

**Tags chain through their parent.** A tag's hash is computed from its parent's hash and its own name. A root has an empty parent hash. So a tag's hash depends on every name on its path, and two tags with the same name under different parents never collide.

**Owner.** Pass an empty owner for shared, public content. Pass an owner (any stable string, such as a user id) for content that belongs to someone, so two people's identical folders don't collide with each other or with public content.

**Question scope.** A question with an empty scope is identified by its text alone, so the same question under two tags is the same question. Pass a tag's hash as the scope when a question belongs to that one tag and should stay separate from identical questions elsewhere.

**Normalization.** The hash decides whether two pieces of text are the same content, so it ignores differences that come from how text was typed or generated rather than what it says. Store and display the original text; only the hash is computed from the normalized form. `V2Normalize` folds together:

- **Case:** `Genes With Introns` and `Genes with Introns`; `Straße` and `STRASSE`.
- **Quote and dash styles:** `Newton’s`, `Newtonʼs` and `Newton's`; `“Blue”` and `"Blue"`; `12″` and `12"`; and en dashes, em dashes, minus signs and non-breaking hyphens with `-`. Primes become `'`.
- **Invisible characters:** zero-width spaces, byte order marks, word joiners, soft hyphens, direction marks and variation selectors are removed, so `⭐️` and `⭐` are the same. The zero-width joiner and non-joiner are kept inside a name, since they change how it renders, and so are the tag characters that make flags such as England's.
- **Whitespace:** leading and trailing whitespace is removed, and every run inside (spaces, tabs, newlines, no-break spaces, control characters) becomes one space.
- **Compatibility forms:** `ﬁ` and `fi`, full-width `Ｓ` and `S`, and an accent typed as one character or as a letter plus a combining mark. This also merges forms that can differ in meaning: `x²` and `x2`, `10³` and `103`, `CO₂` and `CO2`. That is intended for names; for questions, both the prompt and the answer would have to merge for two questions to collide.

The exact rules are in the specification below.

## Examples

### Building a tree top-down

When you already have the parent's hash, `V2TagChild` takes one step instead of rehashing the whole path:

```go
root := contenthash.V2TagChild("", "", "Science")
biology := contenthash.V2TagChild("", root, "Biology")
cells := contenthash.V2TagChild("", biology, "Cells")
// cells == contenthash.V2Tag("", []string{"Science", "Biology", "Cells"})
```

### Finding existing content on import

```go
h := contenthash.V2Tag("", path)
existing, found := store.TagByHash(h) // your lookup
if !found {
	store.CreateTag(h, path)
}
```

Store the hash in a column with a unique index, and the database enforces one row per normalized path.

`V2Tag` panics on an empty path, since a tag has at least a name. Check paths that come from data before calling it.

### Renaming or moving a tag

Renaming or moving a tag changes its path, so it changes the hash of the tag **and of every tag below it**. Recompute them top-down in the same transaction:

```go
func rehash(t *Node, parentHash string) {
	t.Hash = contenthash.V2TagChild(t.Owner, parentHash, t.Name)
	for _, child := range t.Children {
		rehash(child, t.Hash)
	}
}
```

Questions scoped to a rehashed tag need their hash recomputed too, and so do its passages.

### A user's own content

```go
const owner = "user-123"
mine := contenthash.V2Tag(owner, []string{"Biology", "Unit 1", "Vocabulary"})
also := contenthash.V2Tag(owner, []string{"Chemistry", "Unit 1", "Vocabulary"}) // different folder, different hash
q := contenthash.V2Question(owner, mine, "What is osmosis?", "Diffusion of water across a membrane")
```

### Moving from v1 hashes

The `V1` functions compute the earlier, unprefixed hashes, so a migration can tell which stored hashes it knows how to replace:

```go
switch contenthash.Version(stored) {
case 2:
	// already v2
case 1:
	if stored == contenthash.V1Tag(owner, parentName, name) {
		// a v1 hash of this tag's current name: replace it
		stored = contenthash.V2Tag(owner, path)
	} else {
		// a v1-shaped hash this tag's name doesn't produce: hashed under an
		// older name, or another rule; look into it before replacing
	}
}
```

During a migration both versions exist side by side; the `v2:` prefix tells them apart.

**Rows that are separate today can share a v2 hash.** v2 folds case, punctuation and invisible characters, so rows such as `Ohm's Law` and `Ohm’s Law` under one parent get one hash, and writing both fails on a unique index. Compute every row's v2 hash first, group the rows by it, and merge each group before writing.

## API

| Function | Returns |
|---|---|
| `V2Tag(owner string, path []string) string` | The v2 hash of the tag at the end of `path` (root first). Panics if `path` is empty. |
| `V2TagChild(owner, parentHash, name string) string` | The v2 hash of `name` under the tag with `parentHash`; an empty `parentHash` makes a root. |
| `V2Question(owner, scope, prompt, answer string) string` | The v2 hash of a question; `scope` is `""` (shared) or a tag's v2 hash. |
| `V2Passage(owner, tagHash, title string) string` | The v2 hash of a passage in the tag with `tagHash`. |
| `V2Normalize(s string) string` | The text a v2 hash is computed from. |
| `Version(h string) int` | `2` for a v2 hash, `1` for 64 lowercase hex digits, `0` otherwise. |
| `V1Tag(owner, parentName, name string) string` | The v1 tag hash. |
| `V1Question(prompt, answer string) string` | The v1 hash of a shared question. |
| `V1UserQuestion(owner, topicID, prompt, answer string) string` | The v1 hash of an owned question. |
| `V1Passage(title string) string` | The v1 passage hash. |

`V2Prefix` is the `"v2:"` constant. Every function is named for the hash version it computes, so a later version adds functions alongside these rather than changing them.

## Specification

The exact rules, for implementing this in another language. `testdata/reference.py` is one such implementation.

### Version 2

A v2 hash is `v2:` followed by the lowercase hex SHA-256 of an encoded list of fields.

**Encoding.** Each field is its length in bytes (decimal, UTF-8), a colon, its bytes, and a comma. The fields are concatenated in order. `["tag", "", "", "library"]` encodes as `3:tag,0:,0:,7:library,`.

**Fields.**

| Function | Fields |
|---|---|
| `V2TagChild(owner, parentHash, name)` | `"tag"`, owner, parentHash, normalized name |
| `V2Tag(owner, path)` | `V2TagChild` applied down the path from the root, starting with an empty parentHash |
| `V2Question(owner, scope, prompt, answer)` | `"question"`, owner, scope, normalized prompt, normalized answer |
| `V2Passage(owner, tagHash, title)` | `"passage"`, owner, tagHash, normalized title |

Owner, scope, parentHash and tagHash are used exactly as given, with no normalization. A hash field is the whole `v2:…` string.

**Normalizing.** Names, prompts, answers and titles are normalized in these steps, on Unicode 15.0:

1. **Invalid UTF-8:** decode left to right. Where no complete, valid sequence starts (shortest form, not a surrogate, at most U+10FFFF), that one byte becomes U+FFFD and decoding resumes at the next byte. This gives one U+FFFD per byte: `E2 80` becomes two, where most decoders, which replace a maximal subpart, give one.
2. **Ignorable characters:** every `Default_Ignorable_Code_Point` is removed, except U+200C ZERO WIDTH NON-JOINER, U+200D ZERO WIDTH JOINER and the tag characters U+E0020 to U+E007F. U+2033 DOUBLE PRIME and U+2036 REVERSED DOUBLE PRIME become `"`.
3. **Compatibility and case:** NFKC, then full case folding (`CaseFolding.txt`, statuses C and F), then NFKC again.
4. **Punctuation:**
   - U+2018, U+2019, U+201A, U+201B, U+2032, U+2035 and U+02BC become `'`.
   - U+201C, U+201D, U+201E and U+201F become `"`.
   - U+2010 to U+2015, U+2212, U+2E3A and U+2E3B become `-`.
5. **Whitespace:** whitespace is every character with the `White_Space` property and every control character (category Cc). It is removed from both ends, and each run inside becomes one U+0020. Then U+0020, U+200C and U+200D are removed from both ends.

NFKC is used rather than NFC because the hash is for identity, not display: `ﬁ` and `fi` are the same name. The stored text keeps its original form.

Normalization is fixed to Unicode 15.0. Text with characters assigned after Unicode 15.0 is not guaranteed to hash the same in an implementation using later Unicode data.

**Implementation note.** The Go code does canonical composition itself, from a table of Unicode 15.0 composition pairs (`compositions.go`), and uses `golang.org/x/text` for decomposition and case folding. x/text v0.32.0 gets two things wrong for this: it looks composition pairs up by the low 16 bits of each character, so a character above U+FFFF before a combining mark can compose as if it were a BMP letter; and normalizing a whole string inserts U+034F COMBINING GRAPHEME JOINER after 30 combining marks in a row (UAX #15's Stream-Safe Text Format), which plain NFKC doesn't. So the Go code decomposes one character at a time with x/text, then orders and composes itself. Other implementations should use a plain, correct NFKC; the vectors cover both cases.

### Version 1

SHA-256 hex of the inputs run together, with no prefix, separator or normalization.

| Function | Input |
|---|---|
| `V1Tag(owner, parentName, name)` | owner + parentName + name |
| `V1Question(prompt, answer)` | prompt + answer |
| `V1UserQuestion(owner, topicID, prompt, answer)` | owner + topicID + prompt + answer |
| `V1Passage(title)` | title |

v1 has known weaknesses that v2 fixes:
- A tag depends only on its parent's name, so the same names under different parents collide.
- Inputs run together, so `"AB" + "C"` equals `"A" + "BC"`.
- Any difference in case, punctuation or whitespace makes a new hash.
- Nothing marks which formula made a hash.

## Stability

- **A hash version never changes.** Any change to the output (encoding, normalization, fields, Unicode version) is a new version with a new prefix, and the old version's functions stay.
- **Releases are tagged `vX.Y.Z`**, and a published version is permanent.
- `testdata/vectors.json` is the contract: every release passes it, and other implementations can test against it.

## Development

```
make help                           # list the targets
make check                          # format, vet, vectors and tests (what CI runs)
make differential                   # compare V2Normalize with the reference on ~1.5M inputs
make vectors                        # regenerate testdata/vectors.json
make compositions                   # regenerate compositions.go
make release-check VERSION=vX.Y.Z   # check a release could be made, changing nothing
make release VERSION=vX.Y.Z         # tag main and publish the version
```

`testdata/reference.py` is an independent implementation of the specification in Python. `testdata/gen_vectors.py` generates `testdata/vectors.json` from it, and `make differential` (`testdata/differential.py`) compares `V2Normalize` with it on about 1.5 million inputs: every code point and its decomposed, compatibility and case-folded forms, characters above U+FFFF before combining marks, long runs of combining marks, and random strings, checking that normalizing twice changes nothing in either. CI runs both. The reference derives its character sets from Python's Unicode data and needs Python 3.12, whose Unicode data is 15.0; `testdata/gen_compositions.py` generates the composition table the same way. To change a hash: change the specification here, the reference and the Go code together, run `make vectors`, and `make check` and `make differential` must pass.

## License

MIT. See [LICENSE](LICENSE).
