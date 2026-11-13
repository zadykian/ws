package wizard

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

// ask returns the value of key: the environment's, else a line read at the terminal. There an empty
// line takes def, none no value, and a pair of quotes comes off. A value must be one of list, or
// empty. A bad answer at the terminal is asked again, and the environment's stops the run, as the
// Answers stage's.
func (c *Cable) ask(ctx context.Context, key, text, def string, list []string) (string, error) {
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
	hint := "none"
	if def != "" {
		hint = def + ", or none"
	}
	// An empty line takes def in cableAnswer, so the question has no default of its own to print.
	raw, err := c.P.Ask(ctx, effect.Question{Text: text + " (" + key + ") [" + hint + "]",
		Valid: func(raw string) error {
			if v := cableAnswer(raw, def); v != "" && !slices.Contains(list, v) {
				return fmt.Errorf("%s: %s: '%s' is none of those", c.Prog, stage.Cable, v)
			}
			return nil
		}})
	if errors.Is(err, io.EOF) {
		return "", fmt.Errorf("%s: the terminal ended before %s's answer: set %s", stage.Cable, key, key)
	}
	if err != nil {
		return "", err
	}
	return cableAnswer(raw, def), nil
}

// one returns the only entry of list, or an empty string.
func one(list []string) string {
	if len(list) == 1 {
		return list[0]
	}
	return ""
}
