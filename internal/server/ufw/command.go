package ufw

import (
	"fmt"
	"strings"
)

// String describes the rule as the ufw command takes it.
func (r Rule) String() string {
	return strings.Join(r.args(), " ")
}

// args are the ufw command's arguments after allow's verb, in its full form.
func (r Rule) args() []string {
	args := []string{r.Action, r.Dir}
	if r.Interface != "" {
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
