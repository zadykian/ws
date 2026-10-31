package daemon

import (
	"context"
	"fmt"
	"net/netip"
	"path"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/wgkey"
)

// Dir is where root keeps the daemon's settings: private.key, the Mac's key; tunnel.conf, which
// setup copies from mac.conf; and routes.d, whose files hold a network a line.
const Dir = "/etc/ws-tunnel"

// Settings are tunnel.conf and routes.d as read, with the lines of routes.d left out.
type Settings struct {
	Conf     tunnel.Conf
	Routes   []netip.Prefix
	Warnings []string
}

// ReadSettings reads tunnel.conf and routes.d from dir. It fails where tunnel.conf is missing or
// holds a line it does not take, and then still returns the routes, as the two are read apart. A
// file of routes.d that it cannot read is left out.
func ReadSettings(ctx context.Context, m effect.Machine, dir string) (Settings, error) {
	files := map[string][]byte{}
	listed, _ := m.Run(ctx, "ls", "-1", path.Join(dir, "routes.d")) // none where it is missing
	for _, name := range strings.Fields(string(listed)) {
		if data, err := m.ReadFile(path.Join(dir, "routes.d", name)); err == nil {
			files[name] = data
		}
	}
	var s Settings
	s.Routes, s.Warnings = tunnel.ParseRoutes(files)
	text, err := m.ReadFile(path.Join(dir, "tunnel.conf"))
	if err != nil {
		return s, fmt.Errorf("cannot read %s/tunnel.conf: run mac/setup.sh", dir)
	}
	s.Conf, err = tunnel.ParseConf(string(text))
	return s, err
}

// ReadKey reads the Mac's private key, in base64, from dir.
func ReadKey(m effect.Machine, dir string) (string, error) {
	text, err := m.ReadFile(path.Join(dir, "private.key"))
	if err != nil {
		return "", fmt.Errorf("cannot read %s/private.key: run mac/setup.sh", dir)
	}
	key := strings.TrimSpace(string(text))
	if _, err := wgkey.Decode(key); err != nil {
		return "", fmt.Errorf("%s/private.key: %w", dir, err)
	}
	return key, nil
}
