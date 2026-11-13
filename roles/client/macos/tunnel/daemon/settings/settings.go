// Package settings reads the tunnel daemon's settings and key, and its status for ws client macos
// tunnel status.
package settings

import (
	"context"
	"fmt"
	"net/netip"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
	"github.com/zadykian/ws/roles/internal/wgkey"
)

// Paths are the daemon's files. Conf is tunnel.conf, which setup copies from mac.conf. Key is
// private.key, the Mac's key. Routes is routes.d, whose files hold a network a line. State is what
// the daemon writes for status, Log launchd's file for its output.
type Paths struct{ Conf, Key, Routes, State, Log string }

// Settings are tunnel.conf and routes.d as read, with the lines of routes.d left out.
type Settings struct {
	Conf     tunnel.Conf
	Routes   []netip.Prefix
	Warnings []string
}

// ReadSettings reads tunnel.conf and routes.d at p. It fails where tunnel.conf is missing or
// holds a line it does not take, and then still returns the routes, as the two are read apart. A
// file of routes.d that it cannot read is left out.
func ReadSettings(ctx context.Context, m effect.Machine, p Paths) (Settings, error) {
	files := map[string][]byte{}
	listed, _ := m.Run(ctx, "ls", "-1", p.Routes) // none without routes.d
	for _, name := range strings.Fields(string(listed)) {
		if data, err := m.ReadFile(path.Join(p.Routes, name)); err == nil {
			files[name] = data
		}
	}
	var s Settings
	s.Routes, s.Warnings = tunnel.ParseRoutes(files)
	text, err := m.ReadFile(p.Conf)
	if err != nil {
		return s, fmt.Errorf("cannot read %s: run ws client macos apply", p.Conf)
	}
	s.Conf, err = tunnel.ParseConf(string(text))
	return s, err
}

// ReadKey reads the Mac's private key, in base64, from file.
func ReadKey(m effect.Machine, file string) (string, error) {
	text, err := m.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: run ws client macos apply", file)
	}
	key := strings.TrimSpace(string(text))
	if _, err := wgkey.Decode(key); err != nil {
		return "", fmt.Errorf("%s: %w", file, err)
	}
	return key, nil
}
