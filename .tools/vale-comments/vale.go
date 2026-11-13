package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
)

// An alert is the part of Vale's JSON alert that maps back to the source.
type alert struct {
	Check, Message string
	Line           int
	Span           [2]int
}

// A finding is an alert at its source's file, line and rune column.
type finding struct {
	file, check, message string
	line, col            int
}

// findings maps each alert of Vale's JSON output to its source, in the order of file, line and
// column. mirrors holds each mirror by its path.
func findings(out []byte, mirrors map[string]*mirror) ([]finding, error) {
	alerts := map[string][]alert{}
	if err := json.Unmarshal(out, &alerts); err != nil {
		return nil, fmt.Errorf("vale's output: %w", err)
	}
	var found []finding
	for path, list := range alerts {
		for _, a := range list {
			m := mirrors[path]
			o, ok := m.source(a.Line, a.Span[0])
			if !ok {
				return nil, fmt.Errorf("vale names line %d of %s, which mirrors no comment", a.Line, path)
			}
			found = append(found, finding{m.file, a.Check, a.Message, o.line, o.col})
		}
	}
	slices.SortFunc(found, func(a, b finding) int {
		return cmp.Or(cmp.Compare(a.file, b.file), cmp.Compare(a.line, b.line), cmp.Compare(a.col, b.col),
			cmp.Compare(a.check, b.check))
	})
	return found, nil
}
