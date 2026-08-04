# Use a temporary feature flag for the UI transplant

During migration, OrangeCount will expose the Fava frontend through an explicit local feature flag while retaining the legacy UI as a per-route fallback; a route is wholly one implementation or the other, never a mixed component tree. Both UIs use the same Go atomic-write and snapshot-publish path, and the flag plus legacy UI are removed only after every Fava standard route passes the UX parity gate.
