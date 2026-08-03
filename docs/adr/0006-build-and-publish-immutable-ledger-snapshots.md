# Build and publish immutable ledger snapshots

OrangeCount will parse, validate, and evaluate a changed source ledger into a new immutable snapshot, publishing it only if the build succeeds. The web interface continues to serve the last valid snapshot during a failed reload and presents the new diagnostics separately, preventing incomplete or inconsistent accounting views.
