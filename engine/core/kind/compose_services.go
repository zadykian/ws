package kind

import (
	"context"
	"fmt"
	"slices"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// composeFile is the file of a project's directory that compose reads, of the names it looks for.
const composeFile = "compose.yaml"

// keepRunning returns the services of a project's compose.yaml that keep running, sorted: those
// whose restart policy is always or unless-stopped. The others run once, and compose runs them again
// at each up. It reads the file as compose's model, which resolves what plain YAML would leave, such
// as the anchors and the short forms. It resolves no environment, as a file that an env_file names
// may not exist yet, and fills no variable, as each would be blank and warn.
func keepRunning(ctx context.Context, project string, compose []byte) ([]string, error) {
	p, err := loader.LoadWithContext(ctx, types.ConfigDetails{
		WorkingDir:  "/",
		ConfigFiles: []types.ConfigFile{{Filename: composeFile, Content: compose}},
		Environment: types.Mapping{},
	}, func(o *loader.Options) {
		o.SkipResolveEnvironment = true
		o.SkipInterpolation = true
		o.SetProjectName(project, true)
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", composeFile, err)
	}
	var names []string
	for name, s := range p.Services {
		if s.Restart == types.RestartPolicyAlways || s.Restart == types.RestartPolicyUnlessStopped {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names, nil
}
