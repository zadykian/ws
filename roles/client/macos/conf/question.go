package conf

import (
	"regexp"
	"slices"
	"strings"
)

// targets are WS_TARGET's values, in the order that a choice offers them.
var targets = []string{"lan", "cloud", "mac-vm"}

// Question is what the Answers stage asks for one of mac.conf's keys. A Need makes an empty answer
// refused, and is the message that says so.
type Question struct {
	Key, Text, Def, Need string
}

// Options returns the values of a key that takes one of a few, in the order that a choice offers
// them, and nil for a key that takes a line.
func (q Question) Options() []string {
	if q.Key == "WS_TARGET" {
		return slices.Clone(targets)
	}
	return nil
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

// Prompt is the text that the maintainer reads over a line. It names the options of a key that
// takes one of a few, as a line shows none.
func (q Question) Prompt() string {
	text := q.Text
	if o := q.Options(); len(o) > 1 {
		text += ": " + strings.Join(o[:len(o)-1], ", ") + " or " + o[len(o)-1]
	}
	if h := q.hint(); h != "" {
		return text + " (" + q.Key + ") [" + h + "]"
	}
	return text + " (" + q.Key + ")"
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
