package kind

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// oursMark starts a file that ws wrote, in a format whose comments start with #.
const oursMark = "# " + file.Mark

// checkGuards refuses group, group_of and ours on a file that is not root's. An account's own file
// has the account's group already, and the account may write it, so its first line proves nothing.
func (f filePart) checkGuards() error {
	root := f.Owner == "" || f.Owner == "root"
	switch {
	case f.Backup && (f.Secret || f.Absent || f.Line != ""):
		return errors.New("backup takes a file that ws writes whole, never a secret, an absent file or a line")
	case f.Group != "" && !root:
		return errors.New("group takes a file of root's")
	case f.Group != "" && f.GroupOf != "":
		return errors.New("a file takes group or group_of, not both")
	case f.Group != "":
		return checkUser("group", f.Group)
	case f.GroupOf != "" && !root:
		return errors.New("group_of takes a file of root's")
	case f.GroupOf != "":
		return checkUser("group_of", f.GroupOf)
	case f.Ours && !root:
		return errors.New("ours takes a file of root's")
	}
	return nil
}

// ours fails the plan where the file is there and lacks ws's mark at its top: a file written by
// hand, which a write or a removal would lose. A dry run stops there too. The error names the path
// alone, as the file may hold a secret.
type ours struct {
	m    effect.Machine
	path string
}

func (o ours) Plan(context.Context) ([]resource.Change, error) {
	old, err := o.m.ReadRootFile(o.path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	case !bytes.HasPrefix(old, []byte(oursMark)):
		return nil, fmt.Errorf("%s lacks the line '%s' at its top, so it was written by hand: move what it holds "+
			"to where ws reads it, remove the file, then run ws again", o.path, oursMark)
	}
	return nil, nil
}
