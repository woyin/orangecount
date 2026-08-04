# Render Fava-compatible Journal markup in the Go adapter

OrangeCount will preserve Fava 1.30.12's Journal frontend and private HTML contract by rendering Fava-compatible Journal markup from OrangeCount transaction data in the Go adapter. This deliberately rejects a structured-JSON/Svelte rewrite because minimizing changes to Fava's highest-frequency and most composition-sensitive page provides stronger rendering fidelity; the markup remains a loopback-only presentation contract, uses strict escaping, and does not own ledger semantics.
