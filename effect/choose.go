package effect

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// ChoiceQuestion is a question with a few answers to pick from.
type ChoiceQuestion struct {
	Text string
	// Options are the answers, in the order they show. At least one.
	Options []string
	// Default is the option that is chosen first. An empty or unknown one leaves the first.
	Default string
}

// Choosing asks the maintainer to pick one answer. It stands beside Prompt, whose fakes in other
// modules go on compiling: effect is public.
type Choosing interface {
	// Choose returns one of q.Options.
	Choose(ctx context.Context, q ChoiceQuestion) (string, error)
}

// Choose asks q of p. Where p is no Choosing, it asks a line, as Ask does, that must be an option; an
// empty line takes the default.
func Choose(ctx context.Context, p Prompt, q ChoiceQuestion) (string, error) {
	if len(q.Options) == 0 {
		return "", fmt.Errorf("choosing %q: no options", q.Text)
	}
	if c, ok := p.(Choosing); ok {
		return c.Choose(ctx, q)
	}
	first := q.Default
	if !slices.Contains(q.Options, first) {
		first = q.Options[0]
	}
	return p.Ask(ctx, Question{
		Text:    q.Text + " (" + strings.Join(q.Options, ", ") + ")",
		Default: first,
		Valid: func(a string) error {
			if a != "" && !slices.Contains(q.Options, a) {
				return fmt.Errorf("answer one of %s", strings.Join(q.Options, ", "))
			}
			return nil
		},
	})
}
