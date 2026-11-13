package cmd

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
)

// macTarget is ws client macos apply's side of the verbs. only holds the steps that --only names,
// read when the command runs.
func macTarget(p cli.Env, only *[]string) engine.Target {
	return engine.Target{Change: macChange, Prepare: func(_ context.Context, c engine.Call) (engine.Run, error) {
		if c.Selected && len(*only) == 0 {
			return engine.Run{}, engine.UsageError{Err: errors.New("--only names no step")}
		}
		for _, name := range *only {
			if !slices.Contains(macos.Names(), name) {
				return engine.Run{}, engine.UsageError{Err: fmt.Errorf("no step is named %q", name)}
			}
		}
		settings, err := loadMacConf(p)
		if err != nil {
			return engine.Run{}, err
		}
		run := engine.Run{M: c.Machine}
		run.Groups = func(check bool) []tui.Group {
			o := macos.Options{Options: step.Options{Check: check}, Only: *only}
			return []tui.Group{tui.NewGroup("", macOf(p, settings, nil, nil, nil).Rows(o))}
		}
		run.Do = func(ctx context.Context, e engine.Effects, o step.Options) (string, error) {
			_, err := macOf(p, settings, e.M, e.Out, e.Text).Run(ctx, macos.Options{Options: o, Only: *only})
			return "", engine.Cause(err)
		}
		run.Choose = func(choice tui.Choice) (engine.Run, step.Options, error) {
			left := func(_, name string) bool { return !slices.Contains(choice.Only, name) }
			return run, step.Options{Check: choice.Check, LeftOut: left}, nil
		}
		return run, nil
	}}
}
