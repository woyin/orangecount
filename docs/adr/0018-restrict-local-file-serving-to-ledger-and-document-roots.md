# Restrict local file serving to ledger and document roots

The local interface may read only source files in the resolved ledger include graph and document attachments contained within explicitly configured document roots. It will normalize and contain-check paths, expose relative paths rather than arbitrary filesystem browsing, and serve attachments safely to prevent traversal even from localhost pages.
