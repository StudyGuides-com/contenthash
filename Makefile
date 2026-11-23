.PHONY: help build fmt fmt-check vet tidy test test-verbose test-coverage vectors vectors-check compositions compositions-check check release-check release

MODULE := github.com/studyguides-com/contenthash
export VERSION
VECTORS := testdata/vectors.json
# The vectors are defined on Unicode 15.0, which is Python 3.12's
PYTHON := python3.12 -I

help:
	@echo "Available commands:"
	@echo ""
	@echo "Development:"
	@echo "  make build              - Build the package"
	@echo "  make fmt                - Format Go code"
	@echo "  make tidy               - Tidy go.mod and go.sum"
	@echo ""
	@echo "Testing:"
	@echo "  make test               - Run the tests"
	@echo "  make test-verbose       - Run the tests with verbose output"
	@echo "  make test-coverage      - Run the tests with a coverage summary"
	@echo "  make vet                - Run go vet"
	@echo "  make check              - Everything a release needs: format, vet, vectors, tests"
	@echo ""
	@echo "Vectors:"
	@echo "  make vectors            - Regenerate $(VECTORS) from the spec's Python implementation"
	@echo "  make vectors-check      - Fail if $(VECTORS) differs from what the generator writes"
	@echo "  make compositions       - Regenerate compositions.go from Unicode 15.0 data"
	@echo "  make compositions-check - Fail if compositions.go differs from what its generator writes"
	@echo ""
	@echo "Release (a published version is permanent):"
	@echo "  make release-check VERSION=vX.Y.Z - Check a release could be made, changing nothing"
	@echo "  make release VERSION=vX.Y.Z       - Tag main, push the tag, and have the Go proxy fetch it"

build:
	go build ./...

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "not formatted:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

tidy:
	go mod tidy

test:
	go test ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -cover ./...

vectors:
	$(PYTHON) testdata/gen_vectors.py $(VECTORS)

vectors-check:
	@tmp=$$(mktemp); trap 'rm -f $$tmp' EXIT; \
	$(PYTHON) testdata/gen_vectors.py $$tmp >/dev/null || { echo "the vector generator failed"; exit 1; }; \
	cmp -s $$tmp $(VECTORS) || { echo "$(VECTORS) is out of date: run make vectors"; exit 1; }; \
	echo "$(VECTORS) matches the generator"

compositions:
	$(PYTHON) testdata/gen_compositions.py compositions.go && gofmt -w compositions.go

compositions-check:
	@tmp=$$(mktemp); trap 'rm -f $$tmp' EXIT; \
	$(PYTHON) testdata/gen_compositions.py $$tmp >/dev/null || { echo "the compositions generator failed"; exit 1; }; \
	gofmt $$tmp | cmp -s - compositions.go || { echo "compositions.go is out of date: run make compositions"; exit 1; }; \
	echo "compositions.go matches its generator"

check: fmt-check vet vectors-check compositions-check test

# release-check runs every check a release does, without tagging anything.
release-check: check
	@printf '%s\n' "$$VERSION" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$$' || { echo "VERSION must be vX.Y.Z, e.g. make release VERSION=v0.1.0"; exit 1; }
	@[ "$$(git branch --show-current)" = main ] || { echo "releases are tagged on main; this is $$(git branch --show-current)"; exit 1; }
	@[ -z "$$(git status --porcelain)" ] || { echo "the working tree has uncommitted changes"; exit 1; }
	@git fetch -q origin main && [ "$$(git rev-parse HEAD)" = "$$(git rev-parse origin/main)" ] || { echo "main isn't the same as origin/main; pull or push first"; exit 1; }
	@! git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null || { echo "$(VERSION) is already tagged"; exit 1; }
	@remote=$$(git ls-remote --tags origin 'refs/tags/$(VERSION)') || { echo "couldn't read origin's tags"; exit 1; }; [ -z "$$remote" ] || { echo "$(VERSION) is already tagged on origin"; exit 1; }
	@echo "$(VERSION) can be released from $$(git rev-parse --short HEAD)"

# release tags main and publishes the tag. The Go proxy serves a fetched
# version forever, so it asks for the version to be typed again first.
release: release-check
	@printf 'Publishing $(MODULE)@$(VERSION) is permanent. Type the version to confirm: '; read answer; [ "$$answer" = '$(VERSION)' ] || { echo "not released"; exit 1; }
	git tag -a '$(VERSION)' -m '$(VERSION)'
	git push origin '$(VERSION)' || { git tag -d '$(VERSION)'; echo "push failed; the local tag is removed"; exit 1; }
	GOPROXY=https://proxy.golang.org GOPRIVATE= GONOPROXY= GONOSUMDB= GOFLAGS=-mod=mod go list -m '$(MODULE)@$(VERSION)'
