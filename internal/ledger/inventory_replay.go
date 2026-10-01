// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "sort"

// BookedInventoryChange points into the immutable evaluated entry, preserving
// split postings' shared source identity. Consumed counts only pre-existing
// positive units of the exact lot; opening a NONE short is not realized profit.
type BookedInventoryChange struct {
	EntryIndex, PostingIndex int
	Account                  string
	Position                 Position
	Consumed                 Decimal
	ConsumedLots             []Position
}

type InventoryReplay struct {
	Accounts map[string][]Position
	Changes  []BookedInventoryChange
}

// The already-booked inventory has at most one live position per exact key.
// Keep insertion slots for provenance/order and delete zero keys so a later
// reacquisition establishes a new position at its own source span.
type replayInventory struct {
	positions []Position
	indices   map[replayLotKey]int
}

type replayLotKey struct {
	commodity, costNumber, costCurrency, label string
	costPresent, datePresent, labelPresent     bool
	date                                       int
}

func inventoryReplayKey(position Position) replayLotKey {
	key := replayLotKey{commodity: position.Currency}
	if cost := position.Cost; cost != nil {
		key.costPresent = true
		key.costNumber, key.costCurrency = cost.Number.String(), cost.Currency
		key.label, key.labelPresent = cost.Label, cost.LabelPresent
		if cost.Date != nil {
			key.datePresent, key.date = true, dateKey(*cost.Date)
		}
	}
	return key
}

// ReplayBookedInventory folds signed, already-booked postings through cutoff.
// It never books again, consults today's positions, or changes the evaluation.
// Plain units remain separate from zero-cost lots. M legs are folded normally;
// consumers identify economic reductions from the original posting flags.
func ReplayBookedInventory(e Evaluation, cutoff string) InventoryReplay {
	result := InventoryReplay{Accounts: map[string][]Position{}, Changes: []BookedInventoryChange{}}
	inventories := map[string]*replayInventory{}
	indices := make([]int, len(e.Entries))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool { return e.Entries[indices[i]].Date.Raw < e.Entries[indices[j]].Date.Raw })
	for _, entryIndex := range indices {
		entry := e.Entries[entryIndex]
		if cutoff != "" && entry.Date.Raw > cutoff {
			break
		}
		var tx *Transaction
		switch value := entry.Directive.(type) {
		case *Transaction:
			tx = value
		case Transaction:
			tx = &value
		}
		if tx == nil {
			continue
		}
		for postingIndex, posting := range tx.Postings {
			if posting.Units == nil || posting.Units.Number.Raw == "" || posting.Units.Currency == "" {
				continue
			}
			position := Position{Units: DecimalFromNumber(posting.Units.Number), Currency: posting.Units.Currency, Span: posting.Span()}
			if posting.Cost != nil {
				if cost, ok := normalizeCost(*posting.Cost); ok {
					position.Cost = &cost
				}
			}
			inventory := inventories[posting.Account]
			if inventory == nil {
				inventory = &replayInventory{indices: map[replayLotKey]int{}}
				inventories[posting.Account] = inventory
			}
			key := inventoryReplayKey(position)
			oldIndex, exists := inventory.indices[key]
			if position.Cost != nil || posting.Price != nil {
				consumed := Zero()
				consumedLots := []Position{}
				remaining := position.Units.Neg()
				if position.Units.Sign() < 0 && exists {
					old := inventory.positions[oldIndex]
					if old.Units.Sign() > 0 {
						amount := old.Units
						if amount.Cmp(remaining) > 0 {
							amount = remaining
						}
						old.Units = amount
						consumedLots = append(consumedLots, old)
						consumed = consumed.Add(amount)
						remaining = remaining.Sub(amount)
					}
					if consumed.Cmp(position.Units.Neg()) > 0 {
						consumed = position.Units.Neg()
					}
				}
				result.Changes = append(result.Changes, BookedInventoryChange{EntryIndex: entryIndex, PostingIndex: postingIndex, Account: posting.Account, Position: position, Consumed: consumed, ConsumedLots: consumedLots})
			}
			if exists {
				inventory.positions[oldIndex].Units = inventory.positions[oldIndex].Units.Add(position.Units)
				if inventory.positions[oldIndex].Units.IsZero() {
					delete(inventory.indices, key)
				}
			} else if !position.Units.IsZero() {
				inventory.indices[key] = len(inventory.positions)
				inventory.positions = append(inventory.positions, position)
			}
		}
	}
	for account, inventory := range inventories {
		positions := make([]Position, 0, len(inventory.indices))
		for _, position := range inventory.positions {
			if !position.Units.IsZero() {
				positions = append(positions, position)
			}
		}
		result.Accounts[account] = positions
	}
	return result
}
