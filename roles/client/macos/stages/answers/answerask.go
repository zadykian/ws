package answers

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

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
func (a *asker) ask(ctx context.Context, q conf.Question) (string, error) {
	value, set := a.env(q.Key)
	if set {
		if why := q.Refusal(value, a.lan()); why != "" {
			a.out.Say("%s: %s", stage.Answers, why)
			return "", fmt.Errorf("%s: %s comes from the environment: set it to such a value, or unset it", stage.Answers, q.Key)
		}
	} else {
		if !a.tty {
			return "", fmt.Errorf("%s: no terminal to ask %s: set %s", stage.Answers, q.Key, q.Key)
		}
		if !a.told {
			a.out.Say("%s: values for %s; an empty line takes the one in brackets, and none leaves it empty", stage.Answers, a.conf)
			a.told = true
		}
		var err error
		value, err = a.reply(ctx, q)
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%s: the terminal ended before %s's answer: set %s", stage.Answers, q.Key, q.Key)
		}
		if err != nil {
			return "", err
		}
	}
	a.lines += conf.Line(q.Key, value) + "\n"
	return value, nil
}

// reply asks the maintainer for q's value. A screen draws a key that takes one of a few as a choice
// among them, and a line takes any other, as the plain prompt does every key.
func (a *asker) reply(ctx context.Context, q conf.Question) (string, error) {
	if c, ok := a.p.(effect.Choosing); ok && q.Options() != nil {
		return c.Choose(ctx, effect.ChoiceQuestion{Text: q.Text + " (" + q.Key + ")", Options: q.Options(), Default: q.Def})
	}
	raw, err := a.p.Ask(ctx, effect.Question{Text: q.Prompt(), Valid: func(raw string) error {
		if why := q.Refusal(q.Reply(raw), a.lan()); why != "" {
			return fmt.Errorf("%s: %s: %s", a.prog, stage.Answers, why)
		}
		return nil
	}})
	return q.Reply(raw), err
}
