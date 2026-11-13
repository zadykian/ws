package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/roles/internal/sshcfg"
	"github.com/zadykian/ws/roles/server/ubuntu/shell/internal/plan"
)

// signersHeader is the first line of the allowed signers file.
const signersHeader = "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n"

// signers trusts the signing key for the email in git verify-commit. A file set up by hand is kept
// beside it, as ~/.gitconfig is.
type signers struct {
	M     effect.Machine
	A     account.Account
	Key   string
	Path  string
	Email string
}

// Plan reads the public key. A dry run makes no key, so where there is none yet it plans nothing,
// and the key's creation shows the change.
func (s signers) Plan(ctx context.Context) ([]resource.Change, error) {
	pub, err := s.A.ReadHome(s.M, plan.Below(s.A, s.Key)+".pub")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	key, err := sshcfg.ParsePub(string(pub))
	if err != nil {
		return nil, fmt.Errorf("%s.pub %w", s.Key, err)
	}
	f := s.A.File(s.M, plan.Below(s.A, s.Path), []byte(signersHeader+s.Email+" "+key.String()+"\n"), 0o644)
	f.Backup = true
	return f.Plan(ctx)
}

// AllowedSigners returns the allowed signers file, whose directories the file makes.
func AllowedSigners(m effect.Machine, a account.Account, key, file, email string) []resource.Resource {
	return []resource.Resource{signers{M: m, A: a, Key: key, Path: file, Email: email}}
}
