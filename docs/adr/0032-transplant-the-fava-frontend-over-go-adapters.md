# Transplant the Fava frontend over Go adapters

OrangeCount will use Fava 1.30.12's frontend composition, components, styles, and interaction behavior as the main migration path, adapting its data access to OrangeCount's Go-backed internal APIs. The existing hand-built interface is only a temporary per-page fallback and is removed after each page family passes parity; this concentrates effort on preserving the Fava UX rather than repeatedly approximating it through incremental restyling.
