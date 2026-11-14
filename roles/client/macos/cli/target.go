package cli

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// macTarget is ws client macos apply's side of the verbs. only holds the steps that --only names,
// read when the command runs.
// The Mac's roles take their machine and out from Mac, which Run fills from the run's effects.
// They leave the machine, units and out that Steps gets unused.
func macTarget(p enginecli.Env, only *[]string, server *string) engine.Target {
	return engine.Target{Wizard: macSide(p, server), Change: macChange, State: macReport(p, server), Refuse: func(engine.Call) string { return wizardRefusal(p) },
		NoScreen: "it needs a terminal and no --plain: anywhere else, edit mac.conf by hand",
		Sheets: func(ctx context.Context, c engine.Call) ([]tui.Sheet, func(context.Context), error) {
			return macForms(c.In, c.Err, p, server)(ctx)
		}, Prepare: func(_ context.Context, c engine.Call) (engine.Run, error) {
			if c.Selected && len(*only) == 0 {
				return engine.Run{}, engine.UsageError{Err: errors.New("--only names no step")}
			}
			for _, name := range *only {
				if !slices.Contains(macos.Names(), name) {
					return engine.Run{}, engine.UsageError{Err: fmt.Errorf("no step is named %q", name)}
				}
			}
			name, settings, err := loadMacConf(p, server)
			if err != nil {
				return engine.Run{}, err
			}
			if err := refuseBadSibling(p.Machine, conf.Path(p.Getenv, p.Pwd), name); err != nil {
				return engine.Run{}, err
			}
			run := engine.Run{M: c.Machine}
			run.Groups = func(check bool) []tui.Group {
				o := macos.Options{Options: step.Options{Check: check}, Only: *only}
				return []tui.Group{tui.NewGroup("", macOf(p, name, settings, nil, nil, nil).Rows(o))}
			}
			run.Do = func(ctx context.Context, e engine.Effects, o step.Options) (string, error) {
				_, err := macOf(p, name, settings, e.M, e.Out, e.Text).Run(ctx, macos.Options{Options: o, Only: *only})
				return "", engine.Cause(err)
			}
			run.Choose = func(choice tui.Choice) (engine.Run, step.Options, error) {
				left := func(_, name string) bool { return !slices.Contains(choice.Only, name) }
				return run, step.Options{Check: choice.Check, LeftOut: left}, nil
			}
			return run, nil
		}}
}

// macReport readies what ws client macos state reports. The ssh step that runs again is left out:
// under a read it would repeat the first. With several servers and no --server it reports each.
func macReport(p enginecli.Env, server *string) func(context.Context, engine.Call) (engine.Report, error) {
	return func(context.Context, engine.Call) (engine.Report, error) {
		confs, grouped, err := stateConfs(p, server)
		if err != nil {
			return engine.Report{}, err
		}
		return engine.Report{Read: macRead(p, server), Print: func(ctx context.Context, c engine.Call) error {
			term := effect.NewTerminal(strings.NewReader(""), c.Err)
			var docs []serverDoc
			for _, sc := range confs {
				reports, err := macOf(p, sc.Name, sc.Conf, p.Machine, linesOf(term, c.Path), nil).State(ctx)
				docs = append(docs, serverDoc{Name: sc.Name, Steps: reports})
				if err != nil {
					term.Flush()
					return err
				}
			}
			term.Flush()
			if c.JSON {
				doc := macDoc{Version: p.Version, Steps: docs[0].Steps}
				if grouped {
					doc = macDoc{Version: p.Version, Servers: docs}
				}
				return engine.WriteJSON(c.Out, doc)
			}
			return writeMacState(c.Out, docs, grouped)
		}}, nil
	}
}
