package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// filePart is a file's own fields: a file unit's, and those of a part of a service unit.
type filePart struct {
	Path string `toml:"path"`
	// Mode is the file's permission bits, 0o644 where unset.
	Mode uint32 `toml:"mode"`
	// Owner owns the file, root by default. Another owner's file lies below that account's home,
	// as root writes it there through the account, which refuses a link on the way.
	Owner    string `toml:"owner"`
	Template string `toml:"template"`
	Source   string `toml:"source"`
	// Check is a command that refuses a bad file, {} standing for the new file's path. The new file
	// is written beside the live one and moved over it once the check passes.
	Check []string `toml:"check"`
	// CheckBooted runs Check with booted systemd alone, for a checker that needs what a plain
	// container lacks, such as the kernel's netfilter.
	CheckBooted bool `toml:"check_booted"`
	// Secret keeps the content out of the diff.
	Secret bool `toml:"secret"`
	// Absent removes the file, and takes no content.
	Absent bool `toml:"absent"`
}

// check refuses a file's fields that no plan could take. dir holds its template or source.
func (f filePart) check(dir fs.FS) error {
	switch {
	case !path.IsAbs(f.Path) || path.Clean(f.Path) != f.Path:
		return fmt.Errorf("path %q is not a clean absolute path", f.Path)
	case f.Mode > 0o777:
		return fmt.Errorf("mode %o is not a file's permission bits", f.Mode)
	case f.CheckBooted && len(f.Check) == 0:
		return errors.New("check_booted goes with check")
	case f.Secret && len(f.Check) > 0:
		// A refusal's error carries what the checker prints, which may quote the content.
		return errors.New("a secret file takes no check, as a refusal's error shows the checker's output")
	case f.Absent && (f.Template != "" || f.Source != "" || len(f.Check) > 0 || f.Secret || f.Mode != 0):
		return errors.New("an absent file takes a path and an owner alone")
	case f.Absent:
		return nil
	case (f.Template == "") == (f.Source == ""):
		return errors.New("a file takes a template or a source")
	}
	_, err := fs.Stat(dir, f.Template+f.Source)
	return err
}

// content is the file's text: its template filled as it plans, or its source unchanged.
func (f filePart) content(ctx context.Context, e Env, dir fs.FS) ([]byte, error) {
	if f.Template != "" {
		return render(ctx, e, dir, f.Template)
	}
	return fs.ReadFile(dir, f.Source)
}

// built is a file's resources: the file itself, which a service watches, the whole list in order,
// and its owner.
type built struct {
	file  resource.Resource
	all   []resource.Resource
	owner account.Account
}

// resources makes the file's resources as its unit plans. A missing directory above a file of
// root's is made, root's at 0755, and an existing one left alone. An account's file makes its own
// parents, the account's. systemd reads its units again where a file in /etc/systemd/system
// changes.
func (f filePart) resources(ctx context.Context, e Env, dir fs.FS) (built, error) {
	owner, err := ownerOf(ctx, e, f.Owner)
	if err != nil {
		return built{}, err
	}
	rel, below := strings.CutPrefix(f.Path, owner.Home+"/")
	if owner.Home != "" && !below {
		return built{}, fmt.Errorf("%s is not below %s, the home of its owner %s", f.Path, owner.Home, owner.Name)
	}
	var b built
	switch {
	case f.Absent && owner.Home != "":
		b.file = owner.Absent(e.M, rel)
	case f.Absent:
		b.file = file.Absent{M: e.M, Path: f.Path}
	default:
		b.file, err = f.written(ctx, e, dir, owner, rel)
		if err != nil {
			return built{}, err
		}
	}
	for d := path.Dir(f.Path); owner.Home == "" && !f.Absent && d != "/"; d = path.Dir(d) {
		b.all = append(b.all, file.Dir{M: e.M, Path: d, Mode: 0o755, CreateOnly: true})
	}
	slices.Reverse(b.all)
	b.all = append(b.all, b.file)
	if strings.HasPrefix(f.Path, "/etc/systemd/system/") {
		b.all = append(b.all, systemd.DaemonReload{M: e.M, U: e.U, Watch: []resource.Resource{b.file}})
	}
	b.owner = owner
	return b, nil
}

// written is the file with its content, checked before it replaces the live one where it has a
// check that this machine can run.
func (f filePart) written(ctx context.Context, e Env, dir fs.FS, owner account.Account, rel string) (resource.Resource, error) {
	content, err := f.content(ctx, e, dir)
	if err != nil {
		return nil, err
	}
	mode := fs.FileMode(f.Mode)
	if mode == 0 {
		mode = 0o644
	}
	w := file.File{M: e.M, Path: f.Path, Content: content, Mode: mode}
	if owner.Home != "" {
		w = owner.File(e.M, rel, content, mode)
	}
	w.Secret = f.Secret
	if len(f.Check) == 0 || f.CheckBooted && !systemd.Booted(e.M) {
		return w, nil
	}
	return file.Validated{File: w, Check: func(p string) (string, []string) {
		args := slices.Clone(f.Check[1:])
		for i := range args {
			args[i] = strings.ReplaceAll(args[i], "{}", p)
		}
		return f.Check[0], args
	}}, nil
}

// ownerOf is the account of name, root's for an empty one. Another account's Home is its home, as
// the system's database of accounts gives it.
func ownerOf(ctx context.Context, e Env, name string) (account.Account, error) {
	if name == "" || name == "root" {
		return account.Account{User: account.User{Name: "root"}}, nil
	}
	a, err := account.User{Name: name}.Lookup(ctx, e.M)
	if err != nil {
		return account.Account{}, err
	}
	a.Home, err = e.System.home(ctx, e.M, name)
	return a, err
}

// fileSpec is a unit of the kind file: one file, and the services that read it. Where the file
// changes, Service restarts, or reloads with Reload, and a run that fails first leaves the restart
// owed to the next. A file's restart sits in its own unit, as a service unit applied after it would
// find no change to watch.
type fileSpec struct {
	base
	filePart
	Service string `toml:"service"`
	// Enabled makes Service start at boot.
	Enabled bool `toml:"enabled"`
	Reload  bool `toml:"reload"`
	// UserService is a unit of the owner's own systemd, restarted where the file changes. One with
	// no unit file yet is left to whatever installs it.
	UserService string `toml:"user_service"`
}

func (s fileSpec) parts(e Env, dir fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if err := s.check(dir); err != nil {
		return nil, err
	}
	switch {
	case s.Service != "":
		if err := effect.CheckUnitName("service", s.Service); err != nil {
			return nil, err
		}
	case s.Enabled || s.Reload:
		return nil, errors.New("enabled and reload go with service")
	}
	if s.UserService != "" {
		if s.Owner == "" || s.Owner == "root" {
			return nil, errors.New("user_service takes an owner other than root")
		}
		if err := effect.CheckUnitName("user_service", s.UserService); err != nil {
			return nil, err
		}
	}
	return func(ctx context.Context) ([]resource.Resource, error) {
		b, err := s.resources(ctx, e, dir)
		if err != nil {
			return nil, err
		}
		watch := []resource.Resource{b.file}
		if s.Service != "" {
			b.all = append(b.all, systemd.Unit{M: e.M, U: e.U, Out: e.Out, Name: s.Service, Enabled: s.Enabled,
				Run: systemd.Running, Reload: s.Reload, Watch: watch})
		}
		if s.UserService != "" {
			b.all = append(b.all, systemd.UserUnit{M: e.M, Account: b.owner, Out: e.Out, Name: s.UserService, Watch: watch})
		}
		return b.all, nil
	}, nil
}
