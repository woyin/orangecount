# Embed the TypeScript web interface in the Go binary

OrangeCount will use a Go backend and TypeScript web interface whose built static assets are embedded at compile time. Development may use frontend tooling, but the released CLI requires neither Node nor external CDNs and exposes only an internal, versioned localhost HTTP API rather than a public API-compatibility commitment.
