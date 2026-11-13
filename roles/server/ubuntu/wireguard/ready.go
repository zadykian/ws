package wireguard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// ready shows the server's public key, which the Mac's end of the tunnel needs, and checks that
// wg0 holds it. networkd creates wg0 a moment after the reload, so the check waits for it. It
// checks nothing without systemd booted, as in CI's container. A dry run that finds no key says
// that a real run makes it.
type ready struct {
	m     effect.Machine
	out   effect.Out
	tries int
	pause time.Duration
	sleep func(time.Duration)
}

type waitForKey struct {
	r      ready
	public string
}

func (c waitForKey) String() string { return "wait for wg0 to hold the server's key" }

func (c waitForKey) Apply(ctx context.Context) error {
	for i := range c.r.tries {
		if i > 0 {
			c.r.sleep(c.r.pause)
		}
		if c.r.holds(ctx, c.public) {
			return nil
		}
	}
	return fmt.Errorf("wg0 holds another key than %s, whose public key is %s", keyFile, c.public)
}

// holds reports whether wg0 exists and holds the public key.
func (r ready) holds(ctx context.Context, public string) bool {
	out, err := r.m.RunRoot(ctx, "wg", "show", "wg0", "public-key")
	return err == nil && strings.TrimSpace(string(out)) == public
}

// Plan says the public key, and plans the wait where wg0 doesn't hold it yet.
func (r ready) Plan(ctx context.Context) ([]resource.Change, error) {
	public, hasKey, err := serverPublicKey(r.m)
	if err != nil {
		return nil, err
	}
	if !hasKey {
		r.out.Say("The server has no WireGuard key yet: a real run makes it and shows its public key.")
		return nil, nil
	}
	effect.SayLasting(r.out, "ws's WireGuard public key, for WS_PUBLIC_KEY in the Mac's server.conf: %s", public)
	if !systemd.Booted(r.m) || r.holds(ctx, public) {
		return nil, nil
	}
	return []resource.Change{waitForKey{r, public}}, nil
}
