package conf

import "path"

// FirstServer is the name of the one server that ws sets up for now.
const FirstServer = "ws"

// ServerPath returns the path of the server called name's file, in a directory beside mac.conf.
func ServerPath(mac, name string) string {
	return path.Join(path.Dir(mac), "servers", name, "server.conf")
}
