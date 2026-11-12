package host

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/effect"
)

// Account is ws_user as the machine knows it: the name, the home, the uid and the primary gid.
type Account struct {
	User
	UID, GID int
}

// Lookup finds the user's account. root's uid and gid are 0 without a question.
func (u User) Lookup(ctx context.Context, m effect.Machine) (Account, error) {
	a := Account{User: u}
	if u.Name == "root" {
		return a, nil
	}
	ids := [2]int{}
	for i, flag := range []string{"-u", "-g"} {
		out, err := m.Run(ctx, "id", flag, u.Name)
		if err != nil {
			return Account{}, fmt.Errorf("find user %s: %w", u.Name, err)
		}
		if ids[i], err = strconv.Atoi(strings.TrimSpace(string(out))); err != nil {
			return Account{}, fmt.Errorf("id %s %s: %w", flag, u.Name, err)
		}
	}
	a.UID, a.GID = ids[0], ids[1]
	return a, nil
}
