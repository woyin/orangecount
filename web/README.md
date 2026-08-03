# OrangeCount web UI

`src/` contains the typed UI source and translation catalogs. The browser
bundle under `internal/web/assets/` is checked in so the Go build remains
offline and does not require Node. A frontend toolchain may regenerate
`dist/` and then copy the generated bundle into the embedded asset directory;
the release binary never loads a remote script or stylesheet.
