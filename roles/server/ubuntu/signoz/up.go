package signoz

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

// Up is the step that starts the project's containers. Compose's up runs the one-shot services
// again, the schema migrator among them. So the step runs only while a recreation is due, or while
// a service that keeps running is down. Without systemd booted there is no Docker daemon to ask,
// and the step skips.
type Up struct {
	M   effect.Machine
	Out effect.Out
	Dir string
}

type up struct {
	u     Up
	force bool
}

func (c up) String() string {
	if c.force {
		return "start SigNoz's containers, recreating them"
	}
	return "start SigNoz's containers"
}

func (c up) Apply(ctx context.Context) error {
	args := []string{"compose", "--project-directory", c.u.Dir, "--project-name", Project,
		"up", "--detach", "--quiet-pull", "--remove-orphans"}
	if c.force {
		args = append(args, "--force-recreate")
	}
	if err := c.u.M.Pass(ctx, true, "docker", args...); err != nil {
		return err
	}
	_, err := c.u.M.RunRoot(ctx, "rm", "-f", "--", markPath(c.u.Dir))
	return err
}

// Name returns the step's name.
func (Up) Name() string { return "signoz up" }

// WaitsOn returns the step before.
func (Up) WaitsOn() string { return "signoz password" }

// Plan reads the mark and, without it, the running services.
func (u Up) Plan(ctx context.Context) (step.Plan, error) {
	if !host.Booted(u.M) {
		u.Out.Say("%s: skipped, systemd is not running", u.Name())
		return step.Plan{}, nil
	}
	_, err := u.M.RootAttrs(markPath(u.Dir))
	if err == nil {
		return step.Plan{Changes: []step.Change{up{u, true}}}, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return step.Plan{}, err
	}
	want, err := Services(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	out, err := u.M.RunRoot(ctx, "docker", "compose", "--project-name", Project,
		"ps", "--status", "running", "--services")
	if err != nil {
		return step.Plan{}, err
	}
	running := strings.Fields(string(out))
	if slices.ContainsFunc(want, func(s string) bool { return !slices.Contains(running, s) }) {
		return step.Plan{Changes: []step.Change{up{u, false}}}, nil
	}
	return step.Plan{}, nil
}

// Apply starts the containers.
func (Up) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := c.(up).Apply(ctx); err != nil {
			return fmt.Errorf("%s: %w", step.Headline(c), err)
		}
	}
	return nil
}
