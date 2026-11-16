package values

import "strings"

// Refusal says why values.yml can't take the value for the key, or is empty where it can. lan is the
// value of WS_LAN, which WS_LAN_ADDRESS must lie in.
func (q Question) Refusal(value, lan string) string {
	switch {
	case value == "":
		return q.Need
	case strings.Contains(value, "\n"):
		return q.Key + " holds a newline, which would add a line of its own"
	}
	read := Lines
	text := Text(q.Key, value)
	if q.Key == "WS_LAN_ADDRESS" {
		read, text = Parse, Text("WS_LAN", lan, q.Key, value)
	}
	if _, errs := read(text); len(errs) > 0 {
		return errs[0].Msg
	}
	return ""
}
