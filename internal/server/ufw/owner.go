package ufw

import (
	"context"
	"errors"
	"io/fs"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// UserRules is the file where ufw keeps the rules given to it.
const UserRules = "/etc/ufw/user.rules"

// Rules is a host.Resource of the rules a role owns, those with Comment. Those in Want are added
// where missing, and the others deleted, as ufw keeps a rule until then, a former address's too.
type Rules struct {
	M       effect.Machine
	Out     effect.Out
	Comment string
	Want    []Rule
	// Owns tells which of the rules with Comment the role owns. Nil means them all.
	Owns func(Rule) bool
	// NeedsSystemd skips the resource where systemd is not running, as in CI's container, where
	// ufw cannot load its rules.
	NeedsSystemd bool
}

// Plan adds the missing rules, then deletes the stale ones.
func (r Rules) Plan(context.Context) ([]host.Change, error) {
	if r.NeedsSystemd && !host.Booted(r.M) {
		r.Out.Say("ufw rules %q: skipped, systemd is not running", r.Comment)
		return nil, nil
	}
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
		if err := w.Valid(); err != nil {
			return nil, err
		}
		want = append(want, w.normal())
	}
	var changes []host.Change
	for _, w := range want {
		if !slices.Contains(have, w) {
			changes = append(changes, ufwCall{r.M, false, w})
		}
	}
	for _, h := range have {
		if (r.Owns == nil || r.Owns(h)) && !slices.Contains(want, h) {
			changes = append(changes, ufwCall{r.M, true, h})
		}
	}
	return changes, nil
}
