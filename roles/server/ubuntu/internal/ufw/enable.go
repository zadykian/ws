package ufw

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// Enable turns ufw on, as `ufw enable` does, where ConfFile does not say ENABLED=yes or ufw status
// does not say active. It skips
// where systemd is not running, as ufw needs the kernel's netfilter.
type Enable struct {
	M   effect.Machine
	Out effect.Out
}

type enableCall struct{ m effect.Machine }

func (enableCall) String() string { return "ufw enable" }

func (c enableCall) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "ufw", "--force", "enable")
	return err
}

// Plan plans the enable.
func (e Enable) Plan(ctx context.Context) ([]host.Change, error) {
	if !host.Booted(e.M) {
		e.Out.Say("ufw enable: skipped, systemd is not running")
		return nil, nil
	}
	conf, err := readConf(e.M, ConfFile)
	if err != nil {
		return nil, err
	}
	// The conf says what boot loads. `ufw status` says whether the rules are in the kernel now, which
	// a failed ufw.service or a flushed ruleset leaves false.
	if strings.EqualFold(conf["ENABLED"], "yes") && e.active(ctx) {
		return nil, nil
	}
	return []host.Change{enableCall{e.M}}, nil
}

// active is whether ufw reports its rules as loaded. Where ufw fails, it plans the enable.
func (e Enable) active(ctx context.Context) bool {
	out, err := e.M.RunRoot(ctx, "ufw", "status")
	return err == nil && strings.HasPrefix(strings.TrimSpace(string(out)), "Status: active")
}
