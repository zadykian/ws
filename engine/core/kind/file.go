package kind

import (
	"bytes"
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
	// as root writes it there through the account, which refuses a link on the way. A file that
	// names root lies anywhere, and below root's home it refuses a link on the way too.
	Owner string `toml:"owner"`
	// Group gives a file of root's this group of /etc/group, as a service that reads the file runs
	// in it. The group must exist as the unit plans.
	Group    string `toml:"group"`
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
	// Line goes at the top or the end of the owner's file, which keeps its other text (fileline.go).
	Line string `toml:"line"`
	// Has is a command that says whether the file's text, on its standard input, holds Line.
	Has []string `toml:"has"`
	// GroupOf gives a file of root's the primary group of this account, which may then read it.
	GroupOf string `toml:"group_of"`
	// Ours refuses to replace or remove a file there that lacks ws's mark at its top (fileguard.go).
	// The mark follows the comment sign that fileByHand's Mark gives.
	Ours bool `toml:"ours"`
	// Backup keeps the old file beside the new one, as PATH.PID.TIME~, where the content changes.
	Backup bool `toml:"backup"`
	// AbsentWhenEmpty removes the file where its template gives white space alone.
	AbsentWhenEmpty bool `toml:"absent_when_empty"`
	// Link makes the path a link to this target, root's, in place of a file (filelink.go).
	Link string `toml:"link"`
	fileByHand
}

// check refuses a file's fields that no plan could take. dir holds its template or source.
func (f filePart) check(dir fs.FS) error {
	if err := f.checkGuards(); err != nil {
		return err
	}
	switch {
	case !path.IsAbs(f.Path) || path.Clean(f.Path) != f.Path:
		return fmt.Errorf("path %q is not a clean absolute path", f.Path)
	case f.Mode > 0o777:
		return fmt.Errorf("mode %o is not a file's permission bits", f.Mode)
	case f.Link != "":
		return f.checkLink()
	case f.CheckBooted && len(f.Check) == 0:
		return errors.New("check_booted goes with check")
	case f.Line != "" || len(f.Has) > 0:
		return f.checkLine()
	case f.Absent && (f.Template != "" || f.Source != "" || len(f.Check) > 0 || f.Secret || f.Mode != 0 || f.GroupOf != "" || f.Group != ""):
		return errors.New("an absent file takes a path, an owner and ours alone")
	case f.Absent:
		return nil
	case f.AbsentWhenEmpty && f.Template == "":
		return errors.New("absent_when_empty goes with template")
	case (f.Template == "") == (f.Source == ""):
		return errors.New("a file takes a template or a source")
	case f.Template != "":
		if err := checkTemplateName(f.Template); err != nil {
			return err
		}
	}
	_, err := fs.Stat(dir, f.Template+f.Source)
	return err
}

// content is the file's text: its template filled as it plans, or its source unchanged.
func (f filePart) content(ctx context.Context, e Env, dir fs.FS) ([]byte, error) {
	if f.Template == "" {
		return fs.ReadFile(dir, f.Source)
	}
	return render(ctx, e, dir, f.Template)
}

// built is a file's resources: the file itself, which a service watches, the whole list in order,
// and its owner.
type built struct {
	file  resource.Resource
	all   []resource.Resource
	owner account.Account
	// gone is a file that the part removes.
	gone bool
}

// resources makes the file's resources as its unit plans. A missing directory above a file of
// root's is made, root's at 0755, and an existing one left alone. An account's file makes its own
// parents, the account's. systemd reads its units again where a file in /etc/systemd/system
// changes. A link is root's, as are its parents.
func (f filePart) resources(ctx context.Context, e Env, dir fs.FS) (built, error) {
	if f.Link != "" {
		return f.linked(e), nil
	}
	owner, err := ownerOf(ctx, e, f.Owner)
	if err != nil {
		return built{}, err
	}
	// A line is the owner's file below a home, root's too, as a line goes in a file of the home. A
	// file whose spec names root lies in root's home where its path is below it, so a link on the
	// way is refused there too. An account's file names root where the account is root's.
	if owner.Home == "" && (f.Line != "" || f.Owner == "root") {
		var home string
		if home, err = e.System.home(ctx, e.M, "root"); err != nil {
			return built{}, err
		}
		if f.Line != "" || strings.HasPrefix(f.Path, home+"/") {
			owner.Home = home
		}
	}
	rel, below := strings.CutPrefix(f.Path, owner.Home+"/")
	if owner.Home != "" && !below {
		return built{}, fmt.Errorf("%s is not below %s, the home of its owner %s", f.Path, owner.Home, owner.Name)
	}
	content, gone := []byte(nil), f.Absent
	if f.Line == "" && !f.Absent {
		content, err = f.content(ctx, e, dir)
		if err != nil {
			return built{}, err
		}
		gone = f.AbsentWhenEmpty && len(bytes.TrimSpace(content)) == 0
	}
	b := built{gone: gone}
	switch {
	case f.Line != "":
		b.file = f.lines(e, owner, rel)
	case gone && owner.Home != "":
		b.file = owner.Absent(e.M, rel)
	case gone:
		b.file = file.Absent{M: e.M, Path: f.Path}
	default:
		b.file, err = f.written(ctx, e, owner, rel, content)
		if err != nil {
			return built{}, err
		}
	}
	if owner.Home == "" && !gone {
		b.all = parents(e.M, f.Path)
	}
	b.file = f.guarded(e, owner, b.file)
	b.all = append(b.all, b.file)
	if strings.HasPrefix(f.Path, "/etc/systemd/system/") {
		b.all = append(b.all, systemd.DaemonReload{M: e.M, U: e.U, Watch: []resource.Resource{b.file}})
	}
	b.owner = owner
	return b, nil
}

// written is the file with its content, checked before it replaces the live one where it has a
// check that this machine can run.
func (f filePart) written(ctx context.Context, e Env, owner account.Account, rel string, content []byte) (resource.Resource, error) {
	mode := fs.FileMode(f.Mode)
	if mode == 0 {
		mode = 0o644
	}
	w := file.File{M: e.M, Path: f.Path, Content: content, Mode: mode}
	if owner.Home != "" {
		w = owner.File(e.M, rel, content, mode)
	}
	gid, named, err := f.gid(ctx, e.M)
	if err != nil {
		return nil, err
	}
	if named {
		w.GID = gid
	}
	w.Secret = f.Secret
	w.Backup = f.Backup
	if check := f.checker(e); check != nil {
		return file.Validated{File: w, Check: check}, nil
	}
	return w, nil
}

// checker returns the check's command for a new file at a path, or nil where the file has no check
// that this machine can run.
func (f filePart) checker(e Env) func(p string) (string, []string) {
	if len(f.Check) == 0 || f.CheckBooted && !systemd.Booted(e.M) {
		return nil
	}
	return func(p string) (string, []string) {
		args := slices.Clone(f.Check[1:])
		for i := range args {
			args[i] = strings.ReplaceAll(args[i], "{}", p)
		}
		return f.Check[0], args
	}
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
	if err := s.readableOn(e.System); err != nil {
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
	if s.Line != "" && (s.Service != "" || s.UserService != "") {
		return nil, errors.New("a line takes no service, as the file is its owner's")
	}
	if s.Link != "" && (s.Service != "" || s.UserService != "") {
		return nil, errors.New("a link takes no service, as it has no content to change")
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
