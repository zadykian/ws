package base

import (
	"context"
	"fmt"
	"regexp"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// localeParts keeps the locale generated and the system's: /etc/locale.gen names it, locale-gen
// makes it, and /etc/locale.conf, where Ubuntu 26.04 keeps the system's locale, sets LANG to it.
// /etc/default/locale links to that file.
func localeParts(m effect.Machine, name string) []host.Resource {
	return []host.Resource{
		generated{M: m, Name: name},
		host.Line{M: m, Path: "/etc/locale.conf", Text: "LANG=" + name, Match: "^LANG=", Mode: 0o644},
	}
}

// generated is a locale that locale-gen has made, as community.general.locale_gen kept it.
type generated struct {
	M    effect.Machine
	Name string
}

type generate struct{ g generated }

func (generate) String() string { return "generate the locale with locale-gen" }

func (c generate) Apply(ctx context.Context) error {
	_, err := c.g.M.RunRoot(ctx, "locale-gen")
	return err
}

// Plan checks the name, enables its line in /etc/locale.gen, and plans locale-gen where that line
// changed or `locale -a` lacks the locale.
func (g generated) Plan(ctx context.Context) ([]host.Change, error) {
	if !localeName.MatchString(g.Name) {
		return nil, fmt.Errorf("base_locale is %q: it takes a glibc locale such as en_US.UTF-8", g.Name)
	}
	entry := host.Line{M: g.M, Path: "/etc/locale.gen", Text: entry(g.Name), Match: `^#?\s*` + regexp.QuoteMeta(g.Name) + `\s`, Mode: 0o644}
	cs, err := entry.Plan(ctx)
	if err != nil {
		return nil, err
	}
	have, err := g.M.Run(ctx, "locale", "-a")
	if err != nil {
		return nil, err
	}
	if len(cs) > 0 || !listed(string(have), g.Name) {
		cs = append(cs, generate{g})
	}
	return cs, nil
}
