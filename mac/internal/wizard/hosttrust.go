package wizard

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
)

// hostPrint is the command at ws's console that prints the fingerprint of its ed25519 key.
const hostPrint = "ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub"

// trust stops unless fp, the fingerprint of ws's key, is the one at ws's console. WS_HOST_KEY, else
// the answer at the terminal, must be yes, or hold that fingerprint as the console's ssh-keygen
// prints it. --check asks nothing.
func (h *HostKey) trust(ctx context.Context, w Way, fp string) error {
	o := offer(w, fp)
	if h.Check {
		h.Out.Say("%s: %s; a run would ask whether ws's console shows it", hostStage, o)
		return nil
	}
	answer, set := h.Env("WS_HOST_KEY")
	if !set {
		if !h.Tty {
			return fmt.Errorf("%s: %s, and no terminal is there to compare it with ws's console: set "+
				"WS_HOST_KEY to yes where it shows it, as %s prints it", hostStage, o, hostPrint)
		}
		h.Out.Say("%s: %s. At ws's console, %s prints the one it has", hostStage, o, hostPrint)
		var err error
		if answer, err = h.P.Ask(ctx, effect.Question{Text: "Answer yes where the two match, or paste what it prints"}); err != nil {
			return fmt.Errorf("%s: the terminal ended before the answer", hostStage)
		}
	}
	words := strings.Fields(answer)
	if slices.Contains(words, fp) || (len(words) == 1 && strings.EqualFold(words[0], "yes")) {
		return nil
	}
	return fmt.Errorf("%s: '%s' is neither yes nor %s, the key that ws offers at %s:%s: compare that with "+
		"ws's console, and check ws's address", hostStage, answer, fp, w.Host, w.Port)
}
