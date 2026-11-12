package shell

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/gitcfg"
)

// signersHeader is the first line of the allowed signers file.
const signersHeader = "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n"

// treeTo returns the directories from below home down to dir, in order. Each is made with mode;
// the ones above dir are left as they are where they exist, as the file module leaves them, and dir
// too where keep is set. A dir outside home is alone, and its parent must exist.
func treeTo(m effect.Machine, a host.Account, dir string, mode fs.FileMode, keep bool) []host.Resource {
	var chain []string
	for d := dir; strings.HasPrefix(d, a.Home+"/"); d = path.Dir(d) {
		chain = append([]string{d}, chain...)
	}
	if len(chain) == 0 {
		chain = []string{dir}
	}
	var dirs []host.Resource
	for _, d := range chain {
		dirs = append(dirs, host.Dir{M: m, Path: d, Mode: mode, UID: a.UID, GID: a.GID, CreateOnly: keep || d != dir})
	}
	return dirs
}

// signers trusts the signing key for the email in git verify-commit. A file set up by hand is kept
// beside it, as ~/.gitconfig is.
type signers struct {
	M     effect.Machine
	A     host.Account
	Key   string
	Path  string
	Email string
}

// Plan reads the public key. A dry run makes no key, so where there is none yet it plans nothing,
// and the key's creation shows the change.
func (s signers) Plan(ctx context.Context) ([]host.Change, error) {
	pub, err := s.M.ReadRootFile(s.Key + ".pub")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	key := gitcfg.PublicKey(string(pub))
	if key == "" {
		return nil, fmt.Errorf("%s.pub holds no key of one line", s.Key)
	}
	f := s.A.File(s.M, below(s.A, s.Path), []byte(signersHeader+s.Email+" "+key+"\n"), 0o644)
	f.Backup = true
	return f.Plan(ctx)
}

// allowedSigners returns the allowed signers file, whose directories the file makes.
func allowedSigners(m effect.Machine, a host.Account, key, file, email string) []host.Resource {
	return []host.Resource{signers{M: m, A: a, Key: key, Path: file, Email: email}}
}
