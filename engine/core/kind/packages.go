package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
)

// aptName is a package's name, as Debian's policy writes one, and for a pin a version or a glob
// after an =. It starts with no dash and holds no slash, so that no option or path reaches
// apt-get. It ends in no dash, which apt-get's install reads as a removal of the name before it.
var aptName = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*[a-z0-9+.](=[A-Za-z0-9.+~:*-]*[A-Za-z0-9.+~:*])?$`)

// packagesSpec is a unit of the kind packages: apt's packages whose names its template gives, one
// or more a line. mise's own apt takes a fixed list; this kind's list follows the machine, as the
// template fills when the unit plans. Recommends follow apt's default.
type packagesSpec struct {
	base
	Template string `toml:"template"`
	// Simulate checks that apt finds each name, and installs none. It tries a role's names on a
	// machine that has no use for them, so that CI fails where the archive drops one.
	Simulate bool `toml:"simulate"`
}

func (s packagesSpec) parts(e Env, dir fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if s.Template == "" {
		return nil, errors.New("packages take a template")
	}
	if err := checkTemplateName(s.Template); err != nil {
		return nil, err
	}
	if _, err := fs.Stat(dir, s.Template); err != nil {
		return nil, err
	}
	return func(ctx context.Context) ([]resource.Resource, error) {
		text, err := render(ctx, e, dir, s.Template)
		if err != nil {
			return nil, err
		}
		names := strings.Fields(string(text))
		for _, n := range names {
			if !aptName.MatchString(n) {
				return nil, fmt.Errorf("%s gives %q, which is no package's name", s.Template, n)
			}
		}
		if len(names) == 0 {
			return nil, nil
		}
		if s.Simulate {
			return []resource.Resource{simulated{m: e.M, template: s.Template, names: names}}, nil
		}
		return []resource.Resource{apt.Packages{M: e.M, Out: e.Out, Names: names, Recommends: true}}, nil
	}, nil
}

// simulated is a packages unit that simulate sets: apt-get's simulated install of its names, which
// reads apt's lists and changes nothing.
type simulated struct {
	m        effect.Machine
	template string
	names    []string
}

// Plan fails where apt finds no package of a name, and plans nothing otherwise.
func (s simulated) Plan(ctx context.Context) ([]resource.Change, error) {
	if _, err := s.m.Run(ctx, "apt-get", append([]string{"install", "-s"}, s.names...)...); err != nil {
		return nil, fmt.Errorf("apt finds not every package that %s names: %w", s.template, err)
	}
	return nil, nil
}
