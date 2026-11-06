package fresh

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/devtools"
)

// The servers sit under the user's home, which is /home/ws only in the tests: a literal home in
// the template would point four servers nowhere.
func TestTheServersFollowTheHome(t *testing.T) {
	for _, home := range []string{"/home/ws", "/home/zadykian", "/root"} {
		text := string(renderedAt(t, home))
		if home != "/home/ws" && strings.Contains(text, "/home/ws") {
			t.Errorf("home %s: the render names /home/ws", home)
		}
		for name, path := range devtools.ServerPaths(home) {
			if name != "ansible-language-server" && !strings.Contains(text, `"`+path+`"`) {
				t.Errorf("home %s: no command %s", home, path)
			}
		}
	}
}
