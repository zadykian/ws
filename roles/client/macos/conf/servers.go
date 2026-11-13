package conf

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"syscall"

	"github.com/zadykian/ws/engine/core/effect"
)

// NameForm says what a server's name looks like, in the words of each refusal of a bad one.
const NameForm = "a lower-case letter, then up to 11 lower-case letters, digits and -"

// maxName is the longest a server's name goes, which the ssh alias and the daemon's suffix share.
const maxName = 12

// IsName reports whether s is a server's name.
func IsName(s string) bool {
	if s == "" || len(s) > maxName || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// serversDir returns the directory of the servers beside mac.conf at mac.
func serversDir(mac string) string {
	return path.Join(path.Dir(mac), "servers")
}

// Servers returns the names of the servers beside mac.conf at mac: the directories of servers/ that
// hold a server.conf, in order. A directory with a file and a bad name is refused.
func Servers(m effect.Machine, mac string) ([]string, error) {
	dir := serversDir(mac)
	entries, err := m.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e, ".") {
			continue
		}
		_, err := m.Stat(ServerPath(mac, e))
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !IsName(e) {
			return nil, fmt.Errorf("%s/%s is not a server's name: %s: rename it", dir, e, NameForm)
		}
		names = append(names, e)
	}
	return names, nil
}
