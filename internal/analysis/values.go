// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package analysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"orangecount/internal/ledger"
	"orangecount/internal/presentation"
	"orangecount/internal/query"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Reject duplicate keys, excessive nesting, trailing JSON and unknown fields.
func StrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("JSON nesting limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			keys := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || keys[name] {
					return fmt.Errorf("duplicate JSON key")
				}
				keys[name] = true
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return requiredAnalysisFields(data, reflect.TypeOf(target).Elem())
}

var analysisInteger = regexp.MustCompile(`^-?[0-9]+$`)

func requiredAnalysisFields(data []byte, typ reflect.Type) error {
	if typ == reflect.TypeOf(json.RawMessage{}) {
		return nil
	}
	if typ.Kind() == reflect.Pointer {
		if bytes.Equal(data, []byte("null")) {
			return nil
		}
		return requiredAnalysisFields(data, typ.Elem())
	}
	if bytes.Equal(data, []byte("null")) {
		return fmt.Errorf("null required value")
	}
	switch typ.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if json.Unmarshal(data, &fields) != nil {
			return fmt.Errorf("invalid object")
		}
		allowed := map[string]bool{}
		for i := 0; i < typ.NumField(); i++ {
			name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
			if name != "" && name != "-" {
				allowed[name] = true
			}
		}
		for name := range fields {
			if !allowed[name] {
				return fmt.Errorf("unknown JSON field")
			}
		}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			raw, ok := fields[name]
			if !ok {
				if strings.Contains(field.Tag.Get("json"), ",omitempty") {
					continue
				}
				if field.Type == reflect.TypeOf(json.RawMessage{}) && name == "value" {
					continue
				}
				return fmt.Errorf("missing required field")
			}
			if err := requiredAnalysisFields(raw, field.Type); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var items []json.RawMessage
		if json.Unmarshal(data, &items) != nil {
			return fmt.Errorf("invalid array")
		}
		for _, item := range items {
			if err := requiredAnalysisFields(item, typ.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

var analysisExact = regexp.MustCompile(`^-?[0-9]+([.][0-9]+|/[0-9]+)?$`)

func ValidValue(value any, kind string) bool {
	decimal := func(d query.PresentedDecimal) bool {
		if len(d.Exact) > 4096 || !analysisExact.MatchString(d.Exact) {
			return false
		}
		value, err := ledger.ParseDecimal(d.Exact)
		if err != nil {
			return false
		}
		expected := presentation.FormatDecimal(value)
		return d == expected
	}
	amount := func(a query.PresentedAmount) bool { return a.Currency != "" && decimal(a.Number) }
	position := func(p query.PresentedPosition) bool {
		if !amount(p.Units) {
			return false
		}
		if p.Cost == nil {
			return true
		}
		c := p.Cost
		if c.Currency == "" || !decimal(c.Number) {
			return false
		}
		if c.Date != nil {
			if _, err := time.Parse("2006-01-02", *c.Date); err != nil {
				return false
			}
		}
		return true
	}
	switch v := value.(type) {
	case string:
		if kind == "integer" {
			return len(v) <= 4096 && analysisInteger.MatchString(v)
		}
		if kind == "date" {
			_, err := time.Parse("2006-01-02", v)
			return err == nil
		}
	case query.PresentedDecimal:
		return decimal(v)
	case query.PresentedAmount:
		return amount(v)
	case query.PresentedPosition:
		return position(v)
	case []query.PresentedPosition:
		if v == nil {
			return false
		}
		for _, p := range v {
			if !position(p) {
				return false
			}
		}
	}
	return true
}
