package tools

import (
	"context"
	"fmt"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"golang.org/x/mod/semver"
)

// WsInstall is the stage that puts this ws's release on ws, by the remote script's install.
type WsInstall struct {
	Ws *session.Session
	// URL is the address of the releases that Version comes from.
	Version, URL string
	Out          effect.Out
}

// Name is the stage's name.
func (*WsInstall) Name() string { return stage.Install }

// Side is ws: this release is installed there.
func (*WsInstall) Side() guided.Side { return session.Ws }

// WaitsOn is the Sudo stage, which the remote script needs.
func (*WsInstall) WaitsOn() string { return stage.Sudo }

type wsInstall struct{ version string }

func (c wsInstall) String() string {
	return fmt.Sprintf("%s: would install ws %s on ws", stage.Install, c.version)
}

// Plan is empty where ws holds this version. It stops where ws holds a newer release than this
// ws's, so that an update on ws is not undone. It stops too where this ws is no release, or where
// URL is no address to fetch from.
func (i *WsInstall) Plan(ctx context.Context) (step.Plan, error) {
	s, ok := i.Ws.Status(ctx)
	if ok && s.Version == i.Version {
		return step.Plan{}, nil
	}
	if have := s.Version; ok && check.Release(i.Version) && check.Release(have) &&
		semver.Compare(have, i.Version) > 0 {
		return step.Plan{}, fmt.Errorf("%s: ws holds %s, newer than this ws's %s: run ws client macos wizard --update "+
			"to bring this ws to the newest", stage.Install, have, i.Version)
	}
	if !check.Release(i.Version) {
		return step.Plan{}, fmt.Errorf("%s: this ws is %s, not a release: put a linux build of the same tree at "+
			"/usr/local/bin/ws on ws by hand (tools/build), then run the wizard again", stage.Install, i.Version)
	}
	if !check.ReleasesURL(i.URL) {
		return step.Plan{}, fmt.Errorf("%s: WS_RELEASES_URL is %q: it must start with https://, or with "+
			"http://127.0.0.1: and hold no blank or quote", stage.Install, i.URL)
	}
	return step.Plan{Changes: []step.Change{wsInstall{i.Version}}}, nil
}
