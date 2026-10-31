package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/sshcfg"
)

// key returns the .pub of the Mac's key for ws, as setup.PickKey finds it from mac.conf, and its
// type and base64.
func (g *Gh) key(ctx context.Context) (pub, key string, err error) {
	_, c, err := loadConf(confFile{g.Conf, ghStage, g.M, g.Out})
	if err != nil {
		return "", "", err
	}
	id, err := setup.PickKey(ctx, g.M, c, g.Home)
	if err != nil {
		return "", "", err
	}
	pub = sshcfg.Expand(id.Pub, g.Home)
	text, err := g.M.ReadFile(pub)
	if f := strings.Fields(string(text)); err == nil && len(f) >= 2 {
		return pub, f[0] + " " + f[1], nil
	}
	return "", "", fmt.Errorf("%s: cannot read the Mac's key for ws from %s", ghStage, pub)
}

func (g *Gh) hostname(ctx context.Context) string {
	out, err := g.M.Run(ctx, "hostname", "-s")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
