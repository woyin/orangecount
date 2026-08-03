# Use Beancount v3 as a development oracle

OrangeCount releases will have no Python runtime dependency, but development and CI will run official Beancount v3 through `uv` as a differential-testing oracle. Compatibility fixtures will compare parsing, diagnostics, normalized entries, accounting state, queries, and reports so independent Go execution does not merely accept syntax while changing accounting meaning.
