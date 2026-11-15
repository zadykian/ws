package tools

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Apply runs the remote script's install, which prints install.sh's lines as they come.
func (i *WsInstall) Apply(ctx context.Context, _ step.Plan) error {
	i.Out.Say("%s: installing ws %s on %s", stage.Install, i.Version, i.Server)
	err := i.Ws.Install(ctx, i.Version, i.URL)
	if err != nil {
		return fmt.Errorf("%s: install.sh failed on %s, as above; run the wizard again", stage.Install, i.Server)
	}
	return nil
}
