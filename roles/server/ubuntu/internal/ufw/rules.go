// Package ufw reads ufw's rules from its user.rules file and changes them with the ufw command.
// A role owns the rules that carry its comment: it keeps those it wants and removes the others.
package ufw

import (
	"encoding/hex"
	"strings"
)

// Any is the address of a rule that names none, as ufw's user.rules writes it.
const Any = "0.0.0.0/0"

// Rule is one IPv4 rule, in the fields of ufw's user.rules.
type Rule struct {
	Action string // allow, deny, reject or limit
	Proto  string // tcp, udp, or any for both
	// To and From are an address or network; Any is every address. ToPort and FromPort are a
	// number, or "any".
	To, From         string
	ToPort, FromPort string
	// Dir is in or out, and Interface the name the rule matches, or empty for every interface.
	Dir, Interface string
	Comment        string
}

// tupleFields is the number of fields in a rule's tuple. A rule that names applications has two
// more, and is not read.
const tupleFields = 7

// Parse returns the rules of user.rules' text. It skips every line that is no IPv4 rule tuple,
// such as a rule that names applications, and a rule with no comment. A route rule has the action
// route:allow, say, and the interfaces in_IN!out_OUT, which Rule keeps as Interface IN!out_OUT.
func Parse(text string) []Rule {
	var rules []Rule
	for line := range strings.Lines(text) {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != "###" || fields[1] != "tuple" || fields[2] != "###" {
			continue
		}
		comment, ok := commentOf(fields[len(fields)-1])
		if !ok {
			continue
		}
		f := fields[3 : len(fields)-1]
		if len(f) != tupleFields {
			continue
		}
		dir, iface, _ := strings.Cut(f[6], "_")
		rules = append(rules, Rule{Action: f[0], Proto: f[1], ToPort: f[2], To: f[3],
			FromPort: f[4], From: f[5], Dir: dir, Interface: iface, Comment: comment})
	}
	return rules
}

// commentOf decodes the rule's comment, which user.rules writes in hex.
func commentOf(field string) (string, bool) {
	h, ok := strings.CutPrefix(field, "comment=")
	if !ok {
		return "", false
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return "", false
	}
	return string(b), true
}
