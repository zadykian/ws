package wizard

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/check"
	"github.com/zadykian/ws/step"
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

// WaitsOn is the Root's key stage, which the remote script needs.
func (*WsInstall) WaitsOn() string { return rootStage }

type wsInstall struct{ version string }

func (c wsInstall) String() string {
	return fmt.Sprintf("%s: would install ws %s on ws", installStage, c.version)
}

// Plan is empty where ws holds this version. It stops where this ws is no release, or URL is no
// address to fetch from.
func (i *WsInstall) Plan(ctx context.Context) (step.Plan, error) {
	if i.Remote.Status(ctx) && i.Remote.Get("ws") == i.Version {
		return step.Plan{}, nil
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
