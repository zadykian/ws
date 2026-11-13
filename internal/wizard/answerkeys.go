package wizard

import (
	"regexp"
)

// question is what the Answers stage asks for one of mac.conf's keys. A Need makes an empty answer
// refused, and is the message that says so.
type question struct {
	key, text, def, need string
}

// hint is the bracket of the question: the default, with none where an empty answer serves.
func (q question) hint() string {
	if q.need != "" {
		return q.def
	}
	if q.def == "" {
		return "none"
	}
	return q.def + ", or none"
}

// prompt is the text that the maintainer reads.
func (q question) prompt() string {
	if h := q.hint(); h != "" {
		return q.text + " (" + q.key + ") [" + h + "]"
	}
	return q.text + " (" + q.key + ")"
}

// reply reads an answer: an empty one takes the default, none leaves the value empty, and a pair of
// quotes goes, as conf does.
func (q question) reply(raw string) string {
	switch {
	case raw == "":
		return q.def
	case raw == "none":
		return ""
	case len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') && raw[0] == raw[len(raw)-1]:
		return raw[1 : len(raw)-1]
	}
	return raw
}

// hasLine reports whether the text of mac.conf has a line for the key.
func hasLine(text, key string) bool {
	return regexp.MustCompile(`(?m)^[ \t]*` + key + `=`).MatchString(text)
}
