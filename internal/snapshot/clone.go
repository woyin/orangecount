// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package snapshot

import (
	"math/big"
	"orangecount/internal/ledger"
	"reflect"
)

// ASTs contain interfaces and many exported slices/pointers. Preserve concrete
// directive types while recursively copying those collections. Explicit
// numeric cases avoid reflecting into big.Rat/Decimal's private internals.
// Unexported AST bookkeeping is scalar/read-only and remains copied by value.
func cloneValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	if value.CanInterface() {
		switch n := value.Interface().(type) {
		case ledger.Decimal:
			return reflect.ValueOf(ledger.NewDecimal(n.Rat()))
		case *big.Rat:
			if n == nil {
				return reflect.Zero(value.Type())
			}
			return reflect.ValueOf(new(big.Rat).Set(n))
		}
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		if value.Kind() == reflect.Pointer {
			copy := reflect.New(value.Type().Elem())
			copy.Elem().Set(cloneValue(value.Elem()))
			return copy
		}
		copy := reflect.New(value.Type()).Elem()
		copy.Set(cloneValue(value.Elem()))
		return copy
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			copy.Index(i).Set(cloneValue(value.Index(i)))
		}
		return copy
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			copy.SetMapIndex(iterator.Key(), cloneValue(iterator.Value()))
		}
		return copy
	case reflect.Struct:
		copy := reflect.New(value.Type()).Elem()
		copy.Set(value)
		for i := 0; i < value.NumField(); i++ {
			if copy.Field(i).CanSet() {
				copy.Field(i).Set(cloneValue(value.Field(i)))
			}
		}
		return copy
	default:
		return value
	}
}
