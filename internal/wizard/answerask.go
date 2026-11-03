package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/zadykian/ws/effect"
)

// answers names the stage in its messages.
const answers = "Answers"

// asker gets the answers of the Answers stage: from the environment, which sets a key even to
// nothing, else from the maintainer.
type asker struct {
	conf string
	prog string
	env  func(string) (string, bool)
	p    effect.Prompt
	// tty tells whether stdin is a terminal: without one, the answers come from the environment alone.
	tty  bool
	out  effect.Out
	told bool
	// lines holds the lines for the answers, to add to mac.conf.
	lines string
	// lan gives WS_LAN's value, which WS_LAN_ADDRESS must lie in.
	lan func() string
}

// ask returns the answer to q, and keeps its line. The environment's bad value stops the run, and a
// bad answer is asked again.
func (a *asker) ask(ctx context.Context, q question) (string, error) {
	value, set := a.env(q.key)
	if set {
		if why := q.refusal(value, a.lan()); why != "" {
			a.out.Say("%s: %s", answers, why)
			return "", fmt.Errorf("%s: %s comes from the environment: set it to such a value, or unset it", answers, q.key)
		}
	} else {
		if !a.tty {
			return "", fmt.Errorf("%s: no terminal to ask %s: set %s", answers, q.key, q.key)
		}
		if !a.told {
			a.out.Say("%s: values for %s; an empty line takes the one in brackets, and none leaves it empty", answers, a.conf)
			a.told = true
		}
		raw, err := a.p.Ask(ctx, effect.Question{Text: q.prompt(), Valid: func(raw string) error {
			if why := q.refusal(q.reply(raw), a.lan()); why != "" {
				return fmt.Errorf("%s: %s: %s", a.prog, answers, why)
			}
			return nil
		}})
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%s: the terminal ended before %s's answer: set %s", answers, q.key, q.key)
		}
		if err != nil {
			return "", err
		}
		value = q.reply(raw)
	}
	a.lines += line(q.key, value) + "\n"
	return value, nil
}
