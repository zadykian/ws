package conf

import "regexp"

// Question is what the Answers stage asks for one of mac.conf's keys. A Need makes an empty answer
// refused, and is the message that says so.
type Question struct {
	Key, Text, Def, Need string
}

// hint is the bracket of the question: the default, with none where an empty answer serves.
func (q Question) hint() string {
	if q.Need != "" {
		return q.Def
	}
	if q.Def == "" {
		return "none"
	}
	return q.Def + ", or none"
}

// Prompt is the text that the maintainer reads.
func (q Question) Prompt() string {
	if h := q.hint(); h != "" {
		return q.Text + " (" + q.Key + ") [" + h + "]"
	}
	return q.Text + " (" + q.Key + ")"
}

// Reply reads an answer: an empty one takes the default, none leaves the value empty, and a pair of
// quotes goes, as conf does.
func (q Question) Reply(raw string) string {
	switch {
	case raw == "":
		return q.Def
	case raw == "none":
		return ""
	case len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') && raw[0] == raw[len(raw)-1]:
		return raw[1 : len(raw)-1]
	}
	return raw
}

// HasLine reports whether the text of mac.conf has a line for the key.
func HasLine(text, key string) bool {
	return regexp.MustCompile(`(?m)^[ \t]*` + key + `=`).MatchString(text)
}
