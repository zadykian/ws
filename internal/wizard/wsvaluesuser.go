package wizard

import (
	"context"
	"errors"
	"regexp"

	"github.com/zadykian/ws/effect"
)

// userName is the form of a user's name that Ubuntu's adduser takes.
var userName = regexp.MustCompile(`^[a-z][-a-z0-9_]*$`)

// askUser asks ws_user and names it in host_vars. cloud is the target's, as root is ws_user on a
// cloud VM alone.
func (v *WsValues) askUser(ctx context.Context, cloud bool) error {
	user, err := v.P.Ask(ctx, effect.Question{Text: "ws_user, the user that ws is set up for",
		Default: v.offered(cloud), Valid: func(a string) error { return userRefusal(a, cloud) }})
	if err != nil {
		return err
	}
	return v.set(userKey, user)
}

// offered is ws_user's default: the user that ran sudo, else root on a cloud VM, whose login may be
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

// userRefusal says why ws_user can't be the answer, or nil. Off a cloud VM the answer must be a
// user with sudo, as the installer leaves one.
func userRefusal(user string, cloud bool) error {
	switch {
	case user == "":
		return errors.New("name the user that ws is set up for")
	case user == "root" && !cloud:
		return errors.New("root is ws_user on a cloud VM alone: name the user that logs in with sudo")
	case !userName.MatchString(user):
		return errors.New("a user's name: a lowercase letter, then lowercase letters, digits, - and _")
	}
	return nil
}
