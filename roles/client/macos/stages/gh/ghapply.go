package gh

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Apply makes what the plan lists, in order.
func (g *Gh) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		switch c := c.(type) {
		case *resource.Command:
			err = g.install(ctx, c)
		case ghLogin:
			err = g.login(ctx, c)
		case ghAdd:
			err = g.add(ctx, c)
		case ghLater:
			err = c.err
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// install runs the install that ghPath plans where PATH has no gh. The login after it runs gh at
// the path Homebrew gives it.
func (g *Gh) install(ctx context.Context, c *resource.Command) error {
	if err := c.Apply(ctx); err != nil {
		return fmt.Errorf("%s: %s failed", stage.GitHub, c.Line())
	}
	return nil
}

// login runs gh auth with the scopes. Its device flow shows a code to enter in the browser, at
// github.com/login/device. With no terminal, it stops, naming the command.
func (g *Gh) login(ctx context.Context, c ghLogin) error {
	if !g.Tty {
		return fmt.Errorf("%s: gh lacks a login to github.com with the scopes of the Mac's keys, and no "+
			"terminal is there for %s: run it, then the wizard again", stage.GitHub, c.line())
	}
	path, _, err := g.ghPath(ctx)
	if err != nil {
		return err
	}
	args := append(append([]string{"auth"}, c.args...), "-h", "github.com", "-s", ghScopes)
	if err := remote.Talk(ctx, g.F, g.M, path, args...); err != nil {
		return fmt.Errorf("%s: %s failed", stage.GitHub, c.line())
	}
	return nil
}

// add adds the key to the login's keys, unless a login just made lists it already.
func (g *Gh) add(ctx context.Context, c ghAdd) error {
	path, _, err := g.ghPath(ctx)
	if err != nil {
		return err
	}
	_, key, err := g.Key(ctx)
	if err != nil {
		return err
	}
	if listed, refresh, err := g.listed(ctx, path, key); err != nil {
		return err
	} else if listed && !refresh {
		return nil
	}
	if _, err := g.M.Run(ctx, path, "ssh-key", "add", c.pub, "--title", c.title); err != nil {
		return fmt.Errorf("%s: %s failed", stage.GitHub, c.line())
	}
	g.added = true
	g.Out.Say("%s: ran %s", stage.GitHub, c.line())
	return nil
}
