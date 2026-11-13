package wireguard

import "embed"

//go:embed files/docker-user.sh files/wireguard-docker-user.service.tmpl
var files embed.FS

const (
	// dropsDir holds the script that keeps Docker's containers off the Mac.
	dropsDir    = "/usr/local/lib/ws/wireguard"
	dropsScript = dropsDir + "/docker-user.sh"
	dropsUnit   = "/etc/systemd/system/wireguard-docker-user.service"
)

// DropsScript returns docker-user.sh, which drops what Docker's containers send to the Mac through
// the interfaces given to it, but for replies to the Mac's own connections.
func DropsScript() ([]byte, error) { return files.ReadFile("files/docker-user.sh") }

// MacInterfaces returns the interfaces the drops cover: wg0, and the cable where one is named.
func MacInterfaces(cable string) []string {
	if cable == "" {
		return []string{"wg0"}
	}
	return []string{"wg0", cable}
}
