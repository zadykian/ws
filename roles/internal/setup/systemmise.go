package setup

import (
	"fmt"
	"path"

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
)

// Miserc returns the content of MisercPath for side: the set ws that names the env of the side's
// files, which mise loads before any config. The secrets task adds its env by -E ws,secrets. The
// key env sits above the table, which TOML would put it in otherwise.
func Miserc(side string) []byte {
	return fmt.Appendf(nil, `# ws's side, %s, as the set ws. The secrets task adds secrets by -E ws,secrets. env sits
# above the table, which TOML would put it in.
env = ["ws"]
[env_sets]
ws = [%q]
`, side, side)
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
// side, root's and readable by all.
func SystemMise(m effect.Machine, side string) []resource.Resource {
	return []resource.Resource{
		file.Dir{M: m, Path: path.Dir(MisercPath), Mode: 0o755, CreateOnly: true},
		file.File{M: m, Path: MisercPath, Content: Miserc(side), Mode: 0o644},
		file.File{M: m, Path: ConfigPath, Content: Config(), Mode: 0o644},
	}
}
