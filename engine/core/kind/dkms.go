package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// dkmsBinary is where the package dkms puts its command. Before it exists, as in a dry run before
// the package's install, DKMS holds nothing.
const dkmsBinary = "/usr/sbin/dkms"

// dkmsWord is a module's name or a version as DKMS takes them. Each names a directory of /usr/src,
// so it starts with no dot or dash and holds no slash.
var dkmsWord = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

// dkmsSpec is a unit of the kind dkms: a module's source, a tarball pinned by its SHA-256. DKMS
// builds it for the running kernel, and again for each new kernel. The packages of DKMS and the
// kernel's headers are the role's own, a packages unit.
type dkmsSpec struct {
	base
	// Module is DKMS's name of the source, as its dkms.conf gives it.
	Module string `toml:"module"`
	// Version is DKMS's version of the pin. The source unpacks as /usr/src/MODULE-VERSION.
	Version string `toml:"version"`
	URL     string `toml:"url"`
	SHA256  string `toml:"sha256"`
	// Strip drops that many leading parts of each name as the tarball unpacks, as a host's tarball
	// of a commit holds one directory.
	Strip int `toml:"strip"`
	// Conf is a source of the spec's own, written as the tree's dkms.conf with {version} standing
	// for Version, so that the pin is one place. A tree whose own dkms.conf fits needs none.
	Conf string `toml:"conf"`
	// Load is the module that loads once DKMS has built it.
	Load string `toml:"load"`
	// Unload are the modules unloaded first, as another driver of the same device.
	Unload []string `toml:"unload"`
	// Hint follows the line that says the kernel refused Load: what to do, and what holds until then.
	Hint string `toml:"hint"`
}

// check refuses the fields that no plan could take. dir holds the conf.
func (s dkmsSpec) check(dir fs.FS) error {
	switch {
	case !dkmsWord.MatchString(s.Module):
		return fmt.Errorf("module %q is no DKMS module's name", s.Module)
	case !dkmsWord.MatchString(s.Version):
		return fmt.Errorf("version %q is no DKMS version", s.Version)
	case !strings.HasPrefix(s.URL, "https://"):
		return fmt.Errorf("url %q is no https URL", s.URL)
	case len(s.SHA256) != 64 || strings.Trim(s.SHA256, "0123456789abcdef") != "":
		return errors.New("sha256 is not 64 lowercase hex digits")
	case s.Strip < 0:
		return fmt.Errorf("strip %d is below 0", s.Strip)
	case s.Load == "" && (len(s.Unload) > 0 || s.Hint != ""):
		return errors.New("unload and hint go with load")
	case s.Load != "" && !modprobeName.MatchString(s.Load):
		return fmt.Errorf("load %q is no module's name", s.Load)
	}
	for _, name := range s.Unload {
		if !modprobeName.MatchString(name) {
			return fmt.Errorf("unload %q is no module's name", name)
		}
	}
	if s.Conf != "" {
		_, err := fs.Stat(dir, s.Conf)
		return err
	}
	return nil
}

func (s dkmsSpec) parts(e Env, dir fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if err := s.check(dir); err != nil {
		return nil, err
	}
	tree := "/usr/src/" + s.Module + "-" + s.Version
	// The tarball is the same for every architecture, so its pin is under "all".
	source := fetch.Tree{M: e.M, Path: tree, Strip: s.Strip, Source: fetch.PinnedFile{
		M: e.M, Version: s.Version, SHA256: map[string]string{"all": s.SHA256},
		URL: func(string, string) string { return s.URL },
	}}
	return func(context.Context) ([]resource.Resource, error) {
		parts := []resource.Resource{source}
		if s.Conf != "" {
			text, err := fs.ReadFile(dir, s.Conf)
			if err != nil {
				return nil, err
			}
			conf := strings.ReplaceAll(string(text), "{version}", s.Version)
			parts = append(parts, file.File{M: e.M, Path: tree + "/dkms.conf", Content: []byte(conf), Mode: 0o644})
		}
		parts = append(parts, dkmsBuild{m: e.M, out: e.Out, s: s})
		if s.Load != "" {
			parts = append(parts, dkmsModules{m: e.M, out: e.Out, load: s.Load, unload: s.Unload, hint: s.Hint})
		}
		return parts, nil
	}, nil
}

// dkmsBuild is what DKMS holds of a dkms unit's module: the pin added, and built and installed for
// the running kernel. Another version would stand in the pin's way, so it goes with its source.
type dkmsBuild struct {
	m   effect.Machine
	out effect.Out
	s   dkmsSpec
}

// Plan reads DKMS's versions of the module. A version of no plain word's form is left alone with a
// warning, so that no path of its reaches rm.
func (b dkmsBuild) Plan(ctx context.Context) ([]resource.Change, error) {
	kernel, err := kernelRelease(ctx, b.m)
	if err != nil {
		return nil, err
	}
	var held dkmsState
	if _, err := b.m.Stat(dkmsBinary); err == nil {
		out, err := b.m.RunRoot(ctx, "dkms", "status", b.s.Module)
		if err != nil {
			return nil, err
		}
		held = parseDKMS(string(out), b.s.Module, b.s.Version, kernel)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	pin := b.s.Module + "/" + b.s.Version
	var changes []resource.Change
	for _, v := range held.versions {
		if v == b.s.Version {
			continue
		}
		if !dkmsWord.MatchString(v) {
			b.out.Warn("DKMS holds %s's version %q, of no version's form: left alone", b.s.Module, v)
			continue
		}
		old, src := b.s.Module+"/"+v, "/usr/src/"+b.s.Module+"-"+v
		changes = append(changes,
			asRoot{b.m, "remove " + old + " from DKMS", []string{"dkms", "remove", old, "--all"}},
			asRoot{b.m, "remove " + src, []string{"rm", "-rf", "--", src}})
	}
	if !slices.Contains(held.versions, b.s.Version) {
		changes = append(changes, asRoot{b.m, "add " + pin + " to DKMS", []string{"dkms", "add", pin}})
	}
	if !held.installed {
		changes = append(changes, asRoot{b.m, "build and install " + pin + " for " + kernel,
			[]string{"dkms", "install", pin, "-k", kernel}})
	}
	return changes, nil
}

// asRoot is a change that runs a program as root, shown by its text.
type asRoot struct {
	m    effect.Machine
	text string
	args []string
}

func (c asRoot) String() string { return c.text }

func (c asRoot) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, c.args[0], c.args[1:]...)
	return err
}

// dkmsState is what DKMS holds of a module.
type dkmsState struct {
	// versions are the module's versions in DKMS, each once, in the order listed.
	versions []string
	// installed is whether the pinned version is installed for the kernel.
	installed bool
}

// parseDKMS reads the output of `dkms status MODULE` for the pinned version and the kernel. A line
// is `MODULE/VERSION, KERNEL, ARCH: installed`, or `MODULE/VERSION: added`.
func parseDKMS(out, module, version, kernel string) dkmsState {
	line := regexp.MustCompile(`^` + regexp.QuoteMeta(module) + `/([^,:]+)[,:]`)
	prefix := module + "/" + version + ", " + kernel + ", "
	var s dkmsState
	for l := range strings.Lines(out) {
		m := line.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		if !slices.Contains(s.versions, m[1]) {
			s.versions = append(s.versions, m[1])
		}
		if rest, ok := strings.CutPrefix(l, prefix); ok && strings.Contains(rest, ": installed") {
			s.installed = true
		}
	}
	return s
}

// kernelRelease returns the running kernel's release, as uname -r prints it.
func kernelRelease(ctx context.Context, m effect.Machine) (string, error) {
	out, err := m.Run(ctx, "uname", "-r")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
