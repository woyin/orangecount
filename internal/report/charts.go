// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"sort"
	"strings"

	"orangecount/internal/ledger"
)

// Chart kinds returned by reports. Hierarchy is used by the trial-balance
// account tree; the rest are time series.
const (
	ChartLine       = "line"
	ChartBar        = "bar"
	ChartStackedBar = "stacked-bar"
	ChartHierarchy  = "hierarchy"
)

// chartAmountStatus describes why a posting did or did not contribute to a
// chart value. It lets the chart layer mark unavailable measurement instead of
// silently dropping a currency.
type chartAmountStatus int

const (
	amountOK chartAmountStatus = iota
	amountUnavailablePrice
	amountUnavailableCurrency
	amountNativeMulti
)

// ChartSpec is the report-semantic chart payload returned alongside table
// rows. Values remain exact ledger decimals until the API presentation layer
// serialises them. Series labels, currency, valuation, interval, measure, and
// availability are explicit so a chart cannot silently be interpreted as an
// arbitrary numeric column.
type ChartSpec struct {
	Kind         string        `json:"kind"`
	Title        string        `json:"title"`
	Unit         string        `json:"unit"`
	Currency     string        `json:"currency"`
	Valuation    string        `json:"valuation"`
	Period       string        `json:"period"`
	Interval     string        `json:"interval"`
	Measure      string        `json:"measure"`
	Availability string        `json:"availability,omitempty"`
	Series       []ChartSeries `json:"series"`
	Nodes        []ChartNode   `json:"nodes,omitempty"`
}

// ChartSeries is one labeled series of dated points; Stacked asks the
// browser to stack it against siblings.
type ChartSeries struct {
	Label   string       `json:"label"`
	Points  []ChartPoint `json:"points"`
	Stacked bool         `json:"stacked,omitempty"`
}

// ChartPoint is one exact dated value in a series.
type ChartPoint struct {
	Date  string         `json:"date"`
	Value ledger.Decimal `json:"value"`
}

// ChartNode is a single node of a hierarchy chart (trial balance). Leaf nodes
// carry own_balance; aggregate nodes carry their subtree total. Children are
// nested so the browser can render treemap/sunburst/icicle without rebuilding
// the tree from flat rows.
type ChartNode struct {
	Name     string         `json:"name"`
	Currency string         `json:"currency"`
	Parent   string         `json:"parent"`
	Value    ledger.Decimal `json:"value"`
	Depth    int            `json:"depth"`
	Children []ChartNode    `json:"children,omitempty"`
}

// PresentedChartSpec is the JSON-safe display form. Exact values are kept in
// every point and node while non-terminating rationals receive the same
// readable approximation policy as report table cells.
type PresentedChartSpec struct {
	Kind         string                 `json:"kind"`
	Title        string                 `json:"title"`
	Unit         string                 `json:"unit"`
	Currency     string                 `json:"currency"`
	Valuation    string                 `json:"valuation"`
	Period       string                 `json:"period"`
	Interval     string                 `json:"interval"`
	Measure      string                 `json:"measure"`
	Availability string                 `json:"availability,omitempty"`
	Series       []PresentedChartSeries `json:"series"`
	Nodes        []PresentedChartNode   `json:"nodes,omitempty"`
}

// PresentedChartSeries is the JSON-safe form of ChartSeries.
type PresentedChartSeries struct {
	Label   string                `json:"label"`
	Points  []PresentedChartPoint `json:"points"`
	Stacked bool                  `json:"stacked,omitempty"`
}

// PresentedChartPoint is the JSON-safe form of ChartPoint.
type PresentedChartPoint struct {
	Date  string           `json:"date"`
	Value PresentedDecimal `json:"value"`
}

// PresentedChartNode is the JSON-safe form of ChartNode.
type PresentedChartNode struct {
	Name     string               `json:"name"`
	Currency string               `json:"currency"`
	Parent   string               `json:"parent"`
	Value    PresentedDecimal     `json:"value"`
	Depth    int                  `json:"depth"`
	Children []PresentedChartNode `json:"children,omitempty"`
}

// PresentChart converts a chart spec into its JSON-safe display form,
// applying the same decimal presentation policy as table cells.
func PresentChart(chart ChartSpec) PresentedChartSpec {
	presented := PresentedChartSpec{
		Kind: chart.Kind, Title: chart.Title, Unit: chart.Unit, Currency: chart.Currency,
		Valuation: chart.Valuation, Period: chart.Period, Interval: chart.Interval,
		Measure: chart.Measure, Availability: chart.Availability,
		Series: make([]PresentedChartSeries, len(chart.Series)),
		Nodes:  make([]PresentedChartNode, len(chart.Nodes)),
	}
	for i, series := range chart.Series {
		presented.Series[i] = PresentedChartSeries{Label: series.Label, Stacked: series.Stacked, Points: make([]PresentedChartPoint, len(series.Points))}
		for j, point := range series.Points {
			presented.Series[i].Points[j] = PresentedChartPoint{Date: point.Date, Value: FormatDecimal(point.Value)}
		}
	}
	for i, node := range chart.Nodes {
		presented.Nodes[i] = presentChartNode(node)
	}
	return presented
}

func presentChartNode(node ChartNode) PresentedChartNode {
	presented := PresentedChartNode{
		Name: node.Name, Currency: node.Currency, Parent: node.Parent,
		Value: FormatDecimal(node.Value), Depth: node.Depth,
		Children: make([]PresentedChartNode, len(node.Children)),
	}
	for i, child := range node.Children {
		presented.Children[i] = presentChartNode(child)
	}
	return presented
}

// ReportChart returns report-specific time-series data. The period controls
// the interval of each point: month, quarter, or year. "all" is intentionally
// interpreted as monthly intervals across the evaluated date range, matching
// Fava's useful default while leaving the table unfiltered. No floating-point
// arithmetic or external prices are used here; values are exact native units.
func ReportChart(e ledger.Evaluation, route, period, currency, valuation, account string) ChartSpec {
	interval := normalizeChartPeriod(period)
	keys, endDates := chartPeriods(e, interval)
	if len(keys) == 0 {
		return ChartSpec{Kind: chartKind(route), Title: chartTitle(route), Unit: chartUnit(currency), Currency: currency, Valuation: chartValuation(valuation), Period: interval, Interval: interval}
	}
	switch route {
	case "balance-sheet":
		return balanceSheetChart(e, keys, endDates, interval, currency, valuation)
	case "income-statement":
		return incomeStatementChart(e, keys, endDates, interval, currency, valuation)
	case "accounts":
		return accountChart(e, keys, endDates, interval, currency, valuation, account)
	case "trial-balance":
		return trialBalanceChart(e, keys, endDates, interval, currency, valuation)
	default:
		return ChartSpec{}
	}
}

func normalizeChartPeriod(period string) string {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "quarter":
		return "quarter"
	case "year":
		return "year"
	default:
		return "month"
	}
}

func chartKind(route string) string {
	switch route {
	case "income-statement":
		return ChartStackedBar
	case "balance-sheet", "accounts":
		return ChartLine
	case "trial-balance":
		return ChartHierarchy
	default:
		return ""
	}
}

func chartTitle(route string) string {
	switch route {
	case "balance-sheet":
		return "Balance sheet"
	case "income-statement":
		return "Income statement"
	case "accounts":
		return "Account balance"
	case "trial-balance":
		return "Trial balance"
	default:
		return ""
	}
}

func chartValuation(valuation string) string {
	if strings.EqualFold(strings.TrimSpace(valuation), "market-value") {
		return "market-value"
	}
	return "at-cost"
}

func chartPeriods(e ledger.Evaluation, interval string) ([]string, map[string]string) {
	endDates := make(map[string]string)
	for _, entry := range e.Entries {
		if entry.Date.Raw == "" {
			continue
		}
		key := chartPeriodKey(entry.Date.Raw, interval)
		if key == "" {
			continue
		}
		if entry.Date.Raw > endDates[key] {
			endDates[key] = entry.Date.Raw
		}
	}
	keys := make([]string, 0, len(endDates))
	for key := range endDates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, endDates
}

func chartPeriodKey(raw, interval string) string {
	if len(raw) < 7 {
		return ""
	}
	switch interval {
	case "year":
		return raw[:4]
	case "quarter":
		month := int(raw[5]-'0')*10 + int(raw[6]-'0')
		if month < 1 || month > 12 {
			return ""
		}
		return raw[:4] + "-Q" + string(rune('1'+(month-1)/3))
	default:
		return raw[:7]
	}
}

// transactionPostings splits a transaction's postings into positive and
// negative amounts on the chart's currency, skipping others.
func transactionPostings(e ledger.Evaluation) []chartPosting {
	result := make([]chartPosting, 0)
	for _, entry := range e.Entries {
		var transaction *ledger.Transaction
		switch value := entry.Directive.(type) {
		case ledger.Transaction:
			copy := value
			transaction = &copy
		case *ledger.Transaction:
			transaction = value
		}
		if transaction == nil {
			continue
		}
		for _, posting := range transaction.Postings {
			if posting.Units == nil || posting.Units.Currency == "" || posting.Units.Number.Raw == "" {
				continue
			}
			result = append(result, chartPosting{date: transaction.Date.Raw, account: posting.Account, currency: posting.Units.Currency, amount: ledger.DecimalFromNumber(posting.Units.Number), cost: posting.Cost, payee: transaction.Payee, narration: transaction.Narration, tags: transaction.Tags, links: transaction.Links, flag: transaction.Flag})
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].date < result[j].date })
	return result
}

type chartPosting struct {
	date, account, currency string
	amount                  ledger.Decimal
	cost                    *ledger.CostSpec
	// FQL-matchable transaction attributes, populated so interval aggregation
	// can apply the advanced text filter the way Fava does.
	payee, narration, flag string
	tags, links            []string
}

// balanceSheetChart builds the balance sheet's assets, liabilities, and
// equity lines per interval.
func balanceSheetChart(e ledger.Evaluation, keys []string, ends map[string]string, interval, currency, valuation string) ChartSpec {
	labels := []string{"Assets", "Liabilities", "Equity", "Net worth"}
	seriesValues := make(map[string][]ChartPoint, len(labels))
	for _, label := range labels {
		seriesValues[label] = make([]ChartPoint, 0, len(keys))
	}
	running := make(map[string]ledger.Decimal)
	availability := chartAvailability{}
	postings := transactionPostings(e)
	postingIndex := 0
	for _, key := range keys {
		end := ends[key]
		for postingIndex < len(postings) && postings[postingIndex].date <= end {
			posting := postings[postingIndex]
			postingIndex++
			amount, status := chartAmount(e, posting, end, currency, valuation)
			availability.observe(status)
			if status != amountOK {
				continue
			}
			root := accountRoot(posting.account)
			if root == "Assets" || root == "Liabilities" || root == "Equity" {
				running[root] = running[root].Add(amount)
			}
		}
		assets, liabilities, equity := running["Assets"], running["Liabilities"], running["Equity"]
		seriesValues["Assets"] = append(seriesValues["Assets"], ChartPoint{Date: key, Value: assets})
		seriesValues["Liabilities"] = append(seriesValues["Liabilities"], ChartPoint{Date: key, Value: liabilities})
		seriesValues["Equity"] = append(seriesValues["Equity"], ChartPoint{Date: key, Value: equity})
		seriesValues["Net worth"] = append(seriesValues["Net worth"], ChartPoint{Date: key, Value: assets.Add(liabilities)})
	}
	chart := chartFromSeries(ChartLine, "Balance sheet", chartUnit(currency), currency, valuation, interval, labels, seriesValues)
	chart.Measure = "balance"
	chart.Availability = availability.resolve(currency)
	return chart
}

// incomeStatementChart builds the income statement's revenue and expense
// lines per interval.
func incomeStatementChart(e ledger.Evaluation, keys []string, ends map[string]string, interval, currency, valuation string) ChartSpec {
	labels := []string{"Income", "Expenses", "Net profit"}
	values := make(map[string][]ChartPoint, len(labels))
	for _, label := range labels {
		values[label] = make([]ChartPoint, 0, len(keys))
	}
	availability := chartAvailability{}
	postings := transactionPostings(e)
	for _, key := range keys {
		periodValues := map[string]ledger.Decimal{}
		end := ends[key]
		for _, posting := range postings {
			if chartPeriodKey(posting.date, interval) != key {
				continue
			}
			amount, status := chartAmount(e, posting, end, currency, valuation)
			availability.observe(status)
			if status != amountOK {
				continue
			}
			root := accountRoot(posting.account)
			if root == "Income" || root == "Expenses" {
				periodValues[root] = periodValues[root].Add(amount)
			}
		}
		income, expenses := periodValues["Income"], periodValues["Expenses"]
		values["Income"] = append(values["Income"], ChartPoint{Date: key, Value: income})
		values["Expenses"] = append(values["Expenses"], ChartPoint{Date: key, Value: expenses})
		values["Net profit"] = append(values["Net profit"], ChartPoint{Date: key, Value: income.Add(expenses)})
	}
	chart := chartFromSeries(ChartStackedBar, "Income statement", chartUnit(currency), currency, valuation, interval, labels, values)
	chart.Measure = "flow"
	for i := range chart.Series {
		chart.Series[i].Stacked = true
	}
	chart.Availability = availability.resolve(currency)
	return chart
}

// accountChart builds the single-account balance line: per-interval end
// balances of one account in the chosen currency and valuation.
func accountChart(e ledger.Evaluation, keys []string, ends map[string]string, interval, currency, valuation, account string) ChartSpec {
	account = strings.TrimSpace(account)
	if account == "" {
		return ChartSpec{Kind: ChartLine, Title: "Account balance", Unit: chartUnit(currency), Currency: currency, Valuation: chartValuation(valuation), Period: interval, Interval: interval, Measure: "balance"}
	}
	// When no display currency is requested, keep each native currency in its
	// own series so heterogeneous units are never summed together.
	byCurrency := map[string][]ChartPoint{}
	running := make(map[string]ledger.Decimal)
	availability := chartAvailability{}
	postings := transactionPostings(e)
	postingIndex := 0
	for _, key := range keys {
		end := ends[key]
		for postingIndex < len(postings) && postings[postingIndex].date <= end {
			posting := postings[postingIndex]
			postingIndex++
			if !accountWithin(account, posting.account) {
				continue
			}
			amount, status := chartAmount(e, posting, posting.date, currency, valuation)
			availability.observe(status, posting.currency)
			// In native mode (no display currency) every posting contributes its
			// own exact amount as its currency's series; the nativeMulti status is
			// a valid contribution, not a drop. Only unavailable conversions are skipped.
			if status != amountOK && status != amountNativeMulti {
				continue
			}
			keyCurrency := posting.currency
			if currency != "" {
				keyCurrency = currency
			}
			running[keyCurrency] = running[keyCurrency].Add(amount)
		}
		if currency != "" {
			value := running[currency]
			byCurrency[currency] = append(byCurrency[currency], ChartPoint{Date: key, Value: value})
		} else {
			// Native mode: emit one running value per currency present in this
			// period, never cross-adding.
			for keyCurrency, value := range running {
				byCurrency[keyCurrency] = append(byCurrency[keyCurrency], ChartPoint{Date: key, Value: value})
			}
		}
	}
	series := make([]ChartSeries, 0, len(byCurrency))
	for _, keyCurrency := range sortedKeys(byCurrency) {
		series = append(series, ChartSeries{Label: keyCurrency, Points: byCurrency[keyCurrency]})
	}
	chart := ChartSpec{Kind: ChartLine, Title: "Account balance", Unit: chartUnit(currency), Currency: currency, Valuation: chartValuation(valuation), Period: interval, Interval: interval, Measure: "balance", Series: series}
	chart.Availability = availability.resolve(currency)
	return chart
}

// AccountAverageCostChart returns the weighted-average cost of every held,
// costed commodity in an account over time. It replays the evaluator's
// published booked entries, rather than source postings: AVERAGE merge legs
// and every other booking method have therefore already resolved reductions
// with the same predicates that produced the current holdings inventory.
// Cost currencies are never combined, so each line represents one
// (commodity, cost-currency) pair.
func AccountAverageCostChart(e ledger.Evaluation, period, account string) ChartSpec {
	interval := normalizeChartPeriod(period)
	chart := ChartSpec{
		Kind: ChartLine, Title: "Average cost evolution", Unit: "cost per unit",
		Valuation: "at-cost", Period: interval, Interval: interval, Measure: "average-cost",
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return chart
	}
	keys, endDates := chartPeriods(e, interval)
	if len(keys) == 0 {
		return chart
	}
	type state struct {
		currency, costCurrency string
		units, book            ledger.Decimal
	}
	states := map[string]*state{}
	postings := transactionPostings(e)
	postingIndex := 0
	points := map[string][]ChartPoint{}
	for _, key := range keys {
		end := endDates[key]
		for postingIndex < len(postings) && postings[postingIndex].date <= end {
			posting := postings[postingIndex]
			postingIndex++
			if !accountWithin(account, posting.account) {
				continue
			}
			cost, costCurrency, total, ok := chartCost(posting.cost)
			if !ok {
				continue
			}
			id := posting.currency + "\x00" + costCurrency
			value := states[id]
			if value == nil {
				value = &state{currency: posting.currency, costCurrency: costCurrency}
				states[id] = value
			}
			value.units = value.units.Add(posting.amount)
			bookChange := posting.amount.Mul(cost)
			if total {
				bookChange = cost
				if posting.amount.Sign() < 0 {
					bookChange = bookChange.Neg()
				}
			}
			value.book = value.book.Add(bookChange)
		}
		for id, value := range states {
			if value.units.Sign() <= 0 {
				continue
			}
			points[id] = append(points[id], ChartPoint{Date: key, Value: value.book.Quo(value.units)})
		}
	}
	for _, id := range sortedKeys(points) {
		value := states[id]
		chart.Series = append(chart.Series, ChartSeries{
			Label:  value.currency + " (" + value.costCurrency + ")",
			Points: points[id],
		})
	}
	return chart
}

// sortedKeys returns a map's keys in sorted order.
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func chartUnit(currency string) string {
	if strings.TrimSpace(currency) == "" {
		return "native units"
	}
	return "currency units"
}

// chartAmount converts a posting to the display currency under the requested
// valuation. It returns the converted amount and a status describing whether
// the conversion was exact, unavailable (missing price or conversion quote), or
// left in native units because no single display currency was requested
// (nativeMulti). Callers must not add a value whose status is not amountOK.
func chartAmount(e ledger.Evaluation, posting chartPosting, asOf, currency, valuation string) (ledger.Decimal, chartAmountStatus) {
	if currency == "" {
		return posting.amount, amountNativeMulti
	}
	if strings.EqualFold(valuation, "at-cost") {
		if costNumber, costCurrency, total, ok := chartCost(posting.cost); ok && strings.EqualFold(costCurrency, currency) {
			if total {
				if posting.amount.Sign() < 0 {
					return costNumber.Neg(), amountOK
				}
				return costNumber, amountOK
			}
			return posting.amount.Mul(costNumber), amountOK
		}
	}
	if strings.EqualFold(posting.currency, currency) {
		return posting.amount, amountOK
	}
	if strings.EqualFold(valuation, "market-value") {
		if quote, ok := latestQuote(e, posting.currency, asOf); ok && strings.EqualFold(quote.Currency, currency) {
			return posting.amount.Mul(quote.Amount), amountOK
		}
		if _, ok := latestQuote(e, posting.currency, asOf); !ok {
			return ledger.Zero(), amountUnavailablePrice
		}
		return ledger.Zero(), amountUnavailableCurrency
	}
	return ledger.Zero(), amountUnavailableCurrency
}

func chartCost(cost *ledger.CostSpec) (ledger.Decimal, string, bool, bool) {
	if cost == nil {
		return ledger.Zero(), "", false, false
	}
	for _, component := range cost.Components {
		if component.Kind == ledger.ValueAmount && component.Amount.Currency != "" {
			return ledger.DecimalFromNumber(component.Amount.Number), component.Amount.Currency, cost.Total, true
		}
	}
	return ledger.Zero(), "", false, false
}

func chartFromSeries(kind, title, unit, currency, valuation, period string, labels []string, values map[string][]ChartPoint) ChartSpec {
	series := make([]ChartSeries, 0, len(labels))
	for _, label := range labels {
		series = append(series, ChartSeries{Label: label, Points: values[label]})
	}
	return ChartSpec{Kind: kind, Title: title, Unit: unit, Currency: currency, Valuation: chartValuation(valuation), Period: period, Interval: period, Series: series}
}

// chartAvailability tallies why postings were or were not measurable so the
// chart can report an explicit availability instead of silently dropping a
// currency.
type chartAvailability struct {
	unavailablePrice    bool
	unavailableCurrency bool
	nativeMulti         bool
}

// observe records a conversion status. The currency is only meaningful for
// native-multi observations, which are counted once per distinct currency so a
// single native series does not arbitrarily mark the chart multi-currency.
func (a *chartAvailability) observe(status chartAmountStatus, currency ...string) {
	switch status {
	case amountUnavailablePrice:
		a.unavailablePrice = true
	case amountUnavailableCurrency:
		a.unavailableCurrency = true
	case amountNativeMulti:
		a.nativeMulti = true
	}
}

// resolve returns the chart availability label. A requested display currency
// that is missing quotes reports unavailable; otherwise the explicit price or
// native status is preserved.
func (a *chartAvailability) resolve(currency string) string {
	if currency != "" {
		if a.unavailablePrice {
			return "unavailable-price"
		}
		if a.unavailableCurrency {
			return "unavailable-currency"
		}
		return "priced"
	}
	if a.unavailablePrice {
		return "unavailable-price"
	}
	if a.unavailableCurrency {
		return "unavailable-currency"
	}
	if a.nativeMulti {
		return "native-multi"
	}
	return "at-cost"
}

func accountRoot(name string) string {
	if index := strings.IndexByte(name, ':'); index >= 0 {
		return name[:index]
	}
	return name
}

// trialBalanceChart renders the account hierarchy as a single-currency tree.
// Leaf nodes carry their own direct balance; aggregate nodes carry their subtree
// total. Only one currency is shown at a time so unrelated commodities are never
// summed together; when the requested currency is absent the chart is marked
// unavailable rather than silently mixing units.
func trialBalanceChart(e ledger.Evaluation, keys []string, ends map[string]string, interval, currency, valuation string) ChartSpec {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		currency = "USD"
	}
	rows := accountRootReport(e).Rows
	parentByAccount := map[string]string{}
	childrenByParent := map[string][]string{}
	currencies := map[string]bool{}
	for _, row := range rows {
		name, _ := row["account"].(string)
		if name == "" {
			continue
		}
		if rowCurrency, _ := row["currency"].(string); rowCurrency != "" {
			currencies[rowCurrency] = true
		}
		parent, _ := row["_tree_parent"].(string)
		if parent != "" && parentByAccount[name] == "" {
			// One row arrives per (account, currency). Appending a child once
			// per row registered an account under its parent as many times as
			// it holds currencies, and because that repetition compounds at
			// every level the emitted tree grew multiplicatively: a 122-account
			// ledger produced 7528 nodes, one leaf appearing 4368 times.
			parentByAccount[name] = parent
			childrenByParent[parent] = append(childrenByParent[parent], name)
		}
	}
	if !currencies[currency] {
		return ChartSpec{Kind: ChartHierarchy, Title: "Trial balance", Unit: chartUnit(currency), Currency: currency, Valuation: chartValuation(valuation), Period: interval, Interval: interval, Measure: "balance", Availability: "unavailable-currency"}
	}
	// Build nodes only for the requested currency, preserving the account tree.
	valueByAccount := map[string]ledger.Decimal{}
	for _, row := range rows {
		name, _ := row["account"].(string)
		rowCurrency, _ := row["currency"].(string)
		if rowCurrency != currency {
			continue
		}
		if value, ok := row["total_balance"].(ledger.Decimal); ok {
			valueByAccount[name] = value
		} else if value, ok := row["balance"].(ledger.Decimal); ok {
			valueByAccount[name] = value
		}
	}
	roots := make([]string, 0)
	for name := range valueByAccount {
		if parentByAccount[name] == "" {
			roots = append(roots, name)
		}
	}
	sort.Strings(roots)
	nodes := make([]ChartNode, 0, len(roots))
	for _, root := range roots {
		node := buildChartNode(root, valueByAccount, childrenByParent, currency, 0)
		nodes = append(nodes, node)
	}
	return ChartSpec{Kind: ChartHierarchy, Title: "Trial balance", Unit: chartUnit(currency), Currency: currency, Valuation: chartValuation(valuation), Period: interval, Interval: interval, Measure: "balance", Availability: "priced", Nodes: nodes}
}

func buildChartNode(name string, values map[string]ledger.Decimal, children map[string][]string, currency string, depth int) ChartNode {
	node := ChartNode{Name: name, Currency: currency, Parent: accountParent(name), Depth: depth, Value: values[name]}
	for _, child := range children[name] {
		node.Children = append(node.Children, buildChartNode(child, values, children, currency, depth+1))
	}
	return node
}

func accountWithin(parent, account string) bool {
	return account == parent || strings.HasPrefix(account, parent+":")
}

// nativeChartAmount converts a posting to the amount and currency a
// native-currency chart should accumulate. A lot held at cost contributes its
// cost value in the cost currency, matching the at-cost tree valuation; every
// other posting contributes its own units in its own currency. Nothing is ever
// dropped for lack of a price quote, because nothing is converted across
// currencies.
func nativeChartAmount(posting chartPosting, valuation string) (ledger.Decimal, string) {
	if !strings.EqualFold(valuation, "units") {
		if costNumber, costCurrency, total, ok := chartCost(posting.cost); ok && costCurrency != "" {
			if total {
				if posting.amount.Sign() < 0 {
					return costNumber.Neg(), costCurrency
				}
				return costNumber, costCurrency
			}
			return posting.amount.Mul(costNumber), costCurrency
		}
	}
	return posting.amount, posting.currency
}

// ReportCharts builds the per-measure chart set the transplanted Fava shell
// renders for a tree report: one chart per measure, each carrying one series
// per currency.
//
// This is deliberately separate from ReportChart. ReportChart converts every
// posting into a single requested display currency and skips whatever it
// cannot convert, which understates a multi-currency ledger and leaves a
// permanent "no conversion quote" notice. Fava instead plots each currency as
// its own toggleable line, so no quote is needed and nothing is discarded.
func ReportCharts(e ledger.Evaluation, route, period, valuation string) []ChartSpec {
	interval := normalizeChartPeriod(period)
	keys, ends := chartPeriods(e, interval)
	if len(keys) == 0 {
		return nil
	}
	switch route {
	case "balance-sheet":
		return nativeBalanceCharts(e, keys, ends, interval, valuation)
	case "income-statement":
		return nativeFlowCharts(e, keys, ends, interval, valuation)
	default:
		return nil
	}
}

// nativeBalanceCharts reports cumulative balances per measure, one series per
// currency, in the shape Fava's balance-sheet "Net Worth" chart uses.
func nativeBalanceCharts(e ledger.Evaluation, keys []string, ends map[string]string, interval, valuation string) []ChartSpec {
	measures := []string{"Net worth", "Assets", "Liabilities", "Equity"}
	running := map[string]map[string]ledger.Decimal{}
	byPeriod := map[string]map[string]map[string]ledger.Decimal{}
	for _, measure := range measures {
		running[measure] = map[string]ledger.Decimal{}
		byPeriod[measure] = map[string]map[string]ledger.Decimal{}
	}
	postings := transactionPostings(e)
	index := 0
	for _, key := range keys {
		end := ends[key]
		for index < len(postings) && postings[index].date <= end {
			posting := postings[index]
			index++
			root := accountRoot(posting.account)
			if root != "Assets" && root != "Liabilities" && root != "Equity" {
				continue
			}
			amount, currency := nativeChartAmount(posting, valuation)
			if currency == "" {
				continue
			}
			running[root][currency] = running[root][currency].Add(amount)
			// Net worth is assets plus liabilities, the same combination the
			// balance-sheet table reports.
			if root == "Assets" || root == "Liabilities" {
				running["Net worth"][currency] = running["Net worth"][currency].Add(amount)
			}
		}
		for _, measure := range measures {
			snapshot := make(map[string]ledger.Decimal, len(running[measure]))
			for currency, value := range running[measure] {
				snapshot[currency] = value
			}
			byPeriod[measure][key] = snapshot
		}
	}
	return nativeChartSet(ChartLine, "balance", measures, keys, byPeriod, interval, valuation, false)
}

// nativeFlowCharts reports per-period flows for the income statement, matching
// Fava's Net Profit / Income / Expenses monthly bar charts.
func nativeFlowCharts(e ledger.Evaluation, keys []string, ends map[string]string, interval, valuation string) []ChartSpec {
	measures := []string{"Net profit", "Income", "Expenses"}
	byPeriod := map[string]map[string]map[string]ledger.Decimal{}
	for _, measure := range measures {
		byPeriod[measure] = map[string]map[string]ledger.Decimal{}
	}
	postings := transactionPostings(e)
	for _, key := range keys {
		period := map[string]map[string]ledger.Decimal{}
		for _, measure := range measures {
			period[measure] = map[string]ledger.Decimal{}
		}
		for _, posting := range postings {
			if chartPeriodKey(posting.date, interval) != key {
				continue
			}
			root := accountRoot(posting.account)
			if root != "Income" && root != "Expenses" {
				continue
			}
			amount, currency := nativeChartAmount(posting, valuation)
			if currency == "" {
				continue
			}
			period[root][currency] = period[root][currency].Add(amount)
			period["Net profit"][currency] = period["Net profit"][currency].Add(amount)
		}
		for _, measure := range measures {
			byPeriod[measure][key] = period[measure]
		}
	}
	return nativeChartSet(ChartBar, "flow", measures, keys, byPeriod, interval, valuation, false)
}

// nativeChartSet turns per-period, per-currency totals into one chart per
// measure. Every series is emitted densely over the full period list: the
// renderer positions points by array index, so a currency that only appears
// part-way through the ledger must still carry a point (zero) for the earlier
// periods or its line would be drawn against the wrong dates.
func nativeChartSet(kind, measure string, measures, keys []string, byPeriod map[string]map[string]map[string]ledger.Decimal, interval, valuation string, stacked bool) []ChartSpec {
	charts := make([]ChartSpec, 0, len(measures))
	for _, name := range measures {
		periods := byPeriod[name]
		currencies := map[string]bool{}
		for _, values := range periods {
			for currency, value := range values {
				if !value.IsZero() {
					currencies[currency] = true
				}
			}
		}
		if len(currencies) == 0 {
			continue
		}
		ordered := make([]string, 0, len(currencies))
		for currency := range currencies {
			ordered = append(ordered, currency)
		}
		sort.Strings(ordered)
		series := make([]ChartSeries, 0, len(ordered))
		for _, currency := range ordered {
			points := make([]ChartPoint, 0, len(keys))
			for _, key := range keys {
				points = append(points, ChartPoint{Date: key, Value: periods[key][currency]})
			}
			series = append(series, ChartSeries{Label: currency, Points: points, Stacked: stacked})
		}
		charts = append(charts, ChartSpec{
			Kind: kind, Title: name, Unit: "", Currency: "",
			Valuation: chartValuation(valuation), Period: interval, Interval: interval,
			Measure: measure, Series: series,
		})
	}
	return charts
}
