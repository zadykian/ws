package helix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// hxBin is where Ubuntu's package puts hx.
const hxBin = "/usr/bin/hx"

// Grammars fetches and builds the tree-sitter grammars that languages.toml names: Ubuntu's hx
// ships none, and without them hx has no syntax highlighting and no syntax-aware selections.
//
// hx fetches a grammar only where its checkout isn't at the revision hx pins, and builds one only
// where its sources are newer than its library, and a plan can't see a fetch that is pending. So a
// stamp of hx's version and the grammar list, kept beside its config, stands for what the last
// run fetched and built. Both commands run where the stamp differs or a library is missing, and a
// run with nothing to do changes nothing and needs no network.
type Grammars struct {
	M    effect.Machine
	User host.User
	// Names are hx's grammar names, and the libraries' too.
	Names    []string
	UID, GID int
}

func (g Grammars) stampPath() string { return g.User.Home + "/.config/helix/.ws-grammars" }

func (g Grammars) libPath(name string) string {
	return g.User.Home + "/.config/helix/runtime/grammars/" + name + ".so"
}

type buildGrammars struct {
	g     Grammars
	stamp []byte
	why   string
}

func (c buildGrammars) String() string { return "fetch and build helix's grammars: " + c.why }

func (c buildGrammars) Apply(ctx context.Context) error {
	for _, verb := range []string{"fetch", "build"} {
		if err := c.g.User.Pass(ctx, c.g.M, "hx", "--grammar", verb); err != nil {
			return err
		}
	}
	return c.g.M.WriteAtomic(c.g.stampPath(), c.stamp, effect.Attrs{UID: c.g.UID, GID: c.g.GID, Mode: 0o644})
}

// Plan compares the stamp, and looks for each library. Where hx is not installed yet, as in a dry
// run of a new machine, it plans the build.
func (g Grammars) Plan(ctx context.Context) ([]host.Change, error) {
	if _, err := g.M.Stat(hxBin); errors.Is(err, fs.ErrNotExist) {
		return []host.Change{buildGrammars{g: g, why: "hx is not installed yet"}}, nil
	} else if err != nil {
		return nil, err
	}
	version, err := g.User.Output(ctx, g.M, "hx", "--version")
	if err != nil {
		return nil, fmt.Errorf("hx --version: %w", err)
	}
	stamp := []byte(strings.TrimSpace(string(version)) + "\n" + strings.Join(g.Names, "\n") + "\n")
	c := buildGrammars{g: g, stamp: stamp}
	have, err := g.M.ReadRootFile(g.stampPath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		c.why = "no earlier run"
	case err != nil:
		return nil, err
	case !bytes.Equal(have, stamp):
		c.why = "hx or the grammar list changed"
	}
	for _, name := range g.Names {
		if c.why != "" {
			break
		}
		if _, err := g.M.Stat(g.libPath(name)); errors.Is(err, fs.ErrNotExist) {
			c.why = name + " has no library"
		} else if err != nil {
			return nil, err
		}
	}
	if c.why == "" {
		return nil, nil
	}
	return []host.Change{c}, nil
}
