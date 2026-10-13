// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

// Package widgets contains native Go UI controls shared by OrangeCount pages.
package widgets

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"orangecount/internal/report"
)

// AccountTree projects the core's immutable account report into stable rows
// while keeping parent totals and full account paths intact. Keys returned by
// RootKeys and Children are stable FullName values; callers must treat the
// returned slices and nodes as read-only.
type AccountTree struct {
	allRoots   []report.AccountTreeNode
	rootKeys   []string
	byName     map[string]report.AccountTreeNode
	children   map[string][]string
	currencies []string
}

// NewAccountTree builds a visible tree for search. A matching descendant
// keeps its ancestors visible, and filtering never recomputes report totals.
func NewAccountTree(roots []report.AccountTreeNode, search string) *AccountTree {
	tree := &AccountTree{
		allRoots: roots,
		byName:   make(map[string]report.AccountTreeNode),
		children: make(map[string][]string),
	}
	currencySet := make(map[string]struct{})
	var collectCurrencies func([]report.AccountTreeNode)
	collectCurrencies = func(nodes []report.AccountTreeNode) {
		for _, node := range nodes {
			for currency := range node.SubtreeAmounts {
				currencySet[currency] = struct{}{}
			}
			collectCurrencies(node.Children)
		}
	}
	collectCurrencies(roots)
	for currency := range currencySet {
		tree.currencies = append(tree.currencies, currency)
	}
	sort.Strings(tree.currencies)

	visibleRoots := filterAccountNodes(roots, strings.ToLower(strings.TrimSpace(search)))
	tree.rootKeys = make([]string, 0, len(visibleRoots))
	var index func([]report.AccountTreeNode)
	index = func(nodes []report.AccountTreeNode) {
		for _, node := range nodes {
			tree.byName[node.FullName] = node
			if len(node.Children) > 0 {
				keys := make([]string, 0, len(node.Children))
				for _, child := range node.Children {
					keys = append(keys, child.FullName)
				}
				tree.children[node.FullName] = keys
			}
			index(node.Children)
		}
	}
	for _, root := range visibleRoots {
		tree.rootKeys = append(tree.rootKeys, root.FullName)
	}
	index(visibleRoots)
	return tree
}

// RootKeys returns the visible root accounts in core report order.
func (tree *AccountTree) RootKeys() []string { return tree.rootKeys }

// Children returns the visible direct children of a full account path.
func (tree *AccountTree) Children(fullName string) []string { return tree.children[fullName] }

// Node returns the visible account with this stable full-path key.
func (tree *AccountTree) Node(fullName string) (report.AccountTreeNode, bool) {
	node, ok := tree.byName[fullName]
	return node, ok
}

// Currencies returns all currencies present in the unfiltered core projection,
// so search cannot remove columns that still belong to the report.
func (tree *AccountTree) Currencies() []string { return tree.currencies }

// SetExpanded opens all branches in the source tree or closes every branch.
// Opening uses the full source tree even while the visible projection is
// filtered, so clearing a filter does not reveal a partially expanded report.
func (tree *AccountTree) SetExpanded(state *ui.OutlineState[string], expanded bool) {
	if state == nil {
		return
	}
	if !expanded {
		state.Open.Clear()
		return
	}
	var visit func([]report.AccountTreeNode)
	visit = func(nodes []report.AccountTreeNode) {
		for _, node := range nodes {
			if len(node.Children) == 0 {
				continue
			}
			state.Open.Add(node.FullName)
			visit(node.Children)
		}
	}
	visit(tree.allRoots)
}

// ExpandVisible opens every branch in the filtered tree. This exposes matching
// descendants without changing the caller's saved, unfiltered expansion state.
func (tree *AccountTree) ExpandVisible(state *ui.OutlineState[string]) {
	if state == nil {
		return
	}
	var visit func([]string)
	visit = func(items []string) {
		for _, item := range items {
			children := tree.Children(item)
			if len(children) == 0 {
				continue
			}
			state.Open.Add(item)
			visit(children)
		}
	}
	visit(tree.rootKeys)
}

// PruneExpanded removes keys that are no longer branches in the current core
// projection while preserving expansion for accounts that survived a reload.
func (tree *AccountTree) PruneExpanded(state *ui.OutlineState[string]) {
	if state == nil {
		return
	}
	branches := make(map[string]struct{})
	var visit func([]report.AccountTreeNode)
	visit = func(nodes []report.AccountTreeNode) {
		for _, node := range nodes {
			if len(node.Children) > 0 {
				branches[node.FullName] = struct{}{}
				visit(node.Children)
			}
		}
	}
	visit(tree.allRoots)
	for key := range state.Open.All() {
		if _, ok := branches[key]; !ok {
			state.Open.Remove(key)
		}
	}
}

func filterAccountNodes(roots []report.AccountTreeNode, search string) []report.AccountTreeNode {
	if search == "" {
		return roots
	}
	filtered := make([]report.AccountTreeNode, 0, len(roots))
	var filter func(report.AccountTreeNode) (report.AccountTreeNode, bool)
	filter = func(node report.AccountTreeNode) (report.AccountTreeNode, bool) {
		children := make([]report.AccountTreeNode, 0, len(node.Children))
		for _, child := range node.Children {
			if match, ok := filter(child); ok {
				children = append(children, match)
			}
		}
		matches := strings.Contains(strings.ToLower(node.FullName), search)
		if !matches && len(children) == 0 {
			return report.AccountTreeNode{}, false
		}
		node.Children = children
		return node, true
	}
	for _, root := range roots {
		if match, ok := filter(root); ok {
			filtered = append(filtered, match)
		}
	}
	return filtered
}
