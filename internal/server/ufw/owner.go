package ufw

import (
	"context"
	"errors"
	"io/fs"
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// UserRules is the file where ufw keeps the rules it was given.
const UserRules = "/etc/ufw/user.rules"

// Rules is a host.Resource of the rules a role owns, those with Comment. Those in Want are added
// where missing, and the others deleted, as ufw keeps a rule until then, a former address's too.
type Rules struct {
	M       effect.Machine
	Comment string
	Want    []Rule
}

// Plan adds the missing rules, then deletes the stale ones.
func (r Rules) Plan(context.Context) ([]host.Change, error) {
	text, err := r.M.ReadRootFile(UserRules)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var have []Rule
	for _, h := range Parse(string(text)) {
		if h.Comment == r.Comment {
			have = append(have, h)
		}
	}
	var want []Rule
	for _, w := range r.Want {
		w.Comment = r.Comment
		want = append(want, w.normal())
	}
	var changes []host.Change
	for _, w := range want {
		if !slices.Contains(have, w) {
			changes = append(changes, ufwCall{r.M, false, w})
		}
	}
	for _, h := range have {
		if !slices.Contains(want, h) {
			changes = append(changes, ufwCall{r.M, true, h})
		}
	}
	return changes, nil
}
