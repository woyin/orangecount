// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/i18n"
)

// Locale returns this window's interface and diagnostic locale.
func (m *Model) Locale() string { return i18n.Normalize(m.locale) }

// tr is used only for application-owned strings. Ledger text, query text,
// filenames, amounts, and source locations must remain untouched.
func (m *Model) tr(source string) string { return i18n.Text(m.Locale(), source) }

type localizedSelect struct {
	element  *ui.Element
	value    *string
	options  []string
	labels   []string
	selected *string
}

func (s *localizedSelect) Label(label string) *localizedSelect {
	s.element.Label(label)
	return s
}

func (s *localizedSelect) Changed() bool {
	if !s.element.Changed() {
		return false
	}
	for index, label := range s.labels {
		if label == *s.selected {
			*s.value = s.options[index]
			break
		}
	}
	return true
}

// selectLocalized translates only option labels and maps a selection back to
// the original source value, so application state and request semantics stay
// independent from presentation language.
func (m *Model) selectLocalized(c *ui.Context, value *string, options []string) *localizedSelect {
	translations := make(map[string]string, len(options))
	for _, option := range options {
		translations[option] = m.tr(option)
	}
	return m.selectOptions(c, value, options, translations)
}

// selectOptions translates only explicitly named application options; all
// other option values remain exact user data, such as account and commodity
// names that happen to match an application string.
func (m *Model) selectOptions(c *ui.Context, value *string, options []string, translations map[string]string) *localizedSelect {
	labels := make([]string, len(options))
	for index, option := range options {
		labels[index] = option
		if translated, ok := translations[option]; ok {
			labels[index] = translated
		}
	}
	selected := *value
	for index, option := range options {
		if option == *value {
			selected = labels[index]
			break
		}
	}
	element := ui.Select(c, &selected, labels)
	return &localizedSelect{element: element, value: value, options: options, labels: labels, selected: &selected}
}
