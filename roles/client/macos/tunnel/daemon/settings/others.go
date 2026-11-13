package settings

import (
	"context"
	"fmt"
	"net/netip"
	"path"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// Other is what a daemon reads of another server's directory: its endpoints, from its
// tunnel.conf, and the networks of its routes.d.
type Other struct {
	Name   string
	Conf   tunnel.Conf // zero where its tunnel.conf is missing or refused
	Routes []netip.Prefix
}

// readOthers reads the directories of p.Root beside the daemon's own, by name. A server's
// warnings about its own routes.d stay with its daemon, which logs them.
func readOthers(ctx context.Context, m effect.Machine, p Paths) (others []Other, warnings []string) {
	if p.Root == "" {
		return nil, nil
	}
	own := path.Base(path.Dir(p.Conf))
	listed, _ := m.Run(ctx, "ls", "-1p", p.Root) // none without the directory
	for _, entry := range strings.Fields(string(listed)) {
		name, dir := strings.CutSuffix(entry, "/") // private.key is no server's directory
		if !dir || name == own {
			continue
		}
		o := Other{Name: name}
		if text, err := m.ReadFile(path.Join(p.Root, name, "tunnel.conf")); err == nil {
			if o.Conf, err = tunnel.ParseConf(string(text)); err != nil {
				warnings = append(warnings, fmt.Sprintf("leaves out server %s's endpoints: %v", name, err))
			}
		}
		o.Routes, _ = readRoutes(ctx, m, path.Join(p.Root, name, "routes.d"))
		others = append(others, o)
	}
	slices.SortFunc(others, func(a, b Other) int { return strings.Compare(a.Name, b.Name) })
	return others, warnings
}
