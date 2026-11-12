package ikeward

import (
	"context"
	"fmt"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

// Step names. The install waits on the binary's download, and the wait on the install.
const (
	downloadStep = "ikeward download"
	forwardStep  = "ikeward forwarding"
	installStep  = "ikeward install"
	statusStep   = "ikeward status"
)

// Steps returns the role's steps in order. The check of the values comes first, then the binary's
// download. The forwarding for the WireGuard peers follows, so that a peer taken off the list
// loses its way in even while the VPN is down. Then come the install from the profile of the
// secrets, and the wait for the VPN. Without ikeward_profile it returns none, as in CI's
// container: the VPN, the file and ufw's rules stay unchanged.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	return steps(v.Ikeward, s, m, out, statusTries, statusWait)
}

// steps is Steps with the wait's bounds given, for a test to shorten.
func steps(v values.Ikeward, s secrets.Secrets, m effect.Machine, out effect.Out, tries int,
	wait time.Duration) []step.Step {
	if v.Profile == "" {
		return nil
	}
	file := binaryFile(m)
	// A profile that the secrets lack fails the run before anything changes, as the values do. A
	// read holds no secrets, so it says nothing of the profile.
	ready := func() error {
		if err := check(v); err != nil {
			return err
		}
		if !s.Unread && !(profile{files: s.File}).present() {
			return fmt.Errorf("ikeward_profile is %s, and the secrets hold no profile for it", v.Profile)
		}
		return nil
	}
	return []step.Step{
		host.Lazy{Title: downloadStep, Parts: func(context.Context) ([]host.Resource, error) {
			if err := ready(); err != nil {
				return nil, err
			}
			return []host.Resource{file}, nil
		}},
		// The forwarding needs not the binary, so a check shows its plan while the download is due.
		host.Lazy{Title: forwardStep, Parts: func(context.Context) ([]host.Resource, error) {
			if err := ready(); err != nil {
				return nil, err
			}
			return forward(m, out, v.ForwardFrom), nil
		}},
		host.Group{Title: installStep, After: downloadStep, Parts: []host.Resource{
			Install{M: m, Out: out, Values: v, Secrets: s},
		}},
		host.Group{Title: statusStep, After: installStep, Parts: []host.Resource{
			Status{M: m, Out: out, Tries: tries, Wait: wait},
		}},
	}
}
