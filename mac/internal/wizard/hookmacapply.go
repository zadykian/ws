package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/step"
)

// Name is the stage's name.
func (*HookMac) Name() string { return hookMacStage }

// WaitsOn is the gh stage.
func (*HookMac) WaitsOn() string { return ghStage }

// Apply runs the commands of the plan. After a clone it plans again, as the clone may have hooks and
// a Brewfile.
func (h *HookMac) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var r hookRun
		switch c := c.(type) {
		case hookClone:
			r = c.hookRun
		case hookRun:
			r = c
		}
		h.Out.Say("%s: running %s", hookMacStage, r.text)
		if err := h.M.Pass(ctx, false, r.argv[0], r.argv[1:]...); err != nil {
			return fmt.Errorf("%s: %s failed, as above", hookMacStage, r.text)
		}
		if _, ok := c.(hookClone); ok {
			next, err := h.Plan(ctx)
			if err != nil {
				return err
			}
			return h.Apply(ctx, next)
		}
	}
	return nil
}

// Hand adds gh auth setup-git to what stays by hand, where git's config names no credential command
// of gh's for github.com, as ws mac apply's gitconfig does where mac.conf gives git a name. Through that
// command, pushes from the Mac's clones take gh's login.
func (h *HookMac) Hand(ctx context.Context, hand *Hand) {
	out, _ := h.M.Run(ctx, "git", "config", "--global", "--includes", "--get-all",
		"credential.https://github.com.helper")
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasSuffix(l, " auth git-credential") {
			return
		}
	}
	hand.Add("Pushes from the Mac's clones to GitHub: gh auth setup-git, which has git take gh's login " +
		"(docs/mac-git.md).")
}
