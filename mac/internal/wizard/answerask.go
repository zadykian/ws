package wizard

import (
	"fmt"

	"github.com/zadykian/ws/mac/internal/effect"
)

// answers names the stage in its messages.
const answers = "Answers"

// asker gets the answers of the Answers stage: from the environment, which sets a key even to
// nothing.
type asker struct {
	env func(string) (string, bool)
	out effect.Out
	// lines holds the lines for the answers, to add to mac.conf.
	lines string
	// lan gives WS_LAN's value, which WS_LAN_ADDRESS must lie in.
	lan func() string
}

// ask returns the answer to q, and keeps its line. The environment's bad value stops the run, and a
// bad answer is asked again.
func (a *asker) ask(q question) (string, error) {
	value, set := a.env(q.key)
	if !set {
		return "", fmt.Errorf("%s: %s is not set in the environment", answers, q.key)
	}
	if why := q.refusal(value, a.lan()); why != "" {
		a.out.Say("%s: %s", answers, why)
		return "", fmt.Errorf("%s: %s comes from the environment: set it to such a value, or unset it", answers, q.key)
	}
	a.lines += line(q.key, value) + "\n"
	return value, nil
}
