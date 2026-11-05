package ufw

import (
	"fmt"
	"strings"
)

// String describes the rule as the ufw command takes it.
func (r Rule) String() string {
	return strings.Join(r.command(false), " ")
}

// Route makes the rule of ufw's route command, which lets what comes in on one interface go out
// by another. Action is allow, deny, reject or limit; a rule that Parse reads from user.rules
// has the same form.
func Route(action, in, out, from, comment string) Rule {
	return Rule{Action: "route:" + action, Proto: "any", ToPort: "any", To: Any, FromPort: "any",
		From: from, Dir: "in", Interface: in + "!out_" + out, Comment: comment}
}

// command is the ufw command's arguments after --force, which a delete alone takes: route's
// keyword and delete's verb, where they apply, then the rule in its full form.
func (r Rule) command(del bool) []string {
	var args []string
	if strings.HasPrefix(r.Action, "route:") {
		args = append(args, "route")
	}
	if del {
		args = append(args, "delete")
	}
	return append(args, r.args()...)
}

// args are the ufw command's arguments from the action on, in its full form.
func (r Rule) args() []string {
	action, _ := strings.CutPrefix(r.Action, "route:")
	args := []string{action, r.Dir}
	if in, out, ok := strings.Cut(r.Interface, "!out_"); ok {
		args = append(args, "on", in, "out", "on", out)
	} else if r.Interface != "" {
		args = append(args, "on", r.Interface)
	}
	args = append(args, "from", addr(r.From))
	if r.FromPort != "any" {
		args = append(args, "port", r.FromPort)
	}
	args = append(args, "to", addr(r.To))
	if r.ToPort != "any" {
		args = append(args, "port", r.ToPort)
	}
	if r.Proto != "any" {
		args = append(args, "proto", r.Proto)
	}
	return args
}

func addr(a string) string {
	if a == Any {
		return "any"
	}
	return a
}

// Valid refuses a rule ufw would not take, or that Parse would not read back as it is.
func (r Rule) Valid() error {
	switch {
	case r.Action == "" || r.Proto == "" || r.To == "" || r.From == "" || r.ToPort == "" || r.FromPort == "":
		return fmt.Errorf("ufw rule %q: a field is empty", r)
	case r.Dir != "in" && r.Dir != "out":
		return fmt.Errorf("ufw rule %q: the direction is %q, not in or out", r, r.Dir)
	case r.Comment == "":
		return fmt.Errorf("ufw rule %q: no comment, so no owner", r)
	}
	return nil
}
