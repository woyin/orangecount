# Development checks

OrangeCount targets Go 1.22 or newer. The module's `go.mod` is the source of
truth for language/toolchain compatibility; release builds use `-trimpath` and
an empty build ID for reproducible output.

The reproducible local checks are:

```sh
make fmt
make vet
make test
make race
make build
```

`tools/reference/` contains an optional `uv` environment for differential
checks against Beancount v3. It is never a runtime dependency. Set
`ORANGECOUNT_PRIVATE_LEDGER` to an absolute path outside this checkout before
running the harness; it rejects repository paths and reports only redacted
summary counts.
