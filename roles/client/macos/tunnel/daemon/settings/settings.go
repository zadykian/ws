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
// the daemon writes for status, Log launchd's file for its output. Root holds the directory of each
// server, the daemon's own among them, beside the key; none reads no other server.
type Paths struct{ Conf, Key, Routes, State, Log, Root string }

// Settings are tunnel.conf and routes.d as read, with the lines of routes.d left out and the
// networks that another server's routes.d holds. Others are the other servers' directories.
type Settings struct {
	Conf     tunnel.Conf
	Routes   []netip.Prefix
	Others   []Other
	Warnings []string
}

// ReadSettings reads tunnel.conf and routes.d at p. It fails where tunnel.conf is missing or
// holds a line it does not take, and then still returns the routes, as the two are read apart. A
// file of routes.d that it cannot read is left out.
func ReadSettings(ctx context.Context, m effect.Machine, p Paths) (Settings, error) {
	var s Settings
	s.Routes, s.Warnings = readRoutes(ctx, m, p.Routes)
	var skipped []string
	s.Others, skipped = readOthers(ctx, m, p)
	s.Warnings = append(s.Warnings, skipped...)
	held := map[string][]netip.Prefix{}
	for _, o := range s.Others {
		held[o.Name] = o.Routes
	}
	var apart []string
	s.Routes, apart = tunnel.Apart(s.Routes, held)
	s.Warnings = append(s.Warnings, apart...)
	text, err := m.ReadFile(p.Conf)
	if err != nil {
		return s, fmt.Errorf("cannot read %s: run ws client macos apply", p.Conf)
	}
	s.Conf, err = tunnel.ParseConf(string(text))
	return s, err
}

// readRoutes reads the files of the routes.d at dir, none from a missing one.
func readRoutes(ctx context.Context, m effect.Machine, dir string) ([]netip.Prefix, []string) {
	files := map[string][]byte{}
	listed, _ := m.Run(ctx, "ls", "-1", dir)
	for _, name := range strings.Fields(string(listed)) {
		if data, err := m.ReadFile(path.Join(dir, name)); err == nil {
			files[name] = data
		}
	}
	return tunnel.ParseRoutes(files)
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
