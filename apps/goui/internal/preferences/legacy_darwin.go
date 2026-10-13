// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

//go:build darwin

package preferences

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

type darwinLegacyDefaults struct{ defaults objc.ID }

var (
	legacyFoundationOnce sync.Once
	legacyFoundationErr  error
)

func OpenLegacyDefaults(domain string) (LegacyDefaults, error) {
	if domain == "" {
		return nil, errors.New("legacy defaults domain is empty")
	}
	if err := loadLegacyFoundation(); err != nil {
		return nil, err
	}
	var defaults objc.ID
	withLegacyAutoreleasePool(func() {
		suiteName := objc.ID(objc.GetClass("NSString")).Send(legacySelector("stringWithUTF8String:"), domain+"\x00")
		defaults = objc.ID(objc.GetClass("NSUserDefaults")).Send(legacySelector("alloc")).Send(
			legacySelector("initWithSuiteName:"), suiteName,
		)
	})
	if defaults == 0 {
		return nil, errors.New("cannot open legacy macOS preferences domain")
	}
	return &darwinLegacyDefaults{defaults: defaults}, nil
}

func loadLegacyFoundation() error {
	legacyFoundationOnce.Do(func() {
		_, legacyFoundationErr = purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	})
	return legacyFoundationErr
}

func (d *darwinLegacyDefaults) Close() {
	if d != nil && d.defaults != 0 {
		d.defaults.Send(legacySelector("release"))
		d.defaults = 0
	}
}

func (d *darwinLegacyDefaults) StringArray(key string) ([]string, bool, error) {
	var paths []string
	var found bool
	withLegacyAutoreleasePool(func() {
		value := d.value(key)
		if !isLegacyClass(value, "NSArray") {
			return
		}
		found = true
		for i, count := uintptr(0), objc.Send[uintptr](value, legacySelector("count")); i < count; i++ {
			item := objc.Send[objc.ID](value, legacySelector("objectAtIndex:"), i)
			if isLegacyClass(item, "NSString") {
				paths = append(paths, legacyString(item))
			}
		}
	})
	return paths, found, nil
}

func (d *darwinLegacyDefaults) String(key string) (string, bool, error) {
	var result string
	var found bool
	withLegacyAutoreleasePool(func() {
		value := d.value(key)
		if isLegacyClass(value, "NSString") {
			result, found = legacyString(value), true
		}
	})
	return result, found, nil
}

func (d *darwinLegacyDefaults) StringDictionary(key string) (map[string]string, bool, error) {
	result := map[string]string{}
	var found bool
	withLegacyAutoreleasePool(func() {
		value := d.value(key)
		if !isLegacyClass(value, "NSDictionary") {
			return
		}
		found = true
		keys := objc.Send[objc.ID](value, legacySelector("allKeys"))
		for i, count := uintptr(0), objc.Send[uintptr](keys, legacySelector("count")); i < count; i++ {
			keyObject := objc.Send[objc.ID](keys, legacySelector("objectAtIndex:"), i)
			valueObject := objc.Send[objc.ID](value, legacySelector("objectForKey:"), keyObject)
			if isLegacyClass(keyObject, "NSString") && isLegacyClass(valueObject, "NSString") {
				result[legacyString(keyObject)] = legacyString(valueObject)
			}
		}
	})
	return result, found, nil
}

func (d *darwinLegacyDefaults) Data(key string) ([]byte, bool, error) {
	var result []byte
	var found bool
	var readErr error
	withLegacyAutoreleasePool(func() {
		value := d.value(key)
		if !isLegacyClass(value, "NSData") {
			return
		}
		found = true
		size := objc.Send[uintptr](value, legacySelector("length"))
		if size > maxPreferencesBytes {
			readErr = ErrInvalid
			return
		}
		if size == 0 {
			result = []byte{}
			return
		}
		bytes := objc.Send[unsafe.Pointer](value, legacySelector("bytes"))
		if bytes == nil {
			readErr = ErrInvalid
			return
		}
		result = append([]byte(nil), unsafe.Slice((*byte)(bytes), int(size))...)
	})
	return result, found, readErr
}

func (d *darwinLegacyDefaults) value(key string) objc.ID {
	if d == nil || d.defaults == 0 {
		return 0
	}
	keyObject := objc.ID(objc.GetClass("NSString")).Send(legacySelector("stringWithUTF8String:"), key+"\x00")
	return objc.Send[objc.ID](d.defaults, legacySelector("objectForKey:"), keyObject)
}

func isLegacyClass(value objc.ID, name string) bool {
	return value != 0 && objc.Send[bool](value, legacySelector("isKindOfClass:"), objc.GetClass(name))
}

func legacyString(value objc.ID) string {
	if value == 0 {
		return ""
	}
	return objc.Send[string](value, legacySelector("UTF8String"))
}

func withLegacyAutoreleasePool(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(legacySelector("alloc")).Send(legacySelector("init"))
	defer pool.Send(legacySelector("drain"))
	fn()
}

func legacySelector(name string) objc.SEL { return objc.RegisterName(name) }
