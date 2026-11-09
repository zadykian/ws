package wizard

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// claudeStage names the stage in its messages.
const claudeStage = "claude on ws"

// WsClaude is the stage that logs WS_USER's claude in on ws, in a session of ssh -t at the
// terminal. It goes through a login shell, whose PATH has ~/.local/bin, where claude is.
type WsClaude struct {
	// Run is ws's run stage, which installs claude on ws.
	Run *WsRun
	// Tty tells whether a terminal is there to log in at.
	Tty bool
	// Here runs the login on ws itself, for ws server wizard: runuser gives WS_USER's login shell in
	// place of ssh's. Nil on the Mac.
	Here effect.Foreground
}

// Name is the stage's name.
func (*WsClaude) Name() string { return claudeStage }

// WaitsOn is ws's run.
func (*WsClaude) WaitsOn() string { return runStage }

type claudeLogin struct{}

func (claudeLogin) String() string {
	return claudeStage + ": would run claude auth login as WS_USER on ws, at the terminal"
}

// Plan is empty where WS_USER's claude on ws has a login.
func (c *WsClaude) Plan(ctx context.Context) (step.Plan, error) {
	s, ok := c.Run.Remote.status(ctx)
	if ok && s.Claude == "ok" {
		return step.Plan{}, nil
	}
	if s.Claude == "none" {
		fix := "run the wizard with --update"
		if c.Here != nil {
			fix = "run sudo ws server apply"
		}
		return step.Plan{}, errors.New(claudeStage + ": ws lacks WS_USER's claude, which its run installs: " + fix)
	}
	return step.Plan{Changes: []step.Change{claudeLogin{}}}, nil
}

// Apply runs claude auth login as WS_USER on ws. Where it fails, the stage says to run the wizard again.
func (c *WsClaude) Apply(ctx context.Context, _ step.Plan) error {
	if !c.Tty {
		return errors.New(claudeStage + ": WS_USER's claude has no login, and no terminal is there for " +
			"claude auth login: run it on ws, then the wizard again")
	}
	if c.Here != nil {
		return c.here(ctx)
	}
	c.Run.Host.Out.Say("%s: claude auth login runs as WS_USER on ws: take its steps at this terminal, "+
		"and open the address it shows in the Mac's browser", claudeStage)
	c.Run.Remote.stale()
	if err := c.Run.Host.SSH.Pass(ctx, "-t", "ws", `exec bash -lc "claude auth login"`); err != nil {
		return errors.New(claudeStage + ": WS_USER's claude on ws still has no login, as above; run the wizard again")
	}
	return nil
}

// here runs claude auth login as WS_USER, as host_vars names it, on ws itself.
func (c *WsClaude) here(ctx context.Context) error {
	want, err := c.Run.Want()
	if err != nil {
		return fmt.Errorf("%s: %w", claudeStage, err)
	}
	c.Run.Out.Say("%s: claude auth login runs as WS_USER: take its steps at this terminal, "+
		"and open the address it shows in a browser", claudeStage)
	c.Run.Remote.stale()
	if err := c.Here.PassForeground(ctx, false, "runuser", "-l", cmp.Or(want.User, c.Run.Default), "-c",
		"claude auth login"); err != nil {
		return errors.New(claudeStage + ": WS_USER's claude still has no login, as above; run the wizard again")
	}
	return nil
}
