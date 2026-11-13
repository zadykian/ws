package effect

import (
	"os"
	"path/filepath"
)

// confined opens Confine as a root when path lies under it, and gives path inside it. A nil root
// means path is outside, or nothing is confined. The caller closes the root.
func (m OSMachine) confined(path string) (*os.Root, string, error) {
	rel, in := m.inside(path)
	if !in {
		return nil, "", nil
	}
	root, err := os.OpenRoot(m.Confine)
	return root, rel, err
}

// inside gives path inside Confine, and whether it lies there.
func (m OSMachine) inside(path string) (string, bool) {
	if m.Confine == "" {
		return "", false
	}
	rel, err := filepath.Rel(m.Confine, path)
	return rel, err == nil && filepath.IsLocal(rel)
}

// writeAtomicIn is WriteAtomic inside root: a pending file beside rel, with the owner and mode of
// a set on it, renamed over rel.
func writeAtomicIn(root *os.Root, rel string, data []byte, a Attrs) error {
	tmp := filepath.Join(filepath.Dir(rel), "."+filepath.Base(rel)+".ws-new")
	_ = root.Remove(tmp)
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	fail := func(err error) error {
		_ = f.Close()
		_ = root.Remove(tmp)
		return err
	}
	if _, err := f.Write(data); err != nil {
		return fail(err)
	}
	if err := f.Chmod(a.Mode.Perm()); err != nil {
		return fail(err)
	}
	if err := f.Chown(a.UID, a.GID); err != nil {
		return fail(err)
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tmp)
		return err
	}
	if err := root.Rename(tmp, rel); err != nil {
		_ = root.Remove(tmp)
		return err
	}
	return nil
}
