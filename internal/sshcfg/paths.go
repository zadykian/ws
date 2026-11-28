package sshcfg

import (
	"strings"

	"github.com/zadykian/ws/internal/etcws"
)

// File is where ws keeps ssh's hosts: root's, and read by the Include that heads ~/.ssh/config.
const File = etcws.Dir + "/ssh_config"

// Expand returns path with a leading ~/ replaced by the home.
func Expand(path, home string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return home + "/" + rest
	}
	return path
}
