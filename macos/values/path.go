package values

import (
	"strings"

	"github.com/zadykian/ws/internal/etcws"
)

// rootsFile is values.yml's place: root's, in /etc/ws.
const rootsFile = etcws.Dir + "/values.yml"

// Path returns values.yml's path from WS_VALUES, a path from / as a unit's command gets it, else
// /etc/ws.
func Path(getenv func(string) string, pwd string) string {
	switch file := getenv("WS_VALUES"); {
	case file == "":
		return rootsFile
	case strings.HasPrefix(file, "/"):
		return file
	default:
		return pwd + "/" + file
	}
}
