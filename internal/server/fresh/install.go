package fresh

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// install makes the parts that install the downloaded .deb, then remove it.
func install(m effect.Machine) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		arch, err := architecture(ctx, m)
		if err != nil {
			return nil, err
		}
		path := debPath(arch)
		ok, err := installed(ctx, m)
		if err != nil {
			return nil, err
		}
		if ok {
			return []host.Resource{host.Absent{M: m, Path: path}}, nil
		}
		return []host.Resource{host.Deb{M: m, Path: path}, host.Absent{M: m, Path: path}}, nil
	}
}
