# Preserve v3 semantics when corrections change results

OrangeCount will directly correct diagnostics, reload behavior, ordering, and report consistency defects that do not change accounting meaning. Any correction that changes balances, inventories, booking, or query values defaults to Beancount v3 behavior and is introduced first through an explicit compatibility or experimental switch, protecting existing ledgers from silent accounting changes.
