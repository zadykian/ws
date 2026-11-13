package git

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/step"
)

// Notice is a warning that a plan carries after its changes, so that it prints after them: --check
// lists it last, and Apply says it once the changes are made.
type Notice struct {
	Git  *Git
	Text string
}

func (n Notice) String() string { return "warning: " + n.Text }

// Apply warns once in a run.
func (n Notice) Apply(context.Context) error {
	n.Git.say(n.Text)
	return nil
}

// Known is a line that says what Plan found there already, such as a key on GitHub. A plan carries
// it after its changes like a Notice, and only where it holds others: else a run would not be the
// one that changes nothing.
type Known struct {
	Git  *Git
	Text string
}

func (k Known) String() string { return k.Text }

// Apply says the line once in a run.
func (k Known) Apply(context.Context) error {
	k.Git.tell(k.Text)
	return nil
}

// know keeps a line for what Plan found there already.
func (g *Git) know(text string) { g.known = append(g.known, text) }

// note keeps a warning that Plan found.
func (g *Git) note(format string, args ...any) {
	g.notes = append(g.notes, fmt.Sprintf(format, args...))
}

// withNotes ends a plan. Where it holds changes, it carries the warnings after them. Where it holds
// none, nothing follows them, so they print now.
func (g *Git) withNotes(cs []step.Change) []step.Change {
	notes, known := g.notes, g.known
	g.notes, g.known = nil, nil
	if len(cs) == 0 {
		for _, k := range known {
			g.tell(k)
		}
		for _, n := range notes {
			g.say(n)
		}
		return cs
	}
	for _, k := range known {
		cs = append(cs, Known{g, k})
	}
	for _, n := range notes {
		cs = append(cs, Notice{g, n})
	}
	return cs
}

// tell says a line once in a run, as say does for a warning.
func (g *Git) tell(text string) {
	if g.said[text] {
		return
	}
	if g.said == nil {
		g.said = map[string]bool{}
	}
	g.said[text] = true
	g.Out.Say("%s", text)
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
