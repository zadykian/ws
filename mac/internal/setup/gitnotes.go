package setup

import (
	"fmt"

	"github.com/zadykian/ws/mac/internal/step"
)

// Notice is a warning that a plan carries after its changes, so that it prints after them: --check
// lists it last, and Apply says it once the changes are made.
type Notice string

func (n Notice) String() string { return "warning: " + string(n) }

// note keeps a warning that Plan found.
func (g *Git) note(format string, args ...any) {
	g.notes = append(g.notes, fmt.Sprintf(format, args...))
}

// withNotes ends a plan. Where it holds changes, it carries the warnings after them. Where it holds
// none, nothing follows them, so they print now.
func (g *Git) withNotes(cs []step.Change) []step.Change {
	notes := g.notes
	g.notes = nil
	if len(cs) == 0 {
		for _, n := range notes {
			g.say(n)
		}
		return cs
	}
	for _, n := range notes {
		cs = append(cs, Notice(n))
	}
	return cs
}

// say warns once for each text in a run, as Apply plans again and the runner plans after it.
func (g *Git) say(text string) {
	if g.said[text] {
		return
	}
	if g.said == nil {
		g.said = map[string]bool{}
	}
	g.said[text] = true
	g.Out.Warn("%s", text)
}
