GO ?= go
BIN ?= bin/orangecount

.PHONY: all build test race fmt vet license licenses clean

all: build

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
