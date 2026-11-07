package wizard

import (
	"errors"
	"regexp"
)

// userName is the form of a user's name that Ubuntu's adduser takes.
var userName = regexp.MustCompile(`^[a-z][-a-z0-9_]*$`)

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

// userRefusal says why ws_user can't be the answer, or nil. Off a cloud VM it is a user with sudo,
// as the installer leaves one.
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
