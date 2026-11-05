package host

import "fmt"

// SessionEnv is what a command needs to reach the user's own systemd and D-Bus. runuser and sudo
// open no login, so nothing sets these, and systemctl --user fails without them.
func SessionEnv(uid int) []string {
	run := fmt.Sprintf("/run/user/%d", uid)
	return []string{"XDG_RUNTIME_DIR=" + run, "DBUS_SESSION_BUS_ADDRESS=unix:path=" + run + "/bus"}
}
