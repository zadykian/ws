// Package locale keeps a locale generated and the system's, as glibc and Ubuntu 26.04 keep them.
package locale

import (
	"context"
	"fmt"
	"regexp"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// Locale is the locale Name, generated and the system's. /etc/locale.gen names it, and locale-gen
// makes it. /etc/locale.conf, where Ubuntu 26.04 keeps the system's locale, sets LANG to it.
// /etc/default/locale links to that file. Hint ends a refusal of the name: how to give another.
type Locale struct {
	M    effect.Machine
	Name string
	Hint string
}

// Plan plans the generated locale, then LANG.
func (l Locale) Plan(ctx context.Context) ([]resource.Change, error) { return l.parts().Plan(ctx) }

func (l Locale) parts() resource.Joined {
	return resource.Joined{
		generated(l),
		file.Line{M: l.M, Path: "/etc/locale.conf", Text: "LANG=" + l.Name, Match: "^LANG=", Mode: 0o644},
	}
}

// generated is a locale that locale-gen has made, as community.general.locale_gen kept it.
type generated Locale

type generate struct{ g generated }

func (generate) String() string { return "generate the locale with locale-gen" }

func (c generate) Apply(ctx context.Context) error {
	_, err := c.g.M.RunRoot(ctx, "locale-gen")
	return err
}

// Plan checks the name against the locales glibc can generate, as the module did, and plans
// nothing where `locale -a` lists the locale. Else it enables the line in /etc/locale.gen and
// plans locale-gen.
func (g generated) Plan(ctx context.Context) ([]resource.Change, error) {
	supported, err := g.M.ReadFile(supportedFile)
	if err != nil {
		return nil, fmt.Errorf("locale %q: reading %s: %w", g.Name, supportedFile, err)
	}
	line, ok := entry(string(supported), g.Name)
	if !ok {
		return nil, fmt.Errorf("the locale %q is none that %s lists. %s", g.Name, supportedFile, g.Hint)
	}
	have, err := g.M.Run(ctx, "locale", "-a")
	if err != nil {
		return nil, err
	}
	if listed(string(have), g.Name) {
		return nil, nil
	}
	cs, err := file.Line{M: g.M, Path: "/etc/locale.gen", Text: line, Match: `^#?\s*` + regexp.QuoteMeta(g.Name) + `\s`, Mode: 0o644}.Plan(ctx)
	if err != nil {
		return nil, err
	}
	return append(cs, generate{g}), nil
}
