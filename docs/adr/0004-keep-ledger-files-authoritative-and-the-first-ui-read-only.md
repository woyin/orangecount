# Keep ledger files authoritative and the first UI read-only

OrangeCount's source of truth is the user-maintained `.bean` file set, and the initial web interface will not edit it. This preserves existing editor and version-control workflows while avoiding the difficult compatibility problem of safely rewriting formatted, included ledger files before the evaluator is proven.
