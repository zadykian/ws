package setup

import (
	"strings"

	"github.com/zadykian/ws/mac/internal/conf"
)

// tunnelConf is tunnel.conf, the values the daemon takes from mac.conf, so that a root daemon reads
// no file the user can write.
func tunnelConf(c conf.Conf) string {
	var b strings.Builder
	b.WriteString("# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf.\n")
	b.WriteString("WS_PUBLIC_KEY=" + c.PublicKey + "\n")
	if c.LAN != "" {
		b.WriteString("WS_LAN=" + c.LAN + "\n")
		b.WriteString("WS_LAN_ADDRESS=" + c.LANAddress + "\n")
	}
	if c.Remote != "" {
		b.WriteString("WS_REMOTE=" + c.Remote + "\n")
	}
	port := c.Port
	if port == "" {
		port = "51820"
	}
	b.WriteString("WS_PORT=" + port + "\n")
	return b.String()
}
