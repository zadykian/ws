package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

// InputsFile is the file of a server's role that declares its secret inputs, under mise's
// [bootstrap.secrets].
const InputsFile = "mise.secrets.toml"

// inputName is a name that is at once an input's variable and its secret in fnox. fnox exec
// exports a secret under its own name, and mise reads that variable.
var inputName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Inputs reads the variables of every secret input that the server's roles under roles declare,
// sorted, each once. A role without the file declares none.
func Inputs(m effect.Machine, roles string) ([]string, error) {
	names, err := m.ReadDir(roles)
	if err != nil {
		return nil, fmt.Errorf("cannot read the server's roles in %s: %w", roles, err)
	}
	var all []string
	for _, name := range names {
		dir := path.Join(roles, name)
		info, err := m.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("cannot stat %s: %w", dir, err)
		}
		if !info.Mode.IsDir() {
			continue
		}
		file := path.Join(dir, InputsFile)
		data, err := m.ReadFile(file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", file, err)
		}
		envs, err := parseInputs(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		all = append(all, envs...)
	}
	slices.Sort(all)
	return slices.Compact(all), nil
}

// parseInputs reads the variables of one file's inputs, in mise's two forms: KEY = "ENV" and
// KEY = { env = "ENV", ... }.
func parseInputs(data []byte) ([]string, error) {
	var conf struct {
		Bootstrap struct {
			Secrets map[string]any `toml:"secrets"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}
	var envs []string
	for key, v := range conf.Bootstrap.Secrets {
		env, ok := v.(string)
		if table, isTable := v.(map[string]any); isTable {
			env, ok = table["env"].(string)
		}
		if !ok {
			return nil, fmt.Errorf("the input %s names no variable by env", key)
		}
		if !inputName.MatchString(env) {
			return nil, fmt.Errorf("the input %s reads %q, not a name of capitals, digits and underscores", key, env)
		}
		envs = append(envs, env)
	}
	return envs, nil
}
