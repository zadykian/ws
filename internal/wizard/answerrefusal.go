package wizard

import (
	"strings"

	"github.com/zadykian/ws/internal/conf"
)

// refusal says why mac.conf can't take the value for the key, or is empty where it can. lan is the
// value of WS_LAN, which WS_LAN_ADDRESS must lie in.
func (q question) refusal(value, lan string) string {
	switch {
	case value == "":
		return q.need
	case strings.Contains(value, "\n"):
		return q.key + " holds a newline, which would add a line of its own"
	}
	read := conf.Lines
	text := line(q.key, value)
	if q.key == "WS_LAN_ADDRESS" {
		read, text = conf.Parse, "WS_LAN="+lan+"\n"+text
	}
	if _, errs := read(text); len(errs) > 0 {
		return errs[0].Msg
	}
	return ""
}
