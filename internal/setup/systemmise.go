package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

const (
	// TreeDir is where ws's tree stands, which mise trusts.
	TreeDir = "/opt/ws"
	// MisercPath is mise's file for the envs that load, read before any config.
	MisercPath = "/etc/mise/miserc.toml"
	// ConfigPath is mise's config for the whole machine.
	ConfigPath = "/etc/mise/config.toml"
	// SelfContained is the env of the roles that a machine takes where it runs SigNoz and ends its
	// own tunnel. A host takes it always, and a guest where its values turn the mode on.
	SelfContained = "self-contained"
)

// Miserc returns the content of MisercPath for the envs of the machine. The set ws names the env of
// the side's files, then any other env it takes, which mise loads before any config. The secrets
// task adds its env by -E ws,secrets. The key env sits above the table, which TOML would put it in
// otherwise.
func Miserc(envs ...string) []byte {
	quoted := make([]string, len(envs))
	for i, e := range envs {
		quoted[i] = fmt.Sprintf("%q", e)
	}
	return fmt.Appendf(nil, `# ws's envs, %s, as the set ws. The secrets task adds secrets by -E ws,secrets. env sits
# above the table, which TOML would put it in.
env = ["ws"]
[env_sets]
ws = [%s]
`, strings.Join(envs, ", "), strings.Join(quoted, ", "))
}

// SideEnvs returns the envs that MisercPath names for the set ws, which setup wrote, or none where
// the machine has no miserc yet.
func SideEnvs(m effect.Machine) ([]string, error) {
	text, err := m.ReadFile(MisercPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rc struct {
		Sets map[string][]string `toml:"env_sets"`
	}
	if err := toml.Unmarshal(text, &rc); err != nil {
		return nil, fmt.Errorf("%s: %w", MisercPath, err)
	}
	return rc.Sets["ws"], nil
}

// Config returns the content of ConfigPath: ws's tree is trusted, so that a run of mise by hand
// asks for no mise trust. mise reads a setting of its system config under [settings] alone.
func Config() []byte {
	return fmt.Appendf(nil, `# ws's tree is trusted, so that a run by hand needs no mise trust.
[settings]
trusted_config_paths = [%q]
`, TreeDir)
}

// SystemMise returns the system directory of mise, made if missing, and its two files for the
// envs, root's and readable by all.
func SystemMise(m effect.Machine, envs ...string) []resource.Resource {
	return []resource.Resource{
		file.Dir{M: m, Path: path.Dir(MisercPath), Mode: 0o755, CreateOnly: true},
		file.File{M: m, Path: MisercPath, Content: Miserc(envs...), Mode: 0o644},
		file.File{M: m, Path: ConfigPath, Content: Config(), Mode: 0o644},
	}
}
