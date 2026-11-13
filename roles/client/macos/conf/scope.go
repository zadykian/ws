package conf

import "slices"

// Scope is the file that a text is: the Mac's mac.conf, or a server's file beside it. Each file
// refuses the other's keys.
type Scope int

const (
	// Both takes any key: the zero value, for a text not yet known to be one file's.
	Both Scope = iota
	// MacFile takes the Mac's keys alone.
	MacFile
	// ServerFile takes every key but the Mac's.
	ServerFile
)

// macKeys are the keys of mac.conf. Every other key is a server's, so a new key is one unless it
// is listed here.
var macKeys = []string{
	"WS_GIT_NAME", "WS_GIT_EMAIL", "WS_GIT_SIGNING_PUB_KEY", "WS_IDENTITY_PUB_KEY", "WS_IDENTITY_AGENT",
}

// refuse returns the message for the key name where the scope is the other file's, else nothing.
func (s Scope) refuse(name string) string {
	mac := slices.Contains(macKeys, name)
	switch {
	case s == MacFile && !mac:
		return name + " is a server's key: it goes in servers/NAME/server.conf, beside mac.conf"
	case s == ServerFile && mac:
		return name + " is the Mac's key: it goes in mac.conf"
	}
	return ""
}
