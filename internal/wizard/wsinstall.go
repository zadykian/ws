package wizard

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/check"
	"github.com/zadykian/ws/step"
	"golang.org/x/mod/semver"
)

// installStage names the stage in its messages.
const installStage = "ws on ws"

// WsInstall is the stage that puts this ws's release on ws, by the remote script's install.
type WsInstall struct {
	Remote *Remote
	// URL is the address of the releases that Version comes from.
	Version, URL string
	Out          effect.Out
}

// Name is the stage's name.
func (*WsInstall) Name() string { return installStage }

// WaitsOn is the Sudo stage, which the remote script needs.
func (*WsInstall) WaitsOn() string { return sudoStage }

type wsInstall struct{ version string }

func (c wsInstall) String() string {
	return fmt.Sprintf("%s: would install ws %s on ws", installStage, c.version)
}

// Plan is empty where ws holds this version. It stops where ws holds a newer release than this
// ws's, so that an update on ws is not undone. It stops too where this ws is no release, or where
// URL is no address to fetch from.
func (i *WsInstall) Plan(ctx context.Context) (step.Plan, error) {
	if i.Remote.Status(ctx) && i.Remote.Get("ws") == i.Version {
		return step.Plan{}, nil
	}
	if have := i.Remote.Get("ws"); i.Remote.Status(ctx) && check.Release(i.Version) && check.Release(have) &&
		semver.Compare(have, i.Version) > 0 {
		return step.Plan{}, fmt.Errorf("%s: ws holds %s, newer than this ws's %s: run ws mac wizard --update "+
			"to bring this ws to the newest", installStage, have, i.Version)
	}
	if !check.Release(i.Version) {
		return step.Plan{}, fmt.Errorf("%s: this ws is %s, not a release: put a linux build of the same tree at "+
			"/usr/local/bin/ws on ws by hand (tools/build), then run the wizard again", installStage, i.Version)
	}
	if !check.ReleasesURL(i.URL) {
		return step.Plan{}, fmt.Errorf("%s: WS_RELEASES_URL is %q: it must start with https://, or with "+
			"http://127.0.0.1: and hold no blank or quote", installStage, i.URL)
	}
	return step.Plan{Changes: []step.Change{wsInstall{i.Version}}}, nil
}
