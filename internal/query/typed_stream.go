// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package query

import "strings"

func referencesBalance(expr Expr) bool {
	switch node := expr.(type) {
	case fieldExpr:
		return strings.EqualFold(node.name, "balance")
	case binaryExpr:
		return referencesBalance(node.left) || referencesBalance(node.right)
	case unaryExpr:
		return referencesBalance(node.inner)
	case parenExpr:
		return referencesBalance(node.inner)
	case functionExpr:
		for _, arg := range node.args {
			if referencesBalance(arg) {
				return true
			}
		}
	}
	return false
}

type runningInventory struct{ accumulator inventoryAccumulator }
type balanceObservation struct {
	stream   *runningInventory
	position positionValue
	value    inventoryValue
	observed bool
}

func (b *balanceObservation) read() inventoryValue {
	if !b.observed {
		b.stream.accumulator.add(b.position)
		b.value = b.stream.accumulator.value()
		b.observed = true
	}
	return b.value
}

// BeanQuery evaluates WHERE and target arguments together in source order.
// Preserve that order before the shared grouping/sorting pipeline: inventory
// observations are immutable, and repeated columns/aggregates cannot add a
// posting twice. Short-circuited predicates still decide whether to observe it.
func prepareBalanceStream(q *Query, rows []Row, targets []typedExpr, budget *executionBudget) ([]Row, error) {
	stream := &runningInventory{}
	selected := make([]Row, 0, len(rows))
	// The running inventory needs WHERE applied during accumulation, so the
	// filter is consumed here. HAVING still observes upstream's leftover
	// scan-loop row (the last row of the unfiltered source), so record it
	// before the filter can discard it. This oracle-pinned contract is held by
	// TestHavingFiltersGroupsAfterAggregation in standard_forms_test.go.
	q.havingContextRow = havingContext(rows)
	for _, row := range rows {
		if err := budget.step("where"); err != nil {
			return nil, err
		}
		position, _ := row["position"].(positionValue)
		row["_balance_observation"] = &balanceObservation{stream: stream, position: position}
		if q.Where != nil {
			value, err := q.Where.eval(row, []Row{row})
			if err != nil {
				return nil, err
			}
			if !truthy(value) {
				continue
			}
		}
		for _, target := range targets {
			if _, err := target.eval(row, []Row{row}); err != nil {
				return nil, err
			}
		}
		selected = append(selected, row)
	}
	q.Where = nil
	return selected, nil
}
