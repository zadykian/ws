package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// askServer is the step that asks the name of the server that the wizard sets up, where no --server
// names it and no server is set up. It acts on nothing: its answer is the name that the stages
// after it build on, so a plan, which reads alone, takes it.
type askServer struct {
	ask effect.Prompt
	tty bool
	// m and home find ~/.ssh/config, whose own Host a name must not take.
	m    effect.Machine
	home string
	// name is the server's name, which the step sets.
	name *string
}

// Name is the step's name.
func (*askServer) Name() string { return "Server name" }

// WaitsOn is empty: the step stands first, after the update.
func (*askServer) WaitsOn() string { return "" }

// Plan asks the name, offering ws, and asks again on a name of the wrong form. It needs a terminal.
func (a *askServer) Plan(ctx context.Context) (step.Plan, error) {
	if !a.tty {
		return step.Plan{}, errors.New("no terminal to ask the server's name: pass --server NAME")
	}
	name, err := a.ask.Ask(ctx, effect.Question{Text: "The server's name", Default: conf.FirstServer,
		Valid: func(s string) error {
			if !conf.IsName(s) {
				return errors.New("not a server's name: " + conf.NameForm)
			}
			return freeName(a.m, a.home, s)
		}})
	if err != nil {
		return step.Plan{}, err
	}
	*a.name = name
	return step.Plan{}, nil
}

// freeName refuses a name that ~/.ssh/config at home sets its own values for: config.d/ws would not
// win there. The wizard checks a new name before it writes the server's file, as apply refuses
// the file later.
func freeName(m effect.Machine, home, name string) error {
	text, err := m.ReadFile(home + "/.ssh/config")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if own := sshcfg.OwnHosts(string(text), name); len(own) > 0 {
		return fmt.Errorf("line %d of ~/.ssh/config, %s, sets its own values for %s: pick another name, or remove it",
			own[0].Line, own[0].Text, name)
	}
	return nil
}

// Apply has nothing to do.
func (*askServer) Apply(context.Context, step.Plan) error { return nil }
