package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

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
}

func (s packagesSpec) parts(e Env, dir fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if s.Template == "" {
		return nil, errors.New("packages take a template")
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
		return []resource.Resource{apt.Packages{M: e.M, Out: e.Out, Names: names, Recommends: true}}, nil
	}, nil
}
