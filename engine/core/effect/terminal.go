package effect

import (
	"fmt"
	"io"
)

// Terminal is an Out over a writer.
type Terminal struct {
	w      io.Writer
	byHand []string
}

// NewTerminal returns a Terminal that writes to w.
func NewTerminal(w io.Writer) *Terminal {
	return &Terminal{w: w}
}

// Say prints a line.
func (t *Terminal) Say(format string, a ...any) {
	_, _ = fmt.Fprintf(t.w, format+"\n", a...)
}

// Raw prints a line unchanged.
func (t *Terminal) Raw(format string, a ...any) {
	_, _ = fmt.Fprintf(t.w, format+"\n", a...)
}

// Warn prints a line that starts with "warning: ".
func (t *Terminal) Warn(format string, a ...any) {
	_, _ = fmt.Fprintf(t.w, "warning: "+format+"\n", a...)
}

// Diff prints the unified diff of before and after, and nothing where they are equal.
func (t *Terminal) Diff(name string, before, after []byte) {
	_, _ = io.WriteString(t.w, Diff(name, before, after))
}

// ByHand keeps a line for Flush.
func (t *Terminal) ByHand(format string, a ...any) {
	t.byHand = append(t.byHand, fmt.Sprintf(format, a...))
}

// Flush prints the lines that ByHand kept, once, and forgets them, under a blank line and a title.
func (t *Terminal) Flush() {
	if len(t.byHand) == 0 {
		return
	}
	_, _ = io.WriteString(t.w, "\nWhat stays by hand:\n")
	for _, l := range t.byHand {
		_, _ = fmt.Fprintf(t.w, "- %s\n", l)
	}
	t.byHand = nil
}
