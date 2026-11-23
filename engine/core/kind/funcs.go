package kind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/apt"
)

// pure returns the funcs that read nothing: those of a spec's text, which fills at load, before
// any unit has run.
func pure() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) (int, error) {
			if b == 0 {
				return 0, errors.New("div by zero")
			}
			return a / b, nil
		},
		// fail stops the render with msg, where a template finds the machine or the values unfit.
		"fail": func(msg string) (string, error) { return "", errors.New(msg) },
		// has_prefix is whether a text starts with a prefix.
		"has_prefix": strings.HasPrefix,
		// json quotes a value as Go's encoder does, <, > and & escaped, for a file of JSON.
		"json": func(v any) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		// trim_suffix is s without suffix, as a private key's path is its .pub's without .pub.
		"trim_suffix": strings.TrimSuffix,
	}
}

// facts returns pure's funcs and those that read the machine. A template fills with them as its
// unit plans, when an account or a file that an earlier unit makes is there. A new func is a key
// here.
func facts(ctx context.Context, e Env) template.FuncMap {
	fm := pure()
	// envvar reads a variable of an env file that root alone may read, or the run's input of it.
	fm["envvar"] = func(file, name string) (string, error) {
		if v, ok := e.input(name); ok {
			return v, nil
		}
		return envVar(e.M, file, name)
	}
	// mem_mib is the memory, MemTotal, in whole MiB.
	fm["mem_mib"] = func() (int, error) { return memMiB(e.M) }
	// cpus counts the processors.
	fm["cpus"] = func() (int, error) { return cpus(e.M) }
	// fs_bytes is the size of the filesystem that holds the first of the paths that exists.
	fm["fs_bytes"] = func(paths ...string) (int, error) { return fsBytes(ctx, e.M, paths) }
	// installed_version is dpkg's version of a package that is installed, else "".
	fm["installed_version"] = func(name string) (string, error) {
		states, err := apt.PackageStates(ctx, e.M, name)
		if s := states[name]; err == nil && s.Installed() {
			return s.Version, nil
		}
		return "", err
	}
	// kernel_release is the running kernel's release, as uname -r prints it.
	fm["kernel_release"] = func() (string, error) { return kernelRelease(ctx, e.M) }
	// root_file is a file's text, read as root and trimmed, or "" where the file is missing.
	fm["root_file"] = func(path string) (string, error) { return rootFile(e.M, path) }
	// ssh_pub is a user's SSH public key at a path below the home, else "" (pubkey.go).
	fm["ssh_pub"] = func(user, p string) (string, error) { return sshPub(ctx, e, user, p) }
	return fm
}

// templateEnding ends the name of each Go template that a spec names, as a spec's own does.
const templateEnding = ".tmpl"

// checkTemplateName refuses the name of a template file that lacks templateEnding. Without it, an
// editor and a linter read the file as the format it renders into.
func checkTemplateName(name string) error {
	if !strings.HasSuffix(name, templateEnding) {
		return fmt.Errorf("template %q: the name of a Go template ends in %s", name, templateEnding)
	}
	return nil
}

// render fills the template file name of dir with e.Data and facts' funcs.
func render(ctx context.Context, e Env, dir fs.FS, name string) ([]byte, error) {
	text, err := fs.ReadFile(dir, name)
	if err != nil {
		return nil, err
	}
	return fill(name, text, facts(ctx, e), e.Data)
}

// fill fills text, named name in errors, with data and funcs. A key that data lacks is an error,
// never an empty value.
func fill(name string, text []byte, funcs template.FuncMap, data any) ([]byte, error) {
	t, err := template.New(name).Option("missingkey=error").Funcs(funcs).Parse(string(text))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return out.Bytes(), nil
}

// memMiB reads MemTotal of /proc/meminfo, which the kernel gives in kB.
func memMiB(m effect.Machine) (int, error) {
	data, err := m.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for line := range strings.Lines(string(data)) {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "MemTotal:" {
			kb, err := strconv.Atoi(f[1])
			return kb / 1024, err
		}
	}
	return 0, errors.New("/proc/meminfo has no MemTotal")
}

// cpus counts the lines of /proc/cpuinfo that start a processor, as amd64 and arm64 both write them.
func cpus(m effect.Machine) (int, error) {
	data, err := m.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}
	n := 0
	for line := range strings.Lines(string(data)) {
		if key, _, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(key) == "processor" {
			n++
		}
	}
	if n == 0 {
		return 0, errors.New("/proc/cpuinfo names no processor")
	}
	return n, nil
}

// fsBytes asks stat for the size of the filesystem that holds the first path that exists. A later
// path stands in for one that a unit has yet to make, such as a swap file's directory.
func fsBytes(ctx context.Context, m effect.Machine, paths []string) (int, error) {
	for _, p := range paths {
		if _, err := m.Stat(p); err != nil {
			continue
		}
		out, err := m.Run(ctx, "stat", "-f", "-c", "%S %b", p)
		if err != nil {
			return 0, err
		}
		var size, blocks int
		if _, err := fmt.Sscan(string(out), &size, &blocks); err != nil {
			return 0, fmt.Errorf("stat -f %s: %w", p, err)
		}
		return size * blocks, nil
	}
	return 0, fmt.Errorf("none of %q exists", paths)
}

// rootFile reads path as root, for a file in a directory that root alone may read. Its text,
// trimmed of white space, goes into a template, and a missing file gives "".
func rootFile(m effect.Machine, path string) (string, error) {
	text, err := m.ReadRootFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return strings.TrimSpace(string(text)), err
}

// input returns a variable that e hides, as the run's env sets it, and whether it does. mise writes
// such an input to its file in the run before any unit, but in a plan writes nothing. So a plan
// reads the input here, and sees the change that the run makes. A run whose env lacks it, as one
// without mise's inputs, reads the file as it stands.
func (e Env) input(name string) (string, bool) {
	if e.LookupEnv == nil || !slices.Contains(e.Hidden, name) {
		return "", false
	}
	return e.LookupEnv(name)
}

// envVar returns the value of name in an env file: the last line that sets it, as the shell does,
// with one pair of quotes removed. The file is read, not sourced, so that nothing in it runs as
// root. A missing file or variable gives "".
func envVar(m effect.Machine, file, name string) (string, error) {
	text, err := m.ReadRootFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	set := regexp.MustCompile(`^[[:space:]]*(?:export[[:space:]]+)?` + regexp.QuoteMeta(name) + `=(.*)$`)
	value := ""
	for line := range strings.Lines(string(text)) {
		if s := set.FindStringSubmatch(strings.TrimRight(line, "\n")); s != nil {
			value = strings.TrimSpace(s[1])
		}
	}
	if n := len(value); n >= 2 && (value[0] == '"' || value[0] == '\'') && value[n-1] == value[0] {
		value = value[1 : n-1]
	}
	return value, nil
}
