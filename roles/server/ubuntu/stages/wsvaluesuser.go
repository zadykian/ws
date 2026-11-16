package stages

import (
	"context"
	"errors"
	"regexp"

	"github.com/zadykian/ws/engine/core/effect"
)

// userName is the form of a user's name that Ubuntu's adduser takes.
var userName = regexp.MustCompile(`^[a-z][-a-z0-9_]*$`)

// askUser asks WS_USER and names it in values.yml. cloud is the target's, as root is WS_USER on a
// cloud VM alone.
func (v *WsValues) askUser(ctx context.Context, cloud bool) error {
	user, err := v.P.Ask(ctx, effect.Question{Text: "WS_USER, the user that ws is set up for",
		Default: v.offered(cloud), Valid: func(a string) error { return userRefusal(a, cloud) }})
	if err != nil {
		return err
	}
	return v.set(userKey, user)
}

// offered is WS_USER's default: the user that ran sudo, else root on a cloud VM, whose login may be
// root alone.
func (v *WsValues) offered(cloud bool) string {
	switch {
	case v.User != "" && v.User != "root":
		return v.User
	case cloud:
		return "root"
	}
	return ""
}

// userRefusal says why WS_USER can't be the answer, or nil. Off a cloud VM the answer must be a
// user with sudo, as the installer leaves one.
func userRefusal(user string, cloud bool) error {
	switch {
	case user == "":
		return errors.New("name the user that ws is set up for")
	case user == "root" && !cloud:
		return errors.New("root is WS_USER on a cloud VM alone: name the user that logs in with sudo")
	case !userName.MatchString(user):
		return errors.New("a user's name: a lowercase letter, then lowercase letters, digits, - and _")
	}
	return nil
}
