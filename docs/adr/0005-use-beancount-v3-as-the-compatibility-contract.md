# Use Beancount v3 as the compatibility contract

OrangeCount will target Beancount v3, the current stable Beancount version, as its language and semantic reference. Valid v2-era syntax that remains part of v3 core syntax will continue to work, while frozen v2-only tools and plugin execution will be surfaced as migration diagnostics instead of silently approximated.
