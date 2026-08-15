// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package source owns source locations and the authoritative include graph.
// It intentionally does not know about ledger syntax or diagnostic rendering.
package source

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// FileID is a stable identifier assigned in include traversal order.
type FileID uint32

// Position is a one-based line and column together with a zero-based byte
// offset. Columns count UTF-8 code points, while offsets remain byte offsets so
// spans can always be used to slice the original source safely.
type Position struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Span is half-open: Start is included and End is excluded. Line and column
// fields are cached to make diagnostics cheap and deterministic.
type Span struct {
	File        FileID `json:"file"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	StartLine   int    `json:"start_line"`
	StartColumn int    `json:"start_column"`
	EndLine     int    `json:"end_line"`
	EndColumn   int    `json:"end_column"`
}

// Valid reports whether the span identifies a real location (the zero span
// is the "unknown" sentinel).
func (s Span) Valid() bool { return s.File != 0 || s.Start != 0 || s.End != 0 }

// Empty reports whether the span covers zero bytes.
func (s Span) Empty() bool { return s.Start == s.End }

// String renders the span for diagnostics, collapsing single-line spans.
func (s Span) String() string {
	if !s.Valid() {
		return "<unknown>"
	}
	if s.StartLine == s.EndLine {
		return fmt.Sprintf("file#%d:%d:%d-%d", s.File, s.StartLine, s.StartColumn, s.EndColumn)
	}
	return fmt.Sprintf("file#%d:%d:%d-%d:%d", s.File, s.StartLine, s.StartColumn, s.EndLine, s.EndColumn)
}

// SourceFile is an immutable UTF-8 source buffer and its line index.
type SourceFile struct {
	ID   FileID
	Path string
	Data []byte

	lineStarts []int
}

// NewSourceFile makes an immutable source file. The input is copied so callers
// can safely reuse their buffer after this function returns.
func NewSourceFile(id FileID, path string, data []byte) *SourceFile {
	b := append([]byte(nil), data...)
	starts := []int{0}
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &SourceFile{ID: id, Path: path, Data: b, lineStarts: starts}
}

func (f *SourceFile) String() string {
	if f == nil {
		return "<nil>"
	}
	return f.Path
}

// Position returns the source position at byte offset. Offsets outside the
// buffer are clamped, and a UTF-8 continuation byte is attributed to the
// containing code point's column.
func (f *SourceFile) Position(offset int) Position {
	if f == nil {
		return Position{}
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(f.Data) {
		offset = len(f.Data)
	}
	i := sort.Search(len(f.lineStarts), func(i int) bool { return f.lineStarts[i] > offset }) - 1
	if i < 0 {
		i = 0
	}
	lineStart := f.lineStarts[i]
	column := 1
	for p := lineStart; p < offset; {
		_, size := utf8.DecodeRune(f.Data[p:])
		if size <= 0 || p+size > offset {
			// The requested offset falls in the middle of a UTF-8 code
			// point; keep it at that code point's column.
			break
		}
		p += size
		column++
	}
	return Position{Offset: offset, Line: i + 1, Column: column}
}

// Span builds a Span from byte offsets, clamping them to the buffer and
// resolving line/column positions for both ends.
func (f *SourceFile) Span(start, end int) Span {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(f.Data) {
		end = len(f.Data)
	}
	if start > len(f.Data) {
		start = len(f.Data)
	}
	sp := f.Position(start)
	ep := f.Position(end)
	return Span{File: f.ID, Start: start, End: end, StartLine: sp.Line, StartColumn: sp.Column, EndLine: ep.Line, EndColumn: ep.Column}
}

// SpanAt is a descriptive alias for Span used by callers that prefer to make
// the offset nature explicit at call sites.
func (f *SourceFile) SpanAt(start, end int) Span { return f.Span(start, end) }

// Text returns the source text covered by span, or "" when the span belongs
// to another file.
func (f *SourceFile) Text(span Span) string {
	if f == nil || span.File != f.ID {
		return ""
	}
	start, end := span.Start, span.End
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(f.Data) {
		end = len(f.Data)
	}
	if start > len(f.Data) {
		start = len(f.Data)
	}
	return string(f.Data[start:end])
}

// LineText returns a line without its line ending. Lines are one-based.
func (f *SourceFile) LineText(line int) string {
	if f == nil || line < 1 || line > len(f.lineStarts) {
		return ""
	}
	start := f.lineStarts[line-1]
	end := len(f.Data)
	if line < len(f.lineStarts) {
		end = f.lineStarts[line] // includes the previous LF
	}
	return strings.TrimSuffix(strings.TrimSuffix(string(f.Data[start:end]), "\n"), "\r")
}

// LineCount returns the number of logical lines. A trailing newline creates a
// final empty logical line, matching the line/column index used by Span.
func (f *SourceFile) LineCount() int {
	if f == nil {
		return 0
	}
	return len(f.lineStarts)
}

// IncludeEdge records a resolved include relation and the source span of the
// include directive's path literal.
type IncludeEdge struct {
	From    FileID
	To      FileID
	Literal string
	Span    Span
}

// GraphIssue is a load-time issue. Parser and diagnostic packages convert it
// to their richer diagnostic type without introducing a package cycle.
type GraphIssue struct {
	Code       string
	Message    string
	Path       string
	SourcePath string
	Span       Span
	Related    []Span
	Fatal      bool
}

// Graph is an immutable snapshot of a source include graph after LoadGraph
// returns. Files are ordered by deterministic depth-first traversal.
type Graph struct {
	Entry       FileID
	Files       map[FileID]*SourceFile
	ByPath      map[string]FileID
	Order       []FileID
	Edges       map[FileID][]IncludeEdge
	Diagnostics []GraphIssue
}

// File returns the source file for id.
func (g *Graph) File(id FileID) *SourceFile {
	if g == nil {
		return nil
	}
	return g.Files[id]
}

// Path returns a normalized path for id, or an empty string if unknown.
func (g *Graph) Path(id FileID) string {
	if f := g.File(id); f != nil {
		return f.Path
	}
	return ""
}

// DisplayPath returns a stable, non-absolute identifier for a graph member.
// Files beneath the entry directory use entry-relative slash-separated paths.
// An include outside that directory receives an opaque include/<id>/name
// identifier so callers never receive a path that could be interpreted as
// traversal. The identifier is for display and graph lookup only; it is never
// passed to filesystem APIs.
func (g *Graph) DisplayPath(id FileID) string {
	file := g.File(id)
	if file == nil {
		return ""
	}
	entry := g.File(g.Entry)
	if entry == nil {
		return SafeDisplayPath(filepath.Base(file.Path))
	}
	relative, err := filepath.Rel(filepath.Dir(entry.Path), file.Path)
	if err != nil || filepath.IsAbs(relative) {
		return opaqueDisplayPath(id, file.Path)
	}
	relative = filepath.Clean(relative)
	if relative == "." {
		return filepath.ToSlash(filepath.Base(file.Path))
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return opaqueDisplayPath(id, file.Path)
	}
	return filepath.ToSlash(relative)
}

// DisplayPaths returns graph member identifiers in include traversal order.
func (g *Graph) DisplayPaths() []string {
	if g == nil {
		return nil
	}
	paths := make([]string, 0, len(g.Order))
	for _, id := range g.Order {
		if path := g.DisplayPath(id); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

// FileIDForDisplayPath resolves only identifiers returned by DisplayPath. It
// does not accept absolute paths or perform any filesystem access.
func (g *Graph) FileIDForDisplayPath(path string) (FileID, bool) {
	if g == nil || strings.TrimSpace(path) == "" {
		return 0, false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return 0, false
	}
	clean = filepath.ToSlash(clean)
	for _, id := range g.Order {
		if g.DisplayPath(id) == clean {
			return id, true
		}
	}
	return 0, false
}

// SafeDisplayPath strips absolute filesystem context from a path that is not
// associated with a graph member. It preserves safe relative components and
// reduces traversal-shaped values to a basename.
func SafeDisplayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." {
		return ""
	}
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(filepath.Base(clean))
	}
	return filepath.ToSlash(clean)
}

func opaqueDisplayPath(id FileID, path string) string {
	return fmt.Sprintf("include/%d/%s", id, filepath.ToSlash(filepath.Base(path)))
}

// LoadGraph reads entry and all recursively included files. Include paths are
// resolved relative to the including file, normalized, and never copied into
// the repository by this package. Missing includes and cycles are accumulated
// as deterministic diagnostics while the rest of the graph continues loading.
func LoadGraph(entry string) (*Graph, error) {
	entry, err := filepath.Abs(entry)
	if err != nil {
		return nil, err
	}
	entry, err = filepath.EvalSymlinks(entry)
	if err != nil {
		return nil, err
	}
	g := &Graph{
		Files:  make(map[FileID]*SourceFile),
		ByPath: make(map[string]FileID),
		Edges:  make(map[FileID][]IncludeEdge),
	}
	state := make(map[string]visitState)
	stack := make([]FileID, 0, 8)
	var visit func(string, FileID, Span) FileID
	visit = func(path string, from FileID, includeSpan Span) FileID {
		path = cleanPath(path)
		if resolved, resolveErr := filepath.EvalSymlinks(path); resolveErr == nil {
			path = cleanPath(resolved)
		}
		if id, ok := g.ByPath[path]; ok {
			if state[path] == visiting {
				g.Diagnostics = append(g.Diagnostics, GraphIssue{Code: "E-INCLUDE-CYCLE", Message: "include cycle detected", Path: path, SourcePath: g.Path(from), Span: includeSpan, Related: []Span{includeSpan}})
			}
			return id
		}
		data, readErr := os.ReadFile(path)
		id := FileID(len(g.Files) + 1)
		g.ByPath[path] = id
		if readErr != nil {
			g.Diagnostics = append(g.Diagnostics, GraphIssue{Code: "E-INCLUDE-READ", Message: "cannot read included file", Path: path, SourcePath: g.Path(from), Span: includeSpan, Fatal: from == 0})
			delete(g.ByPath, path)
			return 0
		}
		file := NewSourceFile(id, path, data)
		g.Files[id] = file
		g.Order = append(g.Order, id)
		state[path] = visiting
		stack = append(stack, id)
		for _, inc := range scanIncludes(file) {
			childPath := inc.path
			if !filepath.IsAbs(childPath) {
				// Beancount resolves relative includes against the including
				// file's directory; absolute paths are used verbatim.
				childPath = filepath.Join(filepath.Dir(path), inc.path)
			}
			child := visit(childPath, id, inc.span)
			if child != 0 {
				g.Edges[id] = append(g.Edges[id], IncludeEdge{From: id, To: child, Literal: inc.path, Span: inc.span})
			}
		}
		stack = stack[:len(stack)-1]
		state[path] = visited
		return id
	}
	g.Entry = visit(entry, 0, Span{})
	if g.Entry == 0 {
		return g, fmt.Errorf("cannot read entry ledger %q", entry)
	}
	return g, nil
}

// Load is the short alias used by callers that do not need to distinguish the
// graph operation from other source loading helpers.
func Load(entry string) (*Graph, error) { return LoadGraph(entry) }

type visitState uint8

const (
	unseen visitState = iota
	visiting
	visited
)

type includeMatch struct {
	path string
	span Span
}

func scanIncludes(f *SourceFile) []includeMatch {
	if f == nil {
		return nil
	}
	var out []includeMatch
	for line := 1; line <= len(f.lineStarts); line++ {
		text := f.LineText(line)
		trimmed := strings.TrimSpace(text)
		if !strings.HasPrefix(trimmed, "include") {
			continue
		}
		if len(trimmed) == len("include") || !isSpace(trimmed[len("include")]) {
			continue
		}
		rest := strings.TrimSpace(trimmed[len("include"):])
		if len(rest) < 2 || rest[0] != '"' {
			continue
		}
		end := closingQuote(rest[1:])
		if end < 0 {
			continue
		}
		literal := rest[1 : 1+end]
		lineStart := f.lineStarts[line-1]
		column := strings.Index(text, "\"")
		if column < 0 {
			column = 0
		}
		start := lineStart + column
		out = append(out, includeMatch{path: unescapeString(literal), span: f.Span(start, start+end+2)})
	}
	return out
}

func cleanPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' }

func closingQuote(s string) int {
	escaped := false
	for i := 0; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		if s[i] == '\\' {
			escaped = true
			continue
		}
		if s[i] == '"' {
			return i
		}
	}
	return -1
}

func unescapeString(s string) string {
	var out bytes.Buffer
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			out.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		default:
			out.WriteByte(s[i])
		}
	}
	return out.String()
}
