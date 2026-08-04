# Implement FQL as a distinct interface filter language

OrangeCount will implement Fava 1.30.12 FQL as a Go-native parser and evaluator for the Fava standard interface instead of mapping its filter field onto OrangeCount's simpler text filters. FQL remains an interface-scoped filtering contract, distinct from BeanQuery and from ledger semantics, because matching the appearance without matching composed filter behavior would fail workflow parity.
