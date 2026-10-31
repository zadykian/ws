// Package effecttest holds the fakes of package effect, for the tests of steps.
package effecttest

import (
	"context"
	"fmt"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
)

// Prompt answers from a script, and fails the test on a question it was not given.
type Prompt struct {
	T       testing.TB
	Answers map[string]string // by the text of the question
	Asked   []string
}

// Ask returns the scripted answer to q.Text, and asks again of nothing: a bad answer fails.
func (p *Prompt) Ask(_ context.Context, q effect.Question) (string, error) {
	p.Asked = append(p.Asked, q.Text)
	answer, ok := p.Answers[q.Text]
	if !ok {
		p.T.Fatalf("unexpected question %q", q.Text)
	}
	if answer == "" {
		answer = q.Default
	}
	if q.Valid != nil {
		if err := q.Valid(answer); err != nil {
			p.T.Fatalf("scripted answer %q to %q is invalid: %v", answer, q.Text, err)
		}
	}
	return answer, nil
}

// Out records what is said, one entry per call, as "say: ...", "warn: ...", "diff: NAME" or
// "by hand: ...".
type Out struct {
	Lines []string
}

// Say records a line.
func (o *Out) Say(format string, a ...any) { o.add("say: ", format, a) }

// Warn records a warning.
func (o *Out) Warn(format string, a ...any) { o.add("warn: ", format, a) }

// Diff records the name of the diff and its text.
func (o *Out) Diff(name string, before, after []byte) {
	o.Lines = append(o.Lines, "diff: "+name+"\n"+effect.Diff(name, before, after))
}

// ByHand records a line to do by hand.
func (o *Out) ByHand(format string, a ...any) { o.add("by hand: ", format, a) }

func (o *Out) add(prefix, format string, a []any) {
	o.Lines = append(o.Lines, prefix+fmt.Sprintf(format, a...))
}
