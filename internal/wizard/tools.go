package wizard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// toolsStage names the stage of the Command Line Tools.
const toolsStage = "Tools"

// toolsWait is how long the stage waits for the install of the Command Line Tools, and toolsPoll
// how often it looks.
const (
	toolsWait = 60 * time.Minute
	toolsPoll = 10 * time.Second
)

// Tools is the stage of the Command Line Tools, which Homebrew needs. It opens their install
// dialog and waits for the click.
type Tools struct {
	M     effect.Machine
	Out   effect.Out
	Sleep func(time.Duration)
}

// Name is the stage's name.
func (*Tools) Name() string { return toolsStage }

// WaitsOn is empty: the tools come first.
func (*Tools) WaitsOn() string { return "" }

// toolsInstall is the change that opens the install dialog.
type toolsInstall struct{}

func (toolsInstall) String() string {
	return toolsStage + ": would run xcode-select --install, for the Command Line Tools"
}

// done tells whether the Command Line Tools, or Xcode, are there: xcode-select names a directory
// that holds git. Never git itself, whose shim in /usr/bin would open their install.
func (t *Tools) done(ctx context.Context) bool {
	out, err := t.M.Run(ctx, "xcode-select", "-p")
	dir := strings.TrimSpace(string(out))
	if err != nil || dir == "" {
		return false
	}
	info, err := t.M.Stat(dir + "/usr/bin/git")
	return err == nil && !info.Mode.IsDir() && info.Mode&0o111 != 0
}

// Plan plans the install where the tools are missing.
func (t *Tools) Plan(ctx context.Context) (step.Plan, error) {
	if t.done(ctx) {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{toolsInstall{}}}, nil
}

// Apply opens the dialog and waits up to an hour for the tools. A dialog cancelled opens at the
// next run.
func (t *Tools) Apply(ctx context.Context, _ step.Plan) error {
	// An error is ignored: a dialog may be open from an earlier run.
	_, _ = t.M.Run(ctx, "xcode-select", "--install")
	t.Out.Say("%s: click Install in the Command Line Tools' dialog; waiting up to 60 minutes", toolsStage)
	sleep := t.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	for waited := time.Duration(0); !t.done(ctx); waited += toolsPoll {
		if waited >= toolsWait {
			return fmt.Errorf("%s: no Command Line Tools after 60 minutes", toolsStage)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		sleep(toolsPoll)
	}
	return nil
}
