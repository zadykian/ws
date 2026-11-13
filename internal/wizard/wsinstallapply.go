package wizard

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Apply runs the remote script's install, which prints install.sh's lines as they come.
func (i *WsInstall) Apply(ctx context.Context, _ step.Plan) error {
	i.Out.Say("%s: installing ws %s on ws", stage.Install, i.Version)
	err := i.Ws.Install(ctx, i.Version, i.URL)
	if err != nil {
		return fmt.Errorf("%s: install.sh failed on ws, as above; run the wizard again", stage.Install)
	}
	return nil
}
