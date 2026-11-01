package jetbrains

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names. The timer waits on the packages, and the backends on the timer.
const (
	icuStep      = "jetbrains ICU"
	timerStep    = "jetbrains timer"
	backendsStep = "jetbrains backends"
)

// Steps returns the role's steps in order: ICU for Rider's backend, the weekly timer, and the
// backends. The timer comes before the backends, so that a run that fails at one, as on a disk
// short of room, still leaves the timer to try again. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return []step.Step{host.Lazy{Title: timerStep, Parts: func(context.Context) ([]host.Resource, error) {
			return nil, fmt.Errorf("find the ws that runs: %w", err)
		}}}
	}
	site := Site{
		M: m, Out: out, Prune: v.JetBrains.Prune, Avail: StatfsAvail,
		Lock: withLock, Wait: 10 * time.Second,
	}
	owner := host.User{Name: v.User, Home: v.Home}
	site.AsUser = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return owner.Output(ctx, m, name, args...)
	}
	return steps(v, site, effect.OSUnits{}, exe, os.Getenv("WS_JETBRAINS_RUN"), user.Lookup)
}

// steps is Steps with the site, the units, the path of ws, the kind of run and the look-up of the
// user given, for a test to fake. Home and the user's ids are the site's to fill.
func steps(v values.Values, site Site, u effect.Units, exe, run string, lookup func(string) (*user.User, error)) []step.Step {
	var out []step.Step
	timerAfter := ""
	if slices.Contains(v.JetBrains.Backends, "RD") {
		// Rider's backend runs ReSharper's host on .NET, which hangs without ICU. GoLand's needs
		// nothing a server lacks.
		out = append(out, host.Group{Title: icuStep, Parts: []host.Resource{
			host.Packages{M: site.M, Out: site.Out, Names: []string{"libicu78"}},
		}})
		timerAfter = icuStep
	}
	return append(out,
		host.Lazy{Title: timerStep, After: timerAfter, Parts: timer(site.M, u, site.Out, exe)},
		host.Lazy{Title: backendsStep, After: timerStep, Parts: backends(v, site, run, lookup)},
	)
}
