package cli

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// serverTarget is ws server ubuntu apply's side of the verbs. opts holds the flags, read when the
// command runs.
func serverTarget(p enginecli.Env, opts *applyOptions) engine.Target {
	return engine.Target{Wizard: wizardSide(p), Prepare: func(_ context.Context, c engine.Call) (engine.Run, error) {
		r, err := prepareServer(c, p, *opts)
		if err != nil {
			return engine.Run{}, err
		}
		r.term.Flush() // Prepare's lines wait for no run: Apply says the run's own.
		return runOf(r.cfg, r.list), nil
	}, Refuse: func(engine.Call) string {
		if p.Euid != 0 {
			return "run it as root, by sudo: ws's secrets and host_vars are root's"
		}
		return ""
	}, NoScreen: "it needs a terminal and no --plain: anywhere else, run ws server ubuntu secrets " +
		"and ws server ubuntu values set",
		Sheets: func(context.Context, engine.Call) ([]tui.Sheet, func(context.Context), error) {
			return serverSheets(p), nil, nil
		}, State: func(_ context.Context, c engine.Call) (engine.Report, error) {
			return serverReport(c, p, *opts)
		}}
}

// serverReport readies what ws server ubuntu state reports.
func serverReport(c engine.Call, p enginecli.Env, opts applyOptions) (engine.Report, error) {
	r, err := prepareServer(c, p, opts)
	if err != nil {
		return engine.Report{}, err
	}
	// State reads the machine and takes no secrets. A step that needs them would write them out to
	// plan, as ikeward's install does, and a report has no use for them.
	list, err := r.roles(secrets.Secrets{Unread: true})
	if err != nil {
		return engine.Report{}, engine.UsageError{Err: err}
	}
	return engine.Report{
		M: r.cfg.M,
		Read: func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
			// The read takes the screen's effects as a run does, as the steps don't own the terminal under it.
			states, err := role.State(ctx, onScreen(r.cfg, b), list)
			return engine.RoleSections(states), err
		},
		Print: func(ctx context.Context, c engine.Call) error {
			states, err := role.State(ctx, r.cfg, list)
			r.term.Flush()
			if err != nil {
				return err
			}
			if c.JSON {
				return engine.WriteJSON(c.Out, role.StateDoc{Version: p.Version, Roles: states})
			}
			return engine.WriteRoles(c.Out, states)
		},
	}, nil
}

// runOf is the run of the roles in list under cfg's options.
func runOf(cfg role.Config, list []role.Role) engine.Run {
	run := engine.Run{M: cfg.M, Listed: 1}
	run.Groups = func(check bool) []tui.Group {
		cfg.Check = check
		groups := []tui.Group{tui.NewGroup(role.Prepare, cfg.Prepare)}
		for _, r := range list {
			groups = append(groups, tui.NewGroup(r.Name, cfg.Steps(r)))
		}
		return groups
	}
	run.Do = func(ctx context.Context, e engine.Effects, o step.Options) (string, error) {
		at := cfg
		at.M, at.Out, at.Observer, at.Stop = e.M, e.Out, o.Observer, o.Stop
		at.Check, at.Prepare = o.Check, prepareSteps(e.M, e.Out)
		changed, err := role.Run(ctx, at, list)
		if err != nil {
			return "", err
		}
		return role.Summary(changed, o.Check), nil
	}
	run.Choose = func(choice tui.Choice) (engine.Run, step.Options, error) {
		cfg.Only = choice.Only
		chosen, err := role.Select(list, cfg.Only)
		return runOf(cfg, chosen), step.Options{Check: choice.Check}, err
	}
	return run
}

// serverRun is what a command that reads or sets the server up has made ready.
type serverRun struct {
	cfg    role.Config
	list   []role.Role
	values values.Values
	term   *effect.Terminal
	out    engine.Lines
}

// prepareServer checks where the command runs, then loads the values and the secrets that the
// roles read. The steps of --only follow the values, so they are checked last.
func prepareServer(call engine.Call, p enginecli.Env, opts applyOptions) (serverRun, error) {
	if call.Selected && len(opts.Only) == 0 {
		return serverRun{}, engine.UsageError{Err: errors.New("--only names no role")}
	}
	_, err := role.Select(ubuntu.Roles(values.Values{}, secrets.Secrets{}), opts.Only)
	if err != nil {
		return serverRun{}, engine.UsageError{Err: err}
	}
	text, err := p.Machine.ReadFile(osRelease)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return serverRun{}, err
	}
	if why := serverRefusal(p.Euid, text); why != "" {
		return serverRun{}, errors.New(why)
	}
	term := effect.NewTerminal(strings.NewReader(""), call.Err)
	out := linesOf(term, call.Path)
	v, err := values.Load(p.Machine, serverVars)
	if err != nil {
		return serverRun{}, err
	}
	// ws_home is known now: the machine's writes under it stay inside it.
	if m, ok := p.Machine.(local.OSMachine); ok {
		m.Confine = v.Home
		p.Machine = m
	}
	sec, err := loadSecrets(p.Machine, opts.Secrets, p.Getenv)
	if err != nil {
		return serverRun{}, err
	}
	cfg := role.Config{Options: opts.Options, M: p.Machine, U: units.OSUnits{}, Out: out, Observer: out,
		Prepare: prepareSteps(p.Machine, out)}
	r := serverRun{cfg: cfg, values: v, term: term, out: out}
	if r.list, err = r.roles(sec); err != nil {
		return serverRun{}, engine.UsageError{Err: err}
	}
	if err := cfg.CheckSteps(r.list); err != nil {
		return serverRun{}, engine.UsageError{Err: err}
	}
	return r, nil
}

// roles returns the roles that --only names, closed over r's values and the secrets s.
func (r serverRun) roles(s secrets.Secrets) ([]role.Role, error) {
	return role.Select(ubuntu.Roles(r.values, s), r.cfg.Only)
}

// prepareSteps is what every role needs: ca-certificates serves the steps' downloads over HTTPS.
func prepareSteps(m effect.Machine, out effect.Out) []step.Step {
	return []step.Step{resource.Group{Title: "Packages", Parts: []resource.Resource{apt.Packages{
		M: m, Out: out, Names: []string{"ca-certificates"},
	}}}}
}

// onScreen is cfg as a work beside a screen takes it: on the screen's effects, with its output and
// its events on b.
func onScreen(cfg role.Config, b *tui.Bridge) role.Config {
	e, o := b.Effects(), b.Options(step.Options{})
	cfg.M, cfg.Observer, cfg.Stop = e.M, o.Observer, o.Stop
	cfg.Out, cfg.Prepare = b, prepareSteps(e.M, b)
	return cfg
}

// loadSecrets reads the identity from the unit's credential and decrypts the file, or returns no
// secrets where path is empty.
func loadSecrets(m effect.Machine, path string, getenv func(string) string) (secrets.Secrets, error) {
	if path == "" {
		return secrets.Secrets{}, nil
	}
	id, err := secrets.ReadCredential(m, getenv)
	if err != nil {
		return secrets.Secrets{}, err
	}
	s, err := secrets.Load(m, path, id)
	s.Whole = err == nil
	return s, err
}
