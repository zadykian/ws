package wizard

import (
	"context"
	"fmt"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// macUpdateStage names the stage in its messages.
const macUpdateStage = "ws on the Mac"

// MacUpdate is the stage of --update that puts the latest release of ws on the Mac, and runs the
// command again with it. It runs in a step.Run of its own, before the lock is taken: an Exec would
// leave the lock held.
type MacUpdate struct {
	M   effect.Machine
	Out effect.Out
	// Version is this ws's version, URL the address of the releases, and Home the Mac's home. Args
	// holds the arguments that the new ws gets, ws's own without the program.
	Version, URL, Home string
	Args               []string
}

// Name is the stage's name.
func (*MacUpdate) Name() string { return macUpdateStage }

// Side is the Mac: the stage installs ws's tool before the others.
func (*MacUpdate) Side() guided.Side { return OnMac }

// WaitsOn is empty: the stage comes before the others.
func (*MacUpdate) WaitsOn() string { return "" }

type macUpdate struct{ tag string }

func (c macUpdate) String() string {
	return fmt.Sprintf("%s: would install ws %s on the Mac, and run this command again with it", macUpdateStage, c.tag)
}

// latestTag returns the tag that the redirect of URL/latest names, the part after /tag/, or false
// where that is no release.
func latestTag(redirect string) (string, bool) {
	_, tag, found := strings.Cut(strings.TrimSpace(redirect), "/tag/")
	if !found || !check.Release(tag) {
		return "", false
	}
	return tag, true
}

// Plan reads the latest release and plans its install where this ws is another. A ws that is no
// release stays: it warns and plans nothing.
func (u *MacUpdate) Plan(ctx context.Context) (step.Plan, error) {
	if !check.Release(u.Version) {
		u.Out.Warn("%s: this ws is %s, not a release: --update leaves it", macUpdateStage, u.Version)
		return step.Plan{}, nil
	}
	if !check.ReleasesURL(u.URL) {
		return step.Plan{}, fmt.Errorf("WS_RELEASES_URL is %q: it must start with https://, or with "+
			"http://127.0.0.1: and hold no blank or quote", u.URL)
	}
	out, err := u.M.Run(ctx, "curl", "-fsS", "-o", "/dev/null", "-w", "%{redirect_url}", u.URL+"/latest")
	if err != nil {
		return step.Plan{}, fmt.Errorf("reading the latest release: %w", err)
	}
	tag, ok := latestTag(string(out))
	if !ok {
		return step.Plan{}, fmt.Errorf("%s/latest redirects to %q, not to the tag of a release", u.URL, strings.TrimSpace(string(out)))
	}
	if tag == u.Version {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{macUpdate{tag}}}, nil
}
