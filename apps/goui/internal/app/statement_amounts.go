// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/widgets"
	"orangecount/internal/report"
)

func (m *Model) statementAmountColumn(
	c *ui.Context,
	amounts map[string]report.PresentedDecimal,
	currencies []string,
	valuation *report.AccountValuation,
) {
	t := c.Theme()
	ui.Column(c).Gap(2).AlignItems(ui.End).Children(func() {
		displayed := false
		for _, currency := range currencies {
			amount, ok := amounts[currency]
			if !ok {
				continue
			}
			widgets.AmountCell(c, amount, currency)
			displayed = true
		}
		if !displayed {
			if valuation != nil && valuation.Completeness == "complete" {
				ui.Text(c, "0").Label("0").TextColor(t.TextMuted)
			} else {
				ui.Text(c, "—").Label("—").TextColor(t.TextMuted)
			}
		}
		m.statementValuationCell(c, valuation)
	})
}

func (m *Model) statementValuationCell(c *ui.Context, valuation *report.AccountValuation) {
	if valuation == nil || valuation.Mode != "converted" || valuation.Completeness == "empty" {
		return
	}
	t := c.Theme()
	var text, label string
	color := t.TextMuted
	separator := "："
	if m.Locale() == "en" {
		separator = ": "
	}
	switch {
	case valuation.CompleteTotal != nil:
		text = m.tr("折算总计") + separator + widgets.AmountText(*valuation.CompleteTotal) + " " + valuation.TargetCurrency
		label = m.tr("折算总计") + separator + widgets.ExactAmountText(*valuation.CompleteTotal, valuation.TargetCurrency)
	case valuation.ValuedPortion != nil:
		text = m.tr("已折算部分") + separator + widgets.AmountText(*valuation.ValuedPortion) + " " + valuation.TargetCurrency
		label = m.tr("已折算部分") + separator + widgets.ExactAmountText(*valuation.ValuedPortion, valuation.TargetCurrency)
		color = t.Warning
	default:
		text = m.tr("折算不可用")
		label = text
		color = t.Warning
	}
	value := ui.Text(c, text).Label(label).TextColor(color).MaxWidth(220)
	tooltip := make([]string, 0, len(valuation.Quotes)+1)
	if valuation.AsOf != "" {
		tooltip = append(tooltip, m.tr("截至日：")+valuation.AsOf)
	}
	for _, quote := range valuation.Quotes {
		tooltip = append(tooltip, quote.Base+"/"+quote.Quote+" "+quote.Rate.Exact+" @ "+quote.Date)
	}
	if len(tooltip) > 0 {
		value.Tooltip(strings.Join(tooltip, "\n"))
	}
	if len(valuation.Missing) == 0 {
		return
	}
	commodities := make([]string, 0, len(valuation.Missing))
	for _, missing := range valuation.Missing {
		commodities = append(commodities, missing.Commodity)
	}
	missing := m.tr("缺报价：") + strings.Join(commodities, ", ")
	ui.Text(c, missing).Label(missing).TextColor(t.Warning).MaxWidth(220)
}
