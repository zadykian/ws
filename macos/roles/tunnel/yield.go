package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/launchd"
	"github.com/zadykian/ws/macos/roles/tunnel/plan"
)

// YieldWait bounds the wait for the daemons to take a yield, and yieldPoll is the time between
// the reads of their states.
const (
	YieldWait = 5 * time.Second
	yieldPoll = 100 * time.Millisecond
)

// Yielder hands networks to another owner, or takes them back, and waits for the daemons. A caller
// that goes on at once races them: the other owner's floor meets a route that still stands.
type Yielder struct {
	M effect.Machine
	// Sleep waits between the reads of the states. Nil means time.Sleep.
	Sleep func(time.Duration)
}

// Yield writes nets, IPv4 networks each once, to YieldFile whole, or removes it where nets is
// empty, and signals each daemon that launchd has loaded. It returns once each state shows the yield, and fails after YieldWait
// on one that does not, or on a failed signal, naming the daemon.
func (y Yielder) Yield(ctx context.Context, nets []netip.Prefix) error {
	var want []netip.Prefix
	var text strings.Builder
	for _, n := range nets {
		if !slices.Contains(want, n) {
			want = append(want, n)
			fmt.Fprintln(&text, n)
		}
	}
	if err := y.write(ctx, text.String()); err != nil {
		return err
	}
	names, err := Installed(y.M)
	if err != nil {
		return err
	}
	var errs []error
	var signalled []Daemon
	l := launchd.Launchctl{M: y.M}
	for _, name := range names {
		d := For(name)
		if !loaded(ctx, y.M, d.Label) {
			continue // it reads the file as it starts
		}
		if err := l.Signal(ctx, d.Label, "SIGHUP"); err != nil {
			errs = append(errs, fmt.Errorf("cannot signal system/%s, which has not read the yield", d.Label))
			continue
		}
		signalled = append(signalled, d)
	}
	return errors.Join(append(errs, y.wait(ctx, signalled, want)...)...)
}

// write replaces the yield, root's and readable by all, or removes it for none.
func (y Yielder) write(ctx context.Context, text string) error {
	if text == "" {
		if _, err := y.M.RunRoot(ctx, "rm", "-f", YieldFile); err != nil {
			return fmt.Errorf("cannot remove %s: %w", YieldFile, err)
		}
		return nil
	}
	if err := y.M.WriteRootFile(YieldFile, []byte(text), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", YieldFile, err)
	}
	return nil
}

// wait reads the states of pending until each shows want, and names each that does not.
func (y Yielder) wait(ctx context.Context, pending []Daemon, want []netip.Prefix) []error {
	for try := 0; ; try++ {
		pending = slices.DeleteFunc(pending, func(d Daemon) bool { return y.took(d, want) })
		if len(pending) == 0 || try == int(YieldWait/yieldPoll) || ctx.Err() != nil {
			break
		}
		if y.Sleep == nil {
			time.Sleep(yieldPoll)
		} else {
			y.Sleep(yieldPoll)
		}
	}
	var errs []error
	for _, d := range pending {
		errs = append(errs, fmt.Errorf("the daemon of %s has not taken the yield after %v; its log is %s",
			d.Name, YieldWait, d.Log))
	}
	return errs
}

// took is whether d's state shows want, which the daemon writes once its routes took the yield.
func (y Yielder) took(d Daemon, want []netip.Prefix) bool {
	text, err := y.M.ReadFile(d.State)
	if err != nil {
		return false
	}
	s, err := plan.ParseState(string(text))
	return err == nil && slices.Equal(s.Yielded, want)
}
