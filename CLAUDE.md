@../study-guides-wiki/_meta/agent-entrypoint.md
@../study-guides-wiki/repos/contenthash.md

# contenthash

The content hash the Study Guides ecosystem identifies tags, questions and
passages by, computed from text alone, so the pipeline and the web app get the
same hash for the same content. Its design and its consumers live in the wiki
note imported above. Read it before changing anything here.

## Ground rules

- **This repo is public; nothing but the hash goes in it.** No content, no
  client or product names, no schema, no database or service code. A Go
  module version, once fetched by the public proxy, is served forever, so
  anything committed and tagged here is published permanently.
- **Standard library plus `golang.org/x/text`** (NFKC and case folding). No
  other dependencies. v2 is defined on Unicode 15.0: `TestUnicodeVersion`
  fails if a dependency bump changes it, and the vector generator needs
  Python 3.12 (Unicode 15.0).
- **The golden vectors are the contract.** `testdata/vectors.json` maps
  inputs to expected hashes. It is generated from `testdata/reference.py`, an
  independent implementation of the spec in the README: change the spec, the
  reference and the Go code together, regenerate, and the Go tests and
  `make differential` must pass. Keep the reference independent: derive from
  Python's Unicode data, never copy the Go tables.
- **Never change an existing hash.** Any change to the output (encoding,
  normalization, fields) is a new hash version with its own prefix and new
  vectors; the old version stays computable until every stored hash has been
  rehashed.
- **Tags are releases.** Tag `vX.Y.Z` on `main` only when the user says so;
  consumers pin the version in their `go.mod`.

## Commands

```bash
make check                          # format, vet, vectors and tests; CI runs this
make differential                   # Go against the reference on ~1.5M inputs (about a minute)
make vectors                        # regenerate testdata/vectors.json (Python 3.12)
make compositions                   # regenerate compositions.go (Python 3.12)
make release-check VERSION=vX.Y.Z   # check a release could be made, changing nothing
```

`make release VERSION=vX.Y.Z` tags `main` and publishes the version, which is
permanent. Only the user runs it, or tells you to.
