package link

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// cableAnswer is the answer to a question of the Cable stage: a value, empty for none.
func cableAnswer(raw, def string) string {
	switch {
	case raw == "":
		return def
	case raw == "none":
		return ""
	case len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') && raw[0] == raw[len(raw)-1]:
		return raw[1 : len(raw)-1]
	}
	return raw
}

// ask returns the value of key: the environment's, else the maintainer's. shown names list's values
// as the question shows them, in their order. A value must be one of list, or empty. A bad answer
// at the terminal is asked again, and the environment's stops the run, as the Answers stage's.
func (c *Cable) ask(ctx context.Context, key, text, def string, list, shown []string) (string, error) {
	if value, set := c.Env(key); set {
		if value != "" && !slices.Contains(list, value) {
			return "", fmt.Errorf("%s: %s comes from the environment, and '%s' is none of %s: set it to one, "+
				"or to nothing for none", stage.Cable, key, value, strings.Join(list, ", "))
		}
		return value, nil
	}
	if !c.Tty {
		return "", fmt.Errorf("%s: no terminal to ask %s: set %s, or set it to nothing for none", stage.Cable, key, key)
	}
	value, err := c.reply(ctx, key, text, def, list, shown)
	if errors.Is(err, io.EOF) {
		return "", fmt.Errorf("%s: the terminal ended before %s's answer: set %s", stage.Cable, key, key)
	}
	return value, err
}

// reply asks the maintainer for key's value. A screen draws a choice of shown and none, on def or
// on none. A line names shown, and there an empty line takes def, none no value, and a pair of
// quotes comes off.
func (c *Cable) reply(ctx context.Context, key, text, def string, list, shown []string) (string, error) {
	if ch, ok := c.P.(effect.Choosing); ok {
		first := "none"
		if i := slices.Index(list, def); i >= 0 {
			first = shown[i]
		}
		answer, err := ch.Choose(ctx, effect.ChoiceQuestion{Text: text + " (" + key + ")",
			Options: append(slices.Clone(shown), "none"), Default: first})
		if i := slices.Index(shown, answer); err == nil && i >= 0 {
			return list[i], nil
		}
		return "", err
	}
	hint := "none"
	if def != "" {
		hint = def + ", or none"
	}
	// An empty line takes def in cableAnswer, so the question has no default of its own to print.
	line := text + ", of " + strings.Join(shown, ", ") + " (" + key + ") [" + hint + "]"
	raw, err := c.P.Ask(ctx, effect.Question{Text: line,
		Valid: func(raw string) error {
			if v := cableAnswer(raw, def); v != "" && !slices.Contains(list, v) {
				return fmt.Errorf("%s: %s: '%s' is none of those", c.Prog, stage.Cable, v)
			}
			return nil
		}})
	return cableAnswer(raw, def), err
}

// one returns the only entry of list, or an empty string.
func one(list []string) string {
	if len(list) == 1 {
		return list[0]
	}
	return ""
}
