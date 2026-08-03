# Derive state from ledger files without a persistent database

OrangeCount's first release will evaluate the authoritative `.bean` include graph into in-memory immutable snapshots and will not persist derived state in SQLite or another database. This eliminates cache migration and invalidation risks; a future cache, if measurements require one, must be disposable and fully rebuildable from the source ledger.
