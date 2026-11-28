package values

import (
	"strings"

	"github.com/zadykian/ws/internal/etcws"
)

// Dir returns the directory that holds servers/, from WS_VALUES, a path from / as a unit's command
// gets it, else /etc/ws, root's.
func Dir(getenv func(string) string, pwd string) string {
	switch dir := getenv("WS_VALUES"); {
	case dir == "":
		return etcws.Dir
	case strings.HasPrefix(dir, "/"):
		return dir
	default:
		return pwd + "/" + dir
	}
}
