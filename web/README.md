# OrangeCount web frontend

`src/` contains the existing typed UI source and translation catalogs.
`src/fava/` is the development-only Svelte shell workspace for the selective
Fava frontend transplant. It owns the shell boundary, route/URL state, theme
and locale scaffolding, loading/error states, and the private adapter client
interface. It does not implement report pages yet.

The static shell build is deliberately staged inside `web/` and is not copied
into `internal/web/assets/` during this phase:

```sh
npm --prefix web ci
npm --prefix web run build
npm --prefix web run build:check
npm --prefix web test
```

Outputs are written to `web/staging/fava-shell/`. The build has no runtime CDN
or network dependency. `make build` continues to consume the checked-in Go
assets and does not invoke Node. The staging directory is development output
and is ignored by Git.

The old dependency-free bundle remains untouched until a later route cutover.
P3 will connect the narrow client interface in `src/fava/adapter-client.ts` to
the private loopback adapter contract described in the contract map.
