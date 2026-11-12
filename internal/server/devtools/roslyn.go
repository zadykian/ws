package devtools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// dotnetBin is the .NET SDK's command, from devtools' packages.
const dotnetBin = "/usr/bin/dotnet"

// roslynTool is roslyn-language-server at the pinned version, a .NET tool. The tool's store keeps a
// directory per installed version. They are read there, and not with dotnet tool list. The first
// run of dotnet with each SDK version writes files into the home, an HTTPS development certificate
// among them, and a plan must change nothing. NuGet names the directories in lower case.
type roslynTool struct {
	M    effect.Machine
	User host.User
	// Path is where dotnet tool install --global puts the command, and where languages.toml names it.
	Path string
	// Version is the pin. Empty means the package's.
	Version string
}

func (r roslynTool) version() string {
	if r.Version == "" {
		return roslynVersion
	}
	return r.Version
}

type installRoslyn struct {
	r         roslynTool
	installed []string
}

// String names the version that a run installs, and the ones it replaces.
func (c installRoslyn) String() string {
	if len(c.installed) == 0 {
		return "install roslyn-language-server " + c.r.version()
	}
	return fmt.Sprintf("replace roslyn-language-server %s with %s", strings.Join(c.installed, ", "), c.r.version())
}

// Apply updates the tool, which installs it where missing too. --allow-downgrade lets the pin
// go back. NuGet asks every feed the user's NuGet config names, and one it can't reach fails the
// install, so --source asks nuget.org alone.
func (c installRoslyn) Apply(ctx context.Context) error {
	if _, err := c.r.M.Stat(dotnetBin); errors.Is(err, fs.ErrNotExist) {
		return errors.New("roslyn-language-server is installed with the .NET SDK of the devtools role, " +
			"and runs on its .NET 10")
	} else if err != nil {
		return err
	}
	return c.r.User.Pass(ctx, c.r.M, "env", "DOTNET_CLI_TELEMETRY_OPTOUT=1", dotnetBin, "tool", "update",
		"--global", "roslyn-language-server", "--version", c.r.version(), "--allow-downgrade",
		"--source", "https://api.nuget.org/v3/index.json")
}

// Plan reads the store's directories.
func (r roslynTool) Plan(context.Context) ([]host.Change, error) {
	installed, err := r.M.ReadDir(path.Dir(r.Path) + "/.store/roslyn-language-server")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if slices.Equal(installed, []string{strings.ToLower(r.version())}) {
		return nil, nil
	}
	return []host.Change{installRoslyn{r, installed}}, nil
}
