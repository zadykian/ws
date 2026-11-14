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
const NameForm = "a lower-case letter, then up to 11 lower-case letters, digits and -, " +
	"other than direct, first and key"

// maxName is the longest a server's name goes, which the ssh alias and the daemon's suffix share.
const maxName = 12

// IsName reports whether s is a server's name. Three words are not: direct, first and key name
// ssh's direct master socket, ws's first-server file and the key reader's link beside the servers'.
func IsName(s string) bool {
	if s == "direct" || s == "first" || s == "key" {
		return false
	}
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

// Pick returns the name of the server to act on: the one that flag names, or the only one where flag
// is empty. It refuses a flag of a bad form or with no file, and no flag where there are none or
// several.
func Pick(m effect.Machine, mac, flag string) (string, error) {
	dir := serversDir(mac)
	if flag != "" && !IsName(flag) {
		return "", fmt.Errorf("--server %s is not a server's name: %s", flag, NameForm)
	}
	names, err := Servers(m, mac)
	if err != nil {
		return "", err
	}
	switch {
	case flag != "":
		for _, n := range names {
			if n == flag {
				return flag, nil
			}
		}
		return "", fmt.Errorf("no server named %s in %s, which holds %s", flag, dir, List(names))
	case len(names) == 0:
		return "", fmt.Errorf("no server in %s: ws client macos wizard adds one (roles/client/macos/README.md)", dir)
	case len(names) > 1:
		return "", fmt.Errorf("several servers in %s, %s: name one by --server", dir, List(names))
	}
	return names[0], nil
}

// List joins names as prose: ws, box and ws, box, lab and ws. No name gives none.
func List(names []string) string {
	switch len(names) {
	case 0:
		return "none"
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
