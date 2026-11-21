package fresh_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/devtools"
)

// The servers sit under root's home.
func TestTheServersSitUnderRootsHome(t *testing.T) {
	text := string(configFile(t))
	for name, path := range devtools.ServerPaths("/root") {
		if name != "ansible-language-server" && !strings.Contains(text, `"`+path+`"`) {
			t.Errorf("no command %s", path)
		}
	}
}
