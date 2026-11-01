package ufw

import (
	"context"
	"strings"

	"github.com/zadykian/ws/effect"
)

type ufwCall struct {
	m      effect.Machine
	delete bool
	rule   Rule
}

func (c ufwCall) String() string {
	return "ufw " + strings.Join(c.rule.command(c.delete), " ")
}

func (c ufwCall) Apply(ctx context.Context) error {
	args := append([]string{"--force"}, c.rule.command(c.delete)...)
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
