package server

import (
	"context"
	"io/fs"
	"path"
)

// unpack builds the playbook in dir.new and moves it over dir, so that a run that stops half way
// leaves no directory that looks unpacked: dir.new holds .unpacked only once it is whole.
func (p Playbook) unpack(ctx context.Context, c unpackChange) error {
	next := c.dir + ".new"
	if c.makeLib {
		if err := p.M.Mkdir(path.Dir(c.dir), 0o755); err != nil {
			return err
		}
	}
	p.Out.Say("unpacking the playbook into %s", c.dir)
	if err := p.M.Pass(ctx, false, "rm", "-rf", next); err != nil {
		return err
	}
	tarballs, err := p.copyTree(next)
	if err != nil {
		return err
	}
	// LC_ALL is set as for the playbook: ansible exits on a LANG that this machine has not generated.
	// The path is set for the install, so that the collections go where the playbook looks. Every
	// dependency is a pinned tarball of the same command.
	args := append([]string{"LC_ALL=C.UTF-8", "ANSIBLE_COLLECTIONS_PATH=" + next + "/collections",
		"ansible-galaxy", "collection", "install", "--offline", "-p", next + "/collections"}, tarballs...)
	if err := p.M.Pass(ctx, false, "env", args...); err != nil {
		return err
	}
	// ansible.cfg and the inventory sit in the playbook's directory, so host_vars beside them is
	// loaded at host_vars' precedence, below -e.
	hostVars := next + "/host_vars"
	if err := p.M.Mkdir(hostVars, 0o755); err != nil {
		return err
	}
	if err := p.M.Pass(ctx, false, "ln", "-s", p.Vars, hostVars+"/localhost.yml"); err != nil {
		return err
	}
	if err := p.M.WriteFile(next+"/.unpacked", []byte(c.digest), 0o644); err != nil {
		return err
	}
	if err := p.M.Pass(ctx, false, "rm", "-rf", c.dir); err != nil {
		return err
	}
	return p.M.Pass(ctx, false, "mv", next, c.dir)
}

// copyTree writes the embedded playbook into dir, directories 0755 and files 0644, and returns the
// paths of the collections' tarballs in it.
func (p Playbook) copyTree(dir string) ([]string, error) {
	var tarballs []string
	err := fs.WalkDir(p.FS, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		to := path.Join(dir, name)
		if d.IsDir() {
			return p.M.Mkdir(to, 0o755)
		}
		data, err := fs.ReadFile(p.FS, name)
		if err != nil {
			return err
		}
		if path.Dir(name) == "collections" && path.Ext(name) == ".gz" {
			tarballs = append(tarballs, to)
		}
		return p.M.WriteFile(to, data, 0o644)
	})
	return tarballs, err
}
