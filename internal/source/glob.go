// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// includeMatcher retains directory evidence and filesystem enumeration order.
// Python glob does not sort matches; ReadDir(-1) on an open file has that order.
type includeMatcher struct {
	ctx       context.Context
	remaining int
	observe   func(string)
	cache     map[string]*regexp.Regexp
}

func (m *includeMatcher) step() error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	m.remaining--
	if m.remaining < 0 {
		return fmt.Errorf("include matching work budget exceeded")
	}
	return nil
}
func (m *includeMatcher) list(path string, dirsOnly bool) ([]string, error) {
	if err := m.step(); err != nil {
		return nil, err
	}
	m.observe(path)
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()
	names := []string{}
	for {
		items, readErr := f.ReadDir(256)
		for _, item := range items {
			if err := m.step(); err != nil {
				return nil, err
			}
			if dirsOnly {
				info, err := os.Stat(filepath.Join(path, item.Name()))
				if err != nil || !info.IsDir() {
					continue
				}
			}
			names = append(names, item.Name())
		}
		if readErr == io.EOF {
			return names, nil
		}
		if readErr != nil {
			return names, nil
		}
	}
}
func magic(s string) bool { return strings.ContainsAny(s, "*?[") }
func (m *includeMatcher) expand(pattern string, dirsOnly bool, depth int) ([]string, error) {
	if err := m.step(); err != nil {
		return nil, err
	}
	if depth > 128 {
		return nil, fmt.Errorf("include matching depth exceeded")
	}
	if !magic(pattern) {
		m.observe(filepath.Dir(pattern))
		if dirsOnly || strings.HasSuffix(pattern, string(filepath.Separator)) {
			info, err := os.Stat(pattern)
			if err == nil && info.IsDir() {
				return []string{pattern}, nil
			}
		} else if _, err := os.Lstat(pattern); err == nil {
			return []string{pattern}, nil
		}
		return nil, nil
	}
	dir, base := filepath.Split(pattern)
	dir = strings.TrimSuffix(dir, string(filepath.Separator))
	if dir == "" {
		dir = string(filepath.Separator)
	}
	directories := []string{dir}
	var err error
	if magic(dir) {
		directories, err = m.expand(dir, true, depth+1)
		if err != nil {
			return nil, err
		}
	}
	results := []string{}
	for _, directory := range directories {
		if base == "**" {
			results = append(results, directory)
			descendants, err := m.recursive(directory, dirsOnly, depth+1)
			if err != nil {
				return nil, err
			}
			results = append(results, descendants...)
		} else if magic(base) {
			names, err := m.list(directory, dirsOnly)
			if err != nil {
				return nil, err
			}
			matcher := m.component(base)
			for _, name := range names {
				if strings.HasPrefix(name, ".") && !strings.HasPrefix(base, ".") {
					continue
				}
				if matcher.MatchString(name) {
					results = append(results, filepath.Join(directory, name))
				}
			}
		} else {
			child := filepath.Join(directory, base)
			matches, err := m.expand(child, dirsOnly, depth+1)
			if err != nil {
				return nil, err
			}
			results = append(results, matches...)
		}
	}
	return results, nil
}
func (m *includeMatcher) recursive(dir string, dirsOnly bool, depth int) ([]string, error) {
	if depth > 128 {
		return nil, fmt.Errorf("recursive include depth exceeded")
	}
	names, err := m.list(dir, dirsOnly)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, name := range names {
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		result = append(result, path)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		next, err := m.recursive(path, dirsOnly, depth+1)
		if err != nil {
			return nil, err
		}
		result = append(result, next...)
	}
	return result, nil
}
func (m *includeMatcher) component(pattern string) *regexp.Regexp {
	if r := m.cache[pattern]; r != nil {
		return r
	}
	var b strings.Builder
	b.WriteString("(?s)^")
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '[':
			j := i + 1
			if j < len(runes) && runes[j] == '!' {
				j++
			}
			if j < len(runes) && runes[j] == ']' {
				j++
			}
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j == len(runes) {
				b.WriteString(`\[`)
				continue
			}
			b.WriteString(classExpression(runes[i+1 : j]))

			i = j
		default:
			b.WriteString(regexp.QuoteMeta(string(runes[i])))
		}
	}
	b.WriteString("$")
	r, err := regexp.Compile(b.String())
	if err != nil {
		r = regexp.MustCompile(`\b\B`)
	}
	m.cache[pattern] = r
	return r
}

func classExpression(chars []rune) string {
	negated := len(chars) > 0 && chars[0] == '!'
	if negated {
		chars = chars[1:]
	}
	var body strings.Builder
	quote := func(ch rune) {
		if strings.ContainsRune(`\^][-`, ch) {
			body.WriteRune('\\')
		}
		body.WriteRune(ch)
	}
	for i := 0; i < len(chars); i++ {
		if i+2 < len(chars) && chars[i+1] == '-' {
			if chars[i] <= chars[i+2] {
				quote(chars[i])
				body.WriteRune('-')
				quote(chars[i+2])
			}
			i += 2
		} else {
			quote(chars[i])
		}
	}
	if body.Len() == 0 {
		if negated {
			return "."
		}
		return `(?:\b\B)`
	}
	if negated {
		return "[^" + body.String() + "]"
	}
	return "[" + body.String() + "]"
}
