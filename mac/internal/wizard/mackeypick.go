package wizard

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// other returns the .pub in Secretive's PublicKeys other than skip, as Secretive writes one for each
// key, or nothing. It fails where several are there.
func (k *MacKey) other(skip string) (string, error) {
	dir := k.secretive() + "/PublicKeys"
	names, err := k.M.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	got := ""
	for _, name := range names {
		full := dir + "/" + name
		if !strings.HasSuffix(name, ".pub") || strings.HasSuffix(name, "-cert.pub") || strings.HasPrefix(name, ".") ||
			full == skip || !k.isFile(full) {
			continue
		}
		if got != "" {
			return "", fmt.Errorf("%s: Secretive holds several keys, %s and %s: name ws's in mac.conf's WS_IDENTITY, "+
				"and git's in WS_GIT_SIGNING_KEY", macKey, got, full)
		}
		got = full
	}
	return got, nil
}

func (k *MacKey) isFile(path string) bool {
	info, err := k.M.Stat(path)
	return err == nil && info.Mode.IsRegular()
}
