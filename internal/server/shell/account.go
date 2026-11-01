package shell

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// account is ws_user: the commands run as it, and the owner of its files.
type account struct {
	host.User
	UID, GID int
}

// lookup finds the user's uid and primary gid. root's are 0 without a question.
func lookup(ctx context.Context, m effect.Machine, u host.User) (account, error) {
	a := account{User: u}
	if u.Name == "root" {
		return a, nil
	}
	for i, flag := range []string{"-u", "-g"} {
		out, err := m.Run(ctx, "id", flag, u.Name)
		if err != nil {
			return account{}, fmt.Errorf("find user %s: %w", u.Name, err)
		}
		n, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			return account{}, fmt.Errorf("id %s %s: %w", flag, u.Name, err)
		}
		if i == 0 {
			a.UID = n
		} else {
			a.GID = n
		}
	}
	return a, nil
}

// planAll joins the changes of resources, in order.
func planAll(ctx context.Context, rs ...host.Resource) ([]host.Change, error) {
	var all []host.Change
	for _, r := range rs {
		changes, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		all = append(all, changes...)
	}
	return all, nil
}
