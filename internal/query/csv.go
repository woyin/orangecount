// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// WriteCSV exports a captured typed result in column order. CSV itself has no
// type system: null and empty strings share an empty field. The companion
// typed analysis document is required for lossless type/context round trips.
func (result TypedResult) WriteCSV(output io.Writer) error {
	writer := csv.NewWriter(output)
	header := make([]string, len(result.Columns))
	for i, column := range result.Columns {
		header[i] = column.Name
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, row := range result.Rows {
		if len(row) != len(header) {
			return fmt.Errorf("query CSV row width does not match columns")
		}
		fields := make([]string, len(row))
		for i, cell := range row {
			var err error
			fields[i], err = csvCell(cell)
			if err != nil {
				return err
			}
		}
		if err := writer.Write(fields); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func csvCell(cell Cell) (string, error) {
	switch cell.Type {
	case "null":
		return "", nil
	case "string", "date", "integer":
		if value, ok := cell.Value.(string); ok {
			return value, nil
		}
	case "bool":
		if value, ok := cell.Value.(bool); ok {
			return strconv.FormatBool(value), nil
		}
	case "decimal":
		if value, ok := cell.Value.(PresentedDecimal); ok {
			return value.Exact, nil
		}
	case "amount", "position", "inventory":
		data, err := json.Marshal(cell.Value)
		return string(data), err
	}
	return "", fmt.Errorf("invalid CSV cell type/value %q", cell.Type)
}
