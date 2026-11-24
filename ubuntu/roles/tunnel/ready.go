package tunnel

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// waitForKey waits for wg0 to hold the server's key, and says its public key, which the Mac's end
// of the tunnel needs.
type waitForKey struct {
	w      WG0
	public string
}

func (c waitForKey) String() string { return "wait for wg0 to hold the server's key" }

// Apply waits, and says the public key once wg0 holds it, as a run that made the key showed none.
func (c waitForKey) Apply(ctx context.Context) error {
	sleep := c.w.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	for i := range c.w.Tries {
		if i > 0 {
			sleep(c.w.Pause)
		}
		if c.holds(ctx) {
			c.w.Out.Say("wg0 holds ws's WireGuard public key, for WS_PUBLIC_KEY in a server's values.yml: %s", c.public)
			return nil
		}
	}
	return fmt.Errorf("wg0 holds another key than %s, whose public key is %s", keyFile, c.public)
}

// holds reports whether wg0 exists and holds the public key.
func (c waitForKey) holds(ctx context.Context) bool {
	out, err := c.w.M.RunRoot(ctx, "wg", "show", "wg0", "public-key")
	return err == nil && strings.TrimSpace(string(out)) == c.public
}
