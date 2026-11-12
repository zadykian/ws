package shell

import "github.com/zadykian/ws/host"

// account is ws_user: the commands run as it, and the owner of its files.
type account struct {
	host.User
	UID, GID int
}
