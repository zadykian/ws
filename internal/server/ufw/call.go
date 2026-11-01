package ufw

import (
	"context"

	"github.com/zadykian/ws/effect"
)

type ufwCall struct {
	m      effect.Machine
	delete bool
	rule   Rule
}

func (c ufwCall) String() string {
	if c.delete {
		return "ufw delete " + c.rule.String()
	}
	return "ufw " + c.rule.String()
}

func (c ufwCall) Apply(ctx context.Context) error {
	args := []string{"--force"}
	if c.delete {
		args = append(args, "delete")
	}
	args = append(args, c.rule.args()...)
	_, err := c.m.RunRoot(ctx, "ufw", append(args, "comment", c.rule.Comment)...)
	return err
}

// normal puts the addresses in the form user.rules writes.
func (r Rule) normal() Rule {
	if r.To == "any" {
		r.To = Any
	}
	if r.From == "any" {
		r.From = Any
	}
	return r
}
