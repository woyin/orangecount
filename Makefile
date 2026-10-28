GO ?= go
BIN ?= bin/orangecount
NODE ?= node
NPM ?= npm

.PHONY: all build test race fmt vet license licenses clean app \
	web-test web-typecheck web-check web-build-check web-build-embedded check fixturegen visual-reference \
	check-route-manifest check-provenance check-reference-output favagates paritygates genprediff \
	golden-regen parity-diff parity-triage parity-report

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
##
## The embedded UI lives in internal/web/assets/transplanted and is committed,
## so a clean checkout builds and tests with Go alone. `make web-build-embedded`
## is the only target that rewrites those committed files.

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

# Both Phase 0 static checks together with the parity gates.
check: check-route-manifest check-provenance

# Frontend unit tests (node:test, no browser).
web-test:
	$(NPM) --prefix web test

# Frontend static type check.
web-typecheck:
	$(NPM) --prefix web run typecheck

# Frontend unit + type + Phase 0 checks wrapper.
web-check: web-test web-typecheck check

# Deterministic frontend build check (Node + esbuild).
web-build-check:
	$(NPM) --prefix web run build:check

# Fava cutover gate (tools/favagates): route manifest acceptance, approved
# deviations, absence of the retired migration flag and legacy assets, and the
# visual baseline matrix. The tool must be committed before this target can run
# on a clean checkout; a missing tool is an error, never a silent pass.
# Deliberately not a dependency of `check` yet: the gate still reports real
# failures (non-accepted routes, missing baselines), so `check` would go red.
favagates:
	$(GO) run ./tools/favagates

# Superset gate (tools/paritygates): every upstream inventory item is verified or
# carries an approved exclusion with owner and date. Same commit prerequisite and
# same deliberate exclusion from `check`.
paritygates:
	$(GO) run ./tools/paritygates

# Generative differential (superset gate 3): generate ledgers from a fixed seed,
# evaluate each with the Go engine and the pinned Beancount oracle, and compare
# six dimensions of accounting state. Differences are promoted to paired
# .bean/.json regressions under testdata/fixtures/gen. Requires `uv` and the
# committed tools/reference environment; never part of `make build`.
genprediff:
	$(GO) run ./tools/genprediff

# Rebuild the transplanted shell and copy it into the embedded asset tree
# (internal/web/assets/transplanted). This is the only entry point that updates
# the embedded UI; `make build` and `make test` never need Node because the
# resulting assets are committed.
web-build-embedded:
	$(NPM) --prefix web run build:embedded

# Regenerate the committed dense synthetic reference fixture. This target never
# reads a private ledger and accepts no source-ledger input.
fixturegen:
	$(GO) run ./tools/fixturegen -output testdata/fixtures/fava-reference

# Candidate-only Fava reference capture in the controlled OCI environment.
visual-reference: fixturegen
	$(NPM) --prefix web run visual:reference

## Beancount v3 reference and differential testing (ADR-0008 / Parity Plan)
.PHONY: golden-full-regen parity-full

golden-full-regen:
	uv run --project tools/reference python tools/reference/generate_full_golden.py

.PHONY: query-parity query-oracle-check query-golden-regen
query-parity:
	$(GO) run ./tools/queryparity

query-oracle-check:
	uv run --offline --project tools/reference python tools/reference/generate_query_golden.py --check

query-golden-regen:
	uv run --offline --project tools/reference python tools/reference/generate_query_golden.py

# Deterministic checked-in oracle; no Python or network required in CI.
parity-full:
	$(GO) run ./tools/parity -full
	$(GO) test ./internal/snapshot -run 'TestV3Average|TestV3GlobalBooking' -count=1

golden-regen:
	uv run --project tools/reference python tools/reference/generate_golden.py

parity-diff:
	$(GO) run ./tools/parity \
		-fixtures testdata/fixtures/v3-parity \
		-golden testdata/golden/v3-parity \
		-out testdata/golden/v3-parity/differences.json

parity-triage: parity-diff
	uv run --project tools/reference python tools/reference/jev_triage.py \
		--in testdata/golden/v3-parity/differences.json \
		--out testdata/golden/v3-parity/differences-triaged.json

parity-report:
	./tools/reference/differential-corpus.sh

.PHONY: source-export-oracle
source-export-oracle:
	uv run --offline --project tools/reference python tools/reference/verify_source_export.py
