package kind

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// oursMark starts a file that ws wrote, in a format whose comments start with #, the default.
const oursMark = "# " + file.Mark

// checkGuards refuses group on a file that is not root's. An account's own file has the account's
// group already, and the account may write it. It checks the fields of a file edited by hand too
// (filebyhand.go).
func (f filePart) checkGuards() error {
	if err := f.checkByHand(); err != nil {
		return err
	}
	root := f.Owner == "" || f.Owner == "root"
	switch {
	case f.Backup && (f.Secret || f.Absent || f.Line != ""):
		return errors.New("backup takes a file that ws writes whole, never a secret, an absent file or a line")
	case f.Group != "" && !root:
		return errors.New("group takes a file of root's")
	case f.Group != "":
		return checkUser("group", f.Group)
	}
	return nil
}

// readableOn refuses, on the Mac, a file of root's that not all may read. The Mac's units read as
// the user, so that a run that changes nothing asks no password, and such a file could not be read.
func (f filePart) readableOn(sys System) error {
	root := f.Owner == "" || f.Owner == "root"
	if sys == MacOS && root && !f.Absent && f.Mode != 0 && f.Mode&0o004 == 0 {
		return fmt.Errorf("mode %o: a file of root's on the Mac is readable by all, so that a read asks no password", f.Mode)
	}
	return nil
}

// ours plans file where the file is not there or starts with mark, ws's own. A file there that
// lacks the mark is a file by hand, which a write or a removal would lose. So the plan fails, a dry
// run's too, and the error names the path alone, as the file may hold a secret. With keep, the
// file is left, and the plan warns and changes nothing.
type ours struct {
	m    effect.Machine
	out  effect.Out
	path string
	// mark is the format's comment sign, then file.Mark.
	mark string
	// home is the account whose home the file lies below, read through no link. Empty for a
	// file of root's outside a home.
	home account.User
	keep bool
	file resource.Resource
}

func (o ours) Plan(ctx context.Context) ([]resource.Change, error) {
	old, err := o.read()
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	case bytes.HasPrefix(old, []byte(o.mark)):
	case o.keep:
		o.out.Warn("%s lacks the line '%s' at its top, so it is left as it is: remove it, "+
			"and the next run writes ws's", o.path, o.mark)
		return nil, nil
	default:
		return nil, fmt.Errorf("%s lacks the line '%s' at its top, so it was written by hand: move what it holds "+
			"to where ws reads it, remove the file, then run ws again", o.path, o.mark)
	}
	return o.file.Plan(ctx)
}

// read reads the file as root, or as the user on the Mac, below a home through no link.
func (o ours) read() ([]byte, error) {
	if o.home.Home == "" {
		return o.m.ReadRootFile(o.path)
	}
	return o.home.ReadHome(o.m, strings.TrimPrefix(o.path, o.home.Home+"/"))
}
