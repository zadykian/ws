package cld

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is cld as a step of resources.
func (c *Cld) Step() resource.Lazy {
	return resource.Lazy{Title: "cld", Parts: c.parts}
}

// parts is one part, which reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (c *Cld) parts(context.Context) ([]resource.Resource, error) {
	return []resource.Resource{cldPart{c}}, nil
}

// cldPart plans cld's install or update and reports its version.
type cldPart struct{ c *Cld }

// Plan returns the changes, each of which applies itself.
func (p cldPart) Plan(ctx context.Context) ([]resource.Change, error) {
	return resource.Planned{Changes: p.c.plan(ctx).Changes}.Plan(ctx)
}

// Report says whether cld is there, with its version.
func (p cldPart) Report(ctx context.Context) ([]step.Item, error) { return p.c.report(ctx) }

// Part is cld as ws client macos cld settles it, for cld's unit: its install where it is missing,
// else its update where releases/latest is newer. It reads the latest as --check does, so that the
// unit's check and its run agree, and a cld at the latest plans nothing. It leaves ssh's file to
// ssh's unit.
func (c *Cld) Part() resource.Resource { return unitPart{c} }

// unitPart plans cld's install or update, as its unit's check and run see it.
type unitPart struct{ c *Cld }

// Plan returns the install where cld is missing, else its update where a newer release is out.
func (p unitPart) Plan(ctx context.Context) ([]resource.Change, error) {
	p.c.Check = true
	cs := []step.Change{cldInstall{p.c}}
	if p.c.has() {
		cs, _ = p.c.update(ctx)
	}
	return resource.Planned{Changes: cs}.Plan(ctx)
}
