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

Requires Go 1.24 or later. The only dependency is `golang.org/x/text`, for Unicode normalization.

## Quick start

```go
import "github.com/studyguides-com/contenthash"

// A tag, from its path of names, root first.
cells := contenthash.Tag("", "Science", "Biology", "Cells")

// The same name under another parent is a different tag.
other := contenthash.Tag("", "Health", "Biology", "Cells") // != cells

// A question that may appear under many tags.
q := contenthash.Question("", "", "What is a cell?", "The basic unit of life")

// A passage in the Cells tag.
p := contenthash.Passage("", cells, "Inside a Cell")
```

Every hash is a string like `v2:3f9a…` (the version, a colon, 64 hex characters).

## How it works

**Text only.** Every input is text the caller already has: names, prompts, answers, titles, and an owner string. Nothing depends on database ids, timestamps or order of insertion, so any system with the same text gets the same hash.

**Tags chain through their parent.** A tag's hash is computed from its parent's hash and its own name. A root has an empty parent hash. So a tag's hash depends on every name on its path, and two tags with the same name under different parents never collide.

**Owner.** Pass an empty owner for shared, public content. Pass an owner (any stable string, such as a user id) for content that belongs to someone, so two people's identical folders don't collide with each other or with public content.

**Question scope.** A question with an empty scope is identified by its text alone, so the same question under two tags is the same question. Pass a tag's hash as the scope when a question belongs to that one tag and should stay separate from identical questions elsewhere.

**Normalization.** Names, prompts, answers and titles are normalized before hashing, so invisible differences don't create new content:

- Unicode NFC, so `é` typed as one character or as `e` plus a combining accent is the same.
- Leading and trailing whitespace removed, and runs of whitespace (spaces, tabs, newlines, no-break spaces) collapsed to one space.
- Case is kept: `Unit 1` and `unit 1` are different.

`Normalize` is exported, so callers can store or compare names the same way.

## Examples

### Building a tree top-down

When you already have the parent's hash, `TagChild` takes one step instead of rehashing the whole path:

```go
root := contenthash.TagChild("", "", "Science")
biology := contenthash.TagChild("", root, "Biology")
cells := contenthash.TagChild("", biology, "Cells")
// cells == contenthash.Tag("", "Science", "Biology", "Cells")
```

### Finding existing content on import

```go
h := contenthash.Tag("", path...)
existing, found := store.TagByHash(h) // your lookup
if !found {
	store.CreateTag(h, path)
}
```

Store the hash in a column with a unique index, and the database enforces one row per path.

### Renaming or moving a tag

Renaming or moving a tag changes its path, so it changes the hash of the tag **and of every tag below it**. Recompute them top-down in the same transaction:

```go
func rehash(t *Node, parentHash string) {
	t.Hash = contenthash.TagChild(t.Owner, parentHash, t.Name)
	for _, child := range t.Children {
		rehash(child, t.Hash)
	}
}
```

Questions scoped to a rehashed tag need their hash recomputed too; passages always do.

### A user's own content

```go
const owner = "user-123"
mine := contenthash.Tag(owner, "Biology", "Unit 1", "Vocabulary")
also := contenthash.Tag(owner, "Chemistry", "Unit 1", "Vocabulary") // different folder, different hash
q := contenthash.Question(owner, mine, "What is osmosis?", "Diffusion of water across a membrane")
```

### Moving from v1 hashes

The `V1` functions compute the earlier, unprefixed hashes, so a migration can check what's stored and replace it:

```go
if !contenthash.IsV2(stored) {
	if stored == contenthash.V1Tag(owner, parentName, name) {
		// a known v1 hash: safe to replace with the v2 one
	}
	newHash := contenthash.Tag(owner, path...)
	// ...
}
```

During a migration both versions exist side by side; the `v2:` prefix tells them apart.

## API

| Function | Returns |
|---|---|
| `Tag(owner string, path ...string) string` | The v2 hash of the tag at the end of `path` (root first). `""` when `path` is empty. |
| `TagChild(owner, parentHash, name string) string` | The v2 hash of `name` under the tag with `parentHash`; an empty `parentHash` makes a root. |
| `Question(owner, scope, prompt, answer string) string` | The v2 hash of a question; `scope` is `""` (shared) or a tag hash. |
| `Passage(owner, tagHash, title string) string` | The v2 hash of a passage in the tag with `tagHash`. |
| `Normalize(s string) string` | The text a hash is computed from. |
| `IsV2(h string) bool` | Whether `h` is a well-formed v2 hash. |
| `V1Tag(owner, parentName, name string) string` | The v1 tag hash. |
| `V1Question(prompt, answer string) string` | The v1 hash of a shared question. |
| `V1UserQuestion(owner, topicID, prompt, answer string) string` | The v1 hash of an owned question. |
| `V1Passage(title string) string` | The v1 passage hash. |

`Prefix` is the `"v2:"` constant.

## Specification

The exact rules, for implementing this in another language. `testdata/gen_vectors.py` is one such implementation.

### Version 2

A v2 hash is `v2:` followed by the lowercase hex SHA-256 of an encoded list of fields.

**Encoding.** Each field is its length in bytes (decimal, UTF-8), a colon, its bytes, and a comma. The fields are concatenated in order. `["tag", "", "", "Library"]` encodes as `3:tag,0:,0:,7:Library,`.

**Normalizing.** Text fields (names, prompts, answers, titles) are normalized first:

1. Unicode NFC.
2. Leading and trailing whitespace removed, and every run of whitespace inside replaced by one space. Whitespace is the Unicode `White_Space` property.
3. Case is kept.

Owner, scope and hash fields are used exactly as given.

**Fields.**

| Function | Fields |
|---|---|
| `TagChild(owner, parentHash, name)` | `"tag"`, owner, parentHash, name |
| `Tag(owner, path...)` | `TagChild` applied down the path from the root, starting with an empty parentHash |
| `Question(owner, scope, prompt, answer)` | `"question"`, owner, scope, prompt, answer |
| `Passage(owner, tagHash, title)` | `"passage"`, owner, tagHash, title |

### Version 1

SHA-256 hex of the inputs run together, with no prefix, separator or normalization.

| Function | Input |
|---|---|
| `V1Tag(owner, parentName, name)` | owner + parentName + name |
| `V1Question(prompt, answer)` | prompt + answer |
| `V1UserQuestion(owner, topicID, prompt, answer)` | owner + topicID + prompt + answer |
| `V1Passage(title)` | title |

v1 has known weaknesses that v2 fixes: a tag depends only on its parent's name, so the same names under different parents collide; inputs run together, so `"AB" + "C"` equals `"A" + "BC"`; and nothing marks which formula made a hash.

## Stability

- **A hash version never changes.** Any change to the output (encoding, normalization, fields) is a new version with a new prefix, and the old version's functions stay.
- **Releases are tagged `vX.Y.Z`**, and a published version is permanent.
- `testdata/vectors.json` is the contract: every release passes it, and other implementations can test against it.

## Development

```
make help                           # list the targets
make check                          # format, vet, vectors and tests (what CI runs)
make vectors                        # regenerate testdata/vectors.json
make release-check VERSION=vX.Y.Z   # check a release could be made, changing nothing
make release VERSION=vX.Y.Z         # tag main and publish the version
```

`testdata/vectors.json` is generated by `testdata/gen_vectors.py`, an independent implementation of the specification, so the Go code is checked against the spec rather than against itself. To change a hash: change the specification here, the generator and the Go code together, run `make vectors`, and `make check` must pass.

## License

MIT. See [LICENSE](LICENSE).
