GO ?= go
BIN ?= bin/orangecount
NODE ?= node
NPM ?= npm

.PHONY: all build test race fmt vet license licenses clean \
	web-test web-check web-build-check check fixturegen visual-reference \
	check-route-manifest check-provenance check-reference-output

all: build

## Go runtime targets (no Node/Python/container required)
build:
	@mkdir -p $(dir $(BIN))
	$(GO) build -trimpath -ldflags='-buildid=' -o $(BIN) ./cmd/orangecount

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

fmt:
	@test -z "$$($(GO)fmt -l .)" || (echo "gofmt required"; exit 1)

vet:
	$(GO) vet ./...

license:
	@test -s LICENSE && test -s NOTICE
	@test "$$( $(GO) list -m all | wc -l | tr -d ' ' )" -eq 1
	@echo "license check: no runtime Go dependencies"

licenses: license

clean:
	rm -rf bin dist coverage.out coverage.html

## Frontend/dev-only targets. These require Node and are NOT part of the
## Go runtime path: `make build`/`make test` above never invoke them.
## They are Prerequisite Phase 0 gate checks for the Fava transplant.

# Route/state manifest completeness (Prerequisite Phase 0, deliverable A).
check-route-manifest:
	$(NODE) web/scripts/check-route-manifest.mjs

# Provenance guard v1 over web/ + internal/web/assets managed inventory
# (Prerequisite Phase 0, deliverable B).
check-provenance:
	$(NODE) web/scripts/check-provenance.mjs

# Candidate reference completeness check (requires a prior visual-reference run).
check-reference-output:
	$(NODE) web/scripts/check-reference-output.mjs

# Both Phase 0 static checks together.
check: check-route-manifest check-provenance

# Frontend unit tests (node:test, no browser).
web-test:
	$(NPM) --prefix web test

# Frontend unit + Phase 0 checks wrapper.
web-check: web-test check

# Deterministic frontend build check (Node + esbuild).
web-build-check:
	$(NPM) --prefix web run build:check

# Regenerate the committed dense synthetic reference fixture. This target never
# reads a private ledger and accepts no source-ledger input.
fixturegen:
	$(GO) run ./tools/fixturegen -output testdata/fixtures/fava-reference

# Candidate-only Fava reference capture in the controlled OCI environment.
visual-reference: fixturegen
	$(NPM) --prefix web run visual:reference
