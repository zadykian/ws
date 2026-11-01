package ufw

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
)

// ConfFile holds whether ufw is enabled.
const ConfFile = "/etc/ufw/ufw.conf"

// readConf reads a file of KEY=VALUE lines, with the quotes off the values. A missing file is
// empty. The last line of a key counts.
func readConf(m effect.Machine, path string) (map[string]string, error) {
	text, err := m.ReadRootFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	conf := map[string]string{}
	for line := range strings.Lines(string(text)) {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(line, "#") {
			conf[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return conf, nil
}
