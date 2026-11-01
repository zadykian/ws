package wireguard

import (
	"strings"

	"github.com/zadykian/ws/host"
)

// DropsService returns wireguard-docker-user.service, which runs the script before Docker starts
// with the interfaces that Docker's containers may not reach the Mac through.
func DropsService(interfaces []string) ([]byte, error) {
	return host.Render(files, "files/wireguard-docker-user.service.tmpl", map[string]string{
		"Script": dropsScript, "Interfaces": strings.Join(interfaces, " "),
	})
}
