package drops

import (
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
)

// Service returns wireguard-docker-user.service, which runs the script before Docker starts
// with the interfaces that Docker's containers may not reach the Mac through.
func Service(interfaces []string) ([]byte, error) {
	return resource.Render(files, "files/wireguard-docker-user.service.tmpl", map[string]string{
		"Script": dropsScript, "Interfaces": strings.Join(interfaces, " "),
	})
}
