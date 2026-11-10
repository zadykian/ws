package wizard

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Hand is what stays by hand after a run in which a stage acted, a line each. A stage adds a line
// as it runs, or in a function that Late registers.
type Hand struct {
	lines []string
	late  []func(context.Context, *Hand)
}

// Add adds a line.
func (h *Hand) Add(format string, a ...any) {
	h.lines = append(h.lines, fmt.Sprintf(format, a...))
}

// Late registers a function that adds its lines when the run closes, after the stages.
func (h *Hand) Late(f func(context.Context, *Hand)) { h.late = append(h.late, f) }

// Close runs the late functions and prints the list, or nothing where the list is empty. The caller
// closes only after a run in which a stage acted.
func (h *Hand) Close(ctx context.Context, w io.Writer) {
	for _, f := range h.late {
		f(ctx, h)
	}
	if len(h.lines) == 0 {
		return
	}
	_, _ = io.WriteString(w, "\nWhat stays by hand:\n- "+strings.Join(h.lines, "\n- ")+"\n")
}
