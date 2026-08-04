# Use a private Fava-shaped frontend adapter

OrangeCount will implement only the Fava request, response, and error shapes consumed by its transplanted frontend, as loopback-only internal Go adapter endpoints backed by the v3 semantic core. Each adapter requires a source-to-domain contract map and tests; no external Fava HTTP API compatibility is promised, which preserves frontend reuse without coupling the product to Python server internals or external clients.
