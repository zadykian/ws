package values

import "strings"

// homeFile is mac.conf's place under the home.
const homeFile = ".config/ws/mac.conf"

// Path returns mac.conf's path from WS_VALUES, a path from / as ws client macos apply gets it, else
// from the home.
func Path(getenv func(string) string, pwd string) string {
	switch file := getenv("WS_VALUES"); {
	case file == "":
		return getenv("HOME") + "/" + homeFile
	case strings.HasPrefix(file, "/"):
		return file
	default:
		return pwd + "/" + file
	}
}
