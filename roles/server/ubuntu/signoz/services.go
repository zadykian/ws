package signoz

import (
	"context"
	"fmt"
	"slices"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// restartAlways is the restart policy of the services that keep running, as against the one-shot
// ones, which compose runs again at each up.
const restartAlways = "unless-stopped"

// Services returns the names of the services of compose.yaml that keep running, sorted. It reads
// the file as compose's model, which resolves what plain YAML would leave, such as the anchors
// and the short forms. It resolves no environment, as root.env may not exist yet.
func Services(ctx context.Context) ([]string, error) {
	content, err := deployment.ReadFile("deployment/compose.yaml")
	if err != nil {
		return nil, err
	}
	project, err := loader.LoadWithContext(ctx, types.ConfigDetails{
		WorkingDir:  "/",
		ConfigFiles: []types.ConfigFile{{Filename: "compose.yaml", Content: content}},
		Environment: types.Mapping{},
	}, func(o *loader.Options) {
		o.SkipResolveEnvironment = true
		o.SetProjectName(Project, true)
	})
	if err != nil {
		return nil, fmt.Errorf("compose.yaml: %w", err)
	}
	var names []string
	for name, s := range project.Services {
		if s.Restart == restartAlways {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names, nil
}
