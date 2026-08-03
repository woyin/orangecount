# Use local redacted structured logs

OrangeCount will write detailed local structured logs for diagnosis, including version, snapshot ID, file counts, durations, error codes, source positions, and stack traces, but will redact account names, amounts, transaction text, metadata values, query text, and absolute paths by default. A clearly marked, explicit, temporary sensitive-diagnostics mode may capture raw data for owner-controlled troubleshooting.
