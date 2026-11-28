// Package settings reads the tunnel daemon's settings and key, and its status for ws macos
// tunnel status.
package settings

import (
	"context"
	"fmt"
	"net/netip"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/wgkey"
	tunnel "github.com/zadykian/ws/macos/roles/tunnel/plan"
)

// Paths are the daemon's files. Conf is tunnel.conf, which setup copies from the server's file. Key
// is private.key, the Mac's key. Routes is routes.d, and Yield the yield, empty for none. State is
// what the daemon writes for status, Log launchd's file for its output. Root holds the directory
// of each server, the daemon's own among them, beside the key; none reads no other server.
type Paths struct{ Conf, Key, Routes, State, Log, Root, Yield string }

// Settings are tunnel.conf and routes.d as read. Routes leaves out the bad lines of routes.d, the
// networks that another server's routes.d holds, and those that the yield holds. Others are the
// other servers' directories, and Yield the yield's networks.
type Settings struct {
	Conf     tunnel.Conf
	Routes   []netip.Prefix
	Others   []Other
	Yield    []netip.Prefix
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
	s.Yield, skipped = readYield(m, p.Yield)
	s.Warnings = append(s.Warnings, skipped...)
	s.Routes, apart = tunnel.Yield(s.Routes, s.Yield)
	s.Warnings = append(s.Warnings, apart...)
	text, err := m.ReadFile(p.Conf)
	if err != nil {
		return s, fmt.Errorf("cannot read %s: run mise -C /opt/ws/macos bootstrap", p.Conf)
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

// readYield reads the yield at file. A missing file yields nothing.
func readYield(m effect.Machine, file string) ([]netip.Prefix, []string) {
	if file == "" {
		return nil, nil
	}
	text, err := m.ReadFile(file)
	if err != nil {
		return nil, nil
	}
	return tunnel.ParseYield(string(text))
}

// ReadKey reads the Mac's private key, in base64, from file.
func ReadKey(m effect.Machine, file string) (string, error) {
	text, err := m.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: run mise -C /opt/ws/macos bootstrap", file)
	}
	key := strings.TrimSpace(string(text))
	if _, err := wgkey.Decode(key); err != nil {
		return "", fmt.Errorf("%s: %w", file, err)
	}
	return key, nil
}
